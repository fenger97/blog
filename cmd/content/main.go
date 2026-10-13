// Command content validates and prepares Markdown articles for publication.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	urlpath "path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"blog/internal/content"
	"blog/internal/models"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "content:", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: content check|index|new [--dir content] [options]")
	}
	command := args[0]
	if command != "check" && command != "index" && command != "new" {
		return fmt.Errorf("unknown command %q; use check, index or new", command)
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(output)
	dir := flags.String("dir", "content", "content directory")
	var title, date, tags *string
	if command == "new" {
		title = flags.String("title", "", "article title (required)")
		date = flags.String("date", time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02"), "publication date: YYYY-MM-DD or RFC3339")
		tags = flags.String("tags", "", "comma-separated tags")
	}
	if err := flags.Parse(flagsFirst(args[1:], flags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if command != "new" && flags.NArg() != 0 {
		return fmt.Errorf("%s does not accept positional arguments", command)
	}
	switch command {
	case "check":
		entries, err := loadStrictIndex(filepath.Join(*dir, "index.json"))
		if err != nil {
			return err
		}
		if err := validateDocuments(*dir, entries, true); err != nil {
			return err
		}
		fmt.Fprintf(output, "Checked %d articles: metadata, documents and local links are valid.\n", len(entries))
	case "index":
		entries, err := generateIndex(*dir)
		if err != nil {
			return err
		}
		if err := validateDocuments(*dir, entries, false); err != nil {
			return err
		}
		if err := content.WriteIndex(filepath.Join(*dir, "index.json"), entries); err != nil {
			return err
		}
		fmt.Fprintf(output, "Updated %s with %d articles.\n", filepath.Join(*dir, "index.json"), len(entries))
	case "new":
		if flags.NArg() != 1 {
			return errors.New("usage: content new <slug> -title <title> [-date YYYY-MM-DD] [-tags tag1,tag2] [--dir content]")
		}
		if err := newArticle(*dir, flags.Arg(0), *title, *date, *tags); err != nil {
			return err
		}
		fmt.Fprintf(output, "Created %s. Write the article, then run content index and content check.\n", filepath.Join(*dir, flags.Arg(0)+".md"))
	}
	return nil
}

// Go's flag parser stops at the first positional argument. Allow the common
// `new my-post -title ...` spelling as well as flags before the slug.
func flagsFirst(args []string, flags *flag.FlagSet) []string {
	var options, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			options = append(options, arg)
			name := strings.TrimLeft(arg, "-")
			if !strings.Contains(name, "=") && flags.Lookup(name) != nil && i+1 < len(args) {
				i++
				options = append(options, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return append(options, positional...)
}

func loadStrictIndex(path string) ([]content.Metadata, error) {
	entries, err := content.LoadIndex(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var strict []content.Metadata
	if err := decoder.Decode(&strict); err != nil {
		return nil, fmt.Errorf("decode index metadata: %w", err)
	}
	return entries, nil
}

func validateMetadata(meta content.Metadata) error {
	if err := content.ValidateMetadata(meta); err != nil {
		return err
	}
	if meta.CreatedAt.IsZero() {
		return fmt.Errorf("article %q has no created_at date", meta.Slug)
	}
	if meta.UpdatedAt.IsZero() {
		return fmt.Errorf("article %q has no updated_at date", meta.Slug)
	}
	return nil
}

func markdownFiles(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, file := range files {
		if !file.IsDir() && strings.EqualFold(filepath.Ext(file.Name()), ".md") {
			paths = append(paths, filepath.Join(dir, file.Name()))
		}
	}
	return paths, nil
}

func generateIndex(dir string) ([]content.Metadata, error) {
	previous, err := loadStrictIndex(filepath.Join(dir, "index.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	bySlug := make(map[string]content.Metadata, len(previous))
	for _, meta := range previous {
		bySlug[meta.Slug] = meta
	}
	files, err := markdownFiles(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]content.Metadata, 0, len(files))
	for _, path := range files {
		slug := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		frontmatter, body, err := content.ReadDocument(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if frontmatter.Slug != "" && frontmatter.Slug != slug {
			return nil, fmt.Errorf("%s: frontmatter slug %q must match its filename", path, frontmatter.Slug)
		}
		meta := mergeMetadata(bySlug[slug], frontmatter)
		meta.Slug = slug
		if meta.UpdatedAt.IsZero() {
			meta.UpdatedAt = meta.CreatedAt
		}
		hasFrontmatter, err := hasFrontmatter(path)
		if err != nil {
			return nil, err
		}
		if meta.Excerpt == "" || (hasFrontmatter && frontmatter.Excerpt == "") {
			meta.Excerpt = (&models.Post{Content: body}).GetExcerpt()
		}
		meta.ReadingMinutes = (&models.Post{Content: body}).ReadingMinutes()
		if err := validateMetadata(meta); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		entries = append(entries, meta)
	}
	return entries, nil
}

func hasFrontmatter(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	var prefix [4]byte
	n, err := io.ReadFull(file, prefix[:])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false, err
	}
	return n >= 3 && string(prefix[:3]) == "---" && (n == 3 || prefix[3] == '\n' || prefix[3] == '\r'), nil
}

// Preserve legacy index metadata, with explicitly supplied frontmatter taking
// precedence. Files without frontmatter continue to use their existing index.
func mergeMetadata(previous, frontmatter content.Metadata) content.Metadata {
	if frontmatter.Title != "" {
		previous.Title = frontmatter.Title
	}
	if frontmatter.Excerpt != "" {
		previous.Excerpt = frontmatter.Excerpt
	}
	if !frontmatter.CreatedAt.IsZero() {
		previous.CreatedAt = frontmatter.CreatedAt
	}
	if !frontmatter.UpdatedAt.IsZero() {
		previous.UpdatedAt = frontmatter.UpdatedAt
	}
	if frontmatter.Tags != nil {
		previous.Tags = frontmatter.Tags
	}
	if frontmatter.TravelStart != "" {
		previous.TravelStart, previous.TravelEnd = frontmatter.TravelStart, frontmatter.TravelEnd
	}
	return previous
}

func validateDocuments(dir string, entries []content.Metadata, compareFrontmatter bool) error {
	known := make(map[string]content.Metadata, len(entries))
	for _, meta := range entries {
		if err := validateMetadata(meta); err != nil {
			return err
		}
		if _, exists := known[meta.Slug]; exists {
			return fmt.Errorf("duplicate article slug %q", meta.Slug)
		}
		known[meta.Slug] = meta
	}
	files, err := markdownFiles(dir)
	if err != nil {
		return err
	}
	for _, path := range files {
		slug := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, exists := known[slug]; !exists {
			return fmt.Errorf("%s is not in index.json; run content index", path)
		}
	}
	for _, meta := range entries {
		path := filepath.Join(dir, meta.Slug+".md")
		frontmatter, body, err := content.ReadDocument(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("%s: article body is empty", path)
		}
		if frontmatter.Slug != "" && frontmatter.Slug != meta.Slug {
			return fmt.Errorf("%s: frontmatter slug must match its filename", path)
		}
		if compareFrontmatter {
			expected := mergeMetadata(meta, frontmatter)
			expected.ReadingMinutes = (&models.Post{Content: body}).ReadingMinutes()
			if !metadataEqual(meta, expected) {
				return fmt.Errorf("%s: document metadata or reading time differs from index.json; run content index", path)
			}
		}
		if err := validateLinks(dir, path, body, known); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return nil
}

func metadataEqual(a, b content.Metadata) bool {
	return a.Slug == b.Slug && a.Title == b.Title && a.Excerpt == b.Excerpt &&
		a.CreatedAt.Equal(b.CreatedAt) && a.UpdatedAt.Equal(b.UpdatedAt) &&
		slices.Equal(a.Tags, b.Tags) && a.TravelStart == b.TravelStart && a.TravelEnd == b.TravelEnd &&
		a.ReadingMinutes == b.ReadingMinutes
}

var htmlReferencePattern = regexp.MustCompile("(?is)<(img|a)\\b[^>]*?\\b(?:src|href)\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|([^\\s\"'=<>]+))")

func validateLinks(dir, document, body string, known map[string]content.Metadata) error {
	doc := parser.NewWithExtensions(parser.CommonExtensions).Parse([]byte(body))
	var failures []error
	inspect := func(destination string, image bool) {
		if err := validateReference(dir, document, destination, image, known); err != nil {
			failures = append(failures, err)
		}
	}
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}
		switch n := node.(type) {
		case *ast.Image:
			inspect(string(n.Destination), true)
		case *ast.Link:
			inspect(string(n.Destination), false)
		case *ast.HTMLBlock, *ast.HTMLSpan:
			for _, match := range htmlReferencePattern.FindAllSubmatch(node.AsLeaf().Literal, -1) {
				for _, value := range match[2:] {
					if len(value) > 0 {
						inspect(string(value), strings.EqualFold(string(match[1]), "img"))
						break
					}
				}
			}
		}
		return ast.GoToNext
	})
	return errors.Join(failures...)
}

func validateReference(dir, document, destination string, image bool, known map[string]content.Metadata) error {
	reference, err := url.Parse(destination)
	if err != nil {
		return fmt.Errorf("invalid link %q: %w", destination, err)
	}
	if reference.Path == "" {
		return nil
	}
	if reference.Host != "" {
		site := os.Getenv("SITE_URL")
		if site == "" {
			site = "https://fenger.xin"
		}
		canonical, err := url.Parse(site)
		if err != nil || !strings.EqualFold(reference.Host, canonical.Host) {
			return nil // External links are intentionally checked without networking.
		}
	} else if reference.Scheme != "" {
		return nil
	}
	path := reference.Path
	if image {
		path = urlpath.Clean(path)
		switch strings.ToLower(filepath.Ext(path)) {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif":
		default:
			return fmt.Errorf("local image %q must use a supported raster format: jpg, jpeg, png, gif, webp or avif", destination)
		}
		var local string
		switch {
		case strings.HasPrefix(path, "/content/assets/"):
			local = filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(path, "/content/")))
		case strings.HasPrefix(path, "/static/"):
			local = filepath.Join(filepath.Dir(dir), filepath.FromSlash(strings.TrimPrefix(path, "/")))
		case strings.HasPrefix(path, "assets/"):
			local = filepath.Join(filepath.Dir(document), filepath.FromSlash(path))
		default:
			return fmt.Errorf("local image %q must use a supported location: assets/, /content/assets/ or /static/", destination)
		}
		info, err := os.Stat(local)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("local image %q does not exist as a readable file", destination)
		}
		root, err := filepath.Abs(filepath.Dir(dir))
		if err != nil {
			return err
		}
		resolved, err := filepath.EvalSymlinks(local)
		if err != nil {
			return err
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("local image %q is outside the project directory", destination)
		}
		file, err := os.Open(local)
		if err != nil {
			return fmt.Errorf("local image %q is not readable: %w", destination, err)
		}
		return file.Close()
	}
	var slug string
	switch {
	case strings.HasPrefix(path, "/post/"):
		slug = strings.TrimPrefix(path, "/post/")
	case !strings.HasPrefix(path, "/") && strings.EqualFold(filepath.Ext(path), ".md"):
		slug = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if strings.TrimPrefix(path, "./") != slug+".md" {
			return fmt.Errorf("article link %q must point to a Markdown file in the content directory", destination)
		}
	default:
		return nil
	}
	if _, exists := known[slug]; !exists {
		return fmt.Errorf("article link %q has no indexed article", destination)
	}
	return nil
}

func newArticle(dir, slug, title, date, rawTags string) error {
	created, err := time.ParseInLocation("2006-01-02", date, time.FixedZone("Asia/Shanghai", 8*60*60))
	if err != nil {
		created, err = time.Parse(time.RFC3339, date)
		if err != nil {
			return fmt.Errorf("invalid publication date %q; use YYYY-MM-DD or RFC3339", date)
		}
	}
	meta := content.Metadata{Slug: slug, Title: strings.TrimSpace(title), CreatedAt: created, UpdatedAt: created}
	if rawTags != "" {
		for _, tag := range strings.Split(rawTags, ",") {
			meta.Tags = append(meta.Tags, strings.TrimSpace(tag))
		}
	}
	if err := validateMetadata(meta); err != nil {
		return err
	}
	frontmatter, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, slug+".md")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("create article without overwriting: %w", err)
	}
	_, writeErr := fmt.Fprintf(file, "---\n%s---\n\n在这里写下你的正文。\n", frontmatter)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(path)
		return errors.Join(writeErr, closeErr)
	}
	return nil
}
