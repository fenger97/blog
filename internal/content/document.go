package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"blog/internal/models"
	"gopkg.in/yaml.v3"
)

// Metadata is the portable index and frontmatter representation of an article.
type Metadata struct {
	Slug           string    `json:"slug" yaml:"slug"`
	Title          string    `json:"title" yaml:"title"`
	Excerpt        string    `json:"excerpt" yaml:"excerpt"`
	CreatedAt      time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" yaml:"updated_at"`
	Tags           []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	TravelStart    string    `json:"travel_start,omitempty" yaml:"travel_start,omitempty"`
	TravelEnd      string    `json:"travel_end,omitempty" yaml:"travel_end,omitempty"`
	ReadingMinutes int       `json:"reading_minutes,omitempty" yaml:"-"`
}

// ReadDocument reads Markdown, separating optional YAML frontmatter from its body.
// Legacy documents without frontmatter are returned unchanged.
func ReadDocument(path string) (Metadata, string, error) {
	data, err := readBoundedFile(path, maxArticleBytes)
	if err != nil {
		return Metadata{}, "", err
	}
	return parseDocument(data)
}

func parseDocument(data []byte) (Metadata, string, error) {
	if err := validateArticleText(data); err != nil {
		return Metadata{}, "", err
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSuffix(lines[0], "\r") != "---" {
		return Metadata{}, text, nil
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if line := strings.TrimSuffix(lines[i], "\r"); line == "---" || line == "..." {
			end = i
			break
		}
	}
	if end < 0 {
		return Metadata{}, "", fmt.Errorf("unterminated YAML frontmatter")
	}
	var raw struct {
		Slug        string   `yaml:"slug"`
		Title       string   `yaml:"title"`
		Excerpt     string   `yaml:"excerpt"`
		CreatedAt   string   `yaml:"created_at"`
		UpdatedAt   string   `yaml:"updated_at"`
		Tags        []string `yaml:"tags"`
		TravelStart string   `yaml:"travel_start"`
		TravelEnd   string   `yaml:"travel_end"`
	}
	decoder := yaml.NewDecoder(strings.NewReader(strings.Join(lines[1:end], "\n")))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return Metadata{}, "", fmt.Errorf("decode YAML frontmatter: %w", err)
	}
	created, err := parseDate(raw.CreatedAt)
	if err != nil {
		return Metadata{}, "", fmt.Errorf("created_at: %w", err)
	}
	updated, err := parseDate(raw.UpdatedAt)
	if err != nil {
		return Metadata{}, "", fmt.Errorf("updated_at: %w", err)
	}
	meta := Metadata{
		Slug: raw.Slug, Title: raw.Title, Excerpt: raw.Excerpt,
		CreatedAt: created, UpdatedAt: updated, Tags: raw.Tags,
		TravelStart: raw.TravelStart, TravelEnd: raw.TravelEnd,
	}
	if err := validateOptionalMetadata(meta); err != nil {
		return Metadata{}, "", err
	}
	return meta, strings.Join(lines[end+1:], "\n"), nil
}

func validateArticleText(data []byte) error {
	if !utf8.Valid(data) || bytes.ContainsRune(data, '\x00') {
		return fmt.Errorf("article must be valid UTF-8 text")
	}
	return nil
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q (use YYYY-MM-DD or RFC3339)", value)
}

// ValidateMetadata checks the fields required by the index without requiring
// legacy articles to supply dates.
func ValidateMetadata(meta Metadata) error {
	if !models.IsValidSlug(meta.Slug) {
		return fmt.Errorf("invalid article slug %q", meta.Slug)
	}
	if strings.TrimSpace(meta.Title) == "" {
		return fmt.Errorf("article %q has no title", meta.Slug)
	}
	return validateOptionalMetadata(meta)
}

func validateOptionalMetadata(meta Metadata) error {
	if meta.ReadingMinutes < 0 {
		return fmt.Errorf("reading_minutes cannot be negative")
	}
	if meta.Slug != "" && !models.IsValidSlug(meta.Slug) {
		return fmt.Errorf("invalid article slug %q", meta.Slug)
	}
	if !meta.CreatedAt.IsZero() && !meta.UpdatedAt.IsZero() && meta.UpdatedAt.Before(meta.CreatedAt) {
		return fmt.Errorf("article %q updated_at is earlier than created_at", meta.Slug)
	}
	for _, value := range []string{meta.TravelStart, meta.TravelEnd} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return fmt.Errorf("invalid travel date %q (use YYYY-MM-DD)", value)
			}
		}
	}
	if (meta.TravelStart == "") != (meta.TravelEnd == "") {
		return fmt.Errorf("travel_start and travel_end must be provided together")
	}
	if meta.TravelStart != "" && meta.TravelEnd < meta.TravelStart {
		return fmt.Errorf("travel_end is earlier than travel_start")
	}
	seenTags := make(map[string]bool)
	for _, tag := range meta.Tags {
		if strings.TrimSpace(tag) != tag || tag == "" || len([]rune(tag)) > 40 {
			return fmt.Errorf("invalid tag %q", tag)
		}
		if seenTags[tag] {
			return fmt.Errorf("duplicate tag %q", tag)
		}
		seenTags[tag] = true
	}
	return nil
}

func decodeIndex(data []byte) ([]Metadata, error) {
	var entries []Metadata
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode content index: %w", err)
	}
	if entries == nil {
		return nil, fmt.Errorf("content index must be a JSON array")
	}
	if err := validateEntries(entries); err != nil {
		return nil, err
	}
	sortEntries(entries)
	return entries, nil
}

func validateEntries(entries []Metadata) error {
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if err := ValidateMetadata(entry); err != nil {
			return err
		}
		if seen[entry.Slug] {
			return fmt.Errorf("duplicate content index entry %q", entry.Slug)
		}
		seen[entry.Slug] = true
	}
	return nil
}

func sortEntries(entries []Metadata) {
	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
			return entries[i].CreatedAt.After(entries[j].CreatedAt)
		}
		return entries[i].Slug < entries[j].Slug
	})
}

// LoadIndex reads and validates a local JSON index.
func LoadIndex(path string) ([]Metadata, error) {
	data, err := readBoundedFile(path, maxIndexBytes)
	if err != nil {
		return nil, err
	}
	return decodeIndex(data)
}

func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("content file exceeds %d bytes", maxBytes)
	}
	return data, nil
}

// WriteIndex atomically replaces an index only after validation succeeds.
func WriteIndex(path string, entries []Metadata) error {
	if err := validateEntries(entries); err != nil {
		return err
	}
	entries = cloneEntries(entries)
	if entries == nil {
		entries = []Metadata{}
	}
	sortEntries(entries)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if len(data)+1 > maxIndexBytes {
		return fmt.Errorf("index exceeds %d bytes", maxIndexBytes)
	}
	return atomicWrite(path, append(data, '\n'), 0644)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".content-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err = temp.Chmod(mode); err == nil {
		_, err = temp.Write(data)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(temp.Name(), path)
}

func (entry Metadata) post() *models.Post {
	return &models.Post{
		ID: entry.Slug, Title: entry.Title, Excerpt: entry.Excerpt,
		CreatedAt: entry.CreatedAt, UpdatedAt: entry.UpdatedAt,
		Tags:        append([]string(nil), entry.Tags...),
		TravelStart: entry.TravelStart, TravelEnd: entry.TravelEnd,
		EstimatedMinutes: entry.ReadingMinutes,
	}
}

func cloneEntries(entries []Metadata) []Metadata {
	if entries == nil {
		return nil
	}
	cloned := make([]Metadata, len(entries))
	copy(cloned, entries)
	for i := range cloned {
		cloned[i].Tags = append([]string(nil), entries[i].Tags...)
	}
	return cloned
}

// Missing optional index fields can be supplied by a document's frontmatter.
func mergeMetadata(index, document Metadata) Metadata {
	if index.Excerpt == "" {
		index.Excerpt = document.Excerpt
	}
	if index.CreatedAt.IsZero() {
		index.CreatedAt = document.CreatedAt
	}
	if index.UpdatedAt.IsZero() {
		index.UpdatedAt = document.UpdatedAt
	}
	if len(index.Tags) == 0 {
		index.Tags = document.Tags
	}
	if index.TravelStart == "" {
		index.TravelStart = document.TravelStart
		index.TravelEnd = document.TravelEnd
	}
	return index
}

// Local editing treats explicit frontmatter as the current source of truth;
// the index supplies legacy fields until a new index has been generated.
func previewMetadata(index, document Metadata) Metadata {
	if document.Title != "" {
		index.Title = document.Title
	}
	if document.Excerpt != "" {
		index.Excerpt = document.Excerpt
	}
	if !document.CreatedAt.IsZero() {
		index.CreatedAt = document.CreatedAt
	}
	if !document.UpdatedAt.IsZero() {
		index.UpdatedAt = document.UpdatedAt
	}
	if document.Tags != nil {
		index.Tags = document.Tags
	}
	if document.TravelStart != "" {
		index.TravelStart = document.TravelStart
		index.TravelEnd = document.TravelEnd
	}
	return index
}
