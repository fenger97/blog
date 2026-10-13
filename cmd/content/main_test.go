package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"blog/internal/content"
)

func TestNewArticleCreatesPortableFrontmatterAndNeverOverwrites(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "content")
	args := []string{"new", "a-trip", "-title", "海风与落日", "-date", "2025-10-27", "-tags", "旅行, 济州岛", "--dir", dir}
	var output bytes.Buffer
	if err := run(args, &output); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "a-trip.md")
	meta, body, err := content.ReadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "海风与落日" || meta.Slug != "a-trip" || len(meta.Tags) != 2 || strings.TrimSpace(body) == "" {
		t.Fatalf("new document is incomplete: %+v %q", meta, body)
	}
	if meta.CreatedAt.Format(time.RFC3339) != "2025-10-27T00:00:00+08:00" || !meta.UpdatedAt.Equal(meta.CreatedAt) {
		t.Fatalf("publication date was not preserved: %+v", meta)
	}
	original := readFile(t, path)
	if err := run(args, &output); err == nil {
		t.Fatal("creating an existing article must fail")
	}
	if got := readFile(t, path); !bytes.Equal(original, got) {
		t.Fatal("existing document changed")
	}
}

func TestNewArticleRejectsInvalidInputBeforeCreatingFiles(t *testing.T) {
	for _, args := range [][]string{
		{"new", "../outside", "-title", "Title"},
		{"new", "valid-slug", "-title", "   "},
		{"new", "valid-slug", "-title", "Title", "-date", "2025-02-30"},
		{"new", "valid-slug", "-title", "Title", "-tags", "旅行,旅行"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "content")
			if err := run(append(args, "--dir", dir), &bytes.Buffer{}); err == nil {
				t.Fatal("invalid input was accepted")
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatalf("invalid command created content directory: %v", err)
			}
		})
	}
}

func TestIndexMigratesLegacyAndFrontmatterArticlesAndIsRepeatable(t *testing.T) {
	dir := newContentDirectory(t)
	legacy := metadata("old", "2020-01-02")
	legacy.Title, legacy.Excerpt = "保留原来的标题", "人工编写的摘要"
	legacy.Tags = []string{"旧文章"}
	writeIndex(t, dir, []content.Metadata{legacy})
	writeFile(t, filepath.Join(dir, "old.md"), "这是一篇没有 frontmatter 的旧文章。\n")
	writeFile(t, filepath.Join(dir, "new.md"), `---
title: 新的文章
created_at: 2025-10-27
tags: [旅行]
---

新的正文里有足够的文字，用来自动生成摘要。
`)
	if err := run([]string{"index", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var entries []content.Metadata
	original := readFile(t, filepath.Join(dir, "index.json"))
	if err := json.Unmarshal(original, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Slug != "new" || entries[1].Slug != "old" {
		t.Fatalf("articles were not sorted newest first: %+v", entries)
	}
	if entries[1].Title != legacy.Title || entries[1].Excerpt != legacy.Excerpt || entries[1].Tags[0] != "旧文章" {
		t.Fatalf("legacy metadata was lost: %+v", entries[1])
	}
	if entries[0].Excerpt == "" || strings.Contains(entries[0].Excerpt, "created_at") || entries[0].UpdatedAt.IsZero() {
		t.Fatalf("generated metadata is incomplete: %+v", entries[0])
	}
	if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"index", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, readFile(t, filepath.Join(dir, "index.json"))) {
		t.Fatal("rebuilding unchanged documents changed index.json")
	}
}

func TestFailedIndexGenerationPreservesPublishedIndex(t *testing.T) {
	dir := newContentDirectory(t)
	meta := metadata("old", "2020-01-02")
	writeIndex(t, dir, []content.Metadata{meta})
	writeFile(t, filepath.Join(dir, "old.md"), "![missing](assets/missing.jpg)\n")
	path := filepath.Join(dir, "index.json")
	original := readFile(t, path)
	if err := run([]string{"index", "--dir", dir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "local image") {
		t.Fatalf("expected a missing image error, got %v", err)
	}
	if !bytes.Equal(original, readFile(t, path)) {
		t.Fatal("validation failure replaced the existing index")
	}
}

func TestIndexRefreshesAutomaticExcerptsAfterBodyEdits(t *testing.T) {
	dir := newContentDirectory(t)
	path := filepath.Join(dir, "article.md")
	frontmatter := "---\ntitle: 文章标题\ncreated_at: 2025-10-27\nexcerpt: \"\"\n---\n"
	writeFile(t, path, frontmatter+"原来的正文摘要。\n")
	if err := run([]string{"index", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	first, err := content.LoadIndex(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, frontmatter+"修改后的正文描述了另一段旅程。\n")
	if err := run([]string{"index", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	second, err := content.LoadIndex(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if second[0].Excerpt == first[0].Excerpt || !strings.Contains(second[0].Excerpt, "修改后") {
		t.Fatalf("automatic excerpt did not follow the body edit: %q -> %q", first[0].Excerpt, second[0].Excerpt)
	}
	writeFile(t, path, frontmatter+strings.Repeat("更多的文字构成了较长的正文。", 200))
	if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "reading time") {
		t.Fatalf("expected out-of-date reading time error, got %v", err)
	}
	if err := run([]string{"index", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	third, err := content.LoadIndex(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if third[0].ReadingMinutes <= second[0].ReadingMinutes {
		t.Fatalf("reading time did not follow the body edit: %d -> %d", second[0].ReadingMinutes, third[0].ReadingMinutes)
	}
}

func TestCheckRequiresIndexToMatchFrontmatter(t *testing.T) {
	dir := newContentDirectory(t)
	writeIndex(t, dir, []content.Metadata{metadata("article", "2025-10-27")})
	writeFile(t, filepath.Join(dir, "article.md"), "---\ntitle: Changed title\n---\n正文\n")
	if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "run content index") {
		t.Fatalf("expected stale metadata error, got %v", err)
	}
	if err := run([]string{"index", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsBrokenPublicationInputs(t *testing.T) {
	tests := []struct {
		name   string
		index  string
		body   string
		needle string
	}{
		{"unknown field", `[{"slug":"article","title":"A","created_at":"2025-10-27T00:00:00Z","updated_at":"2025-10-27T00:00:00Z","titel":"typo"}]`, "Text", "unknown field"},
		{"missing date", `[{"slug":"article","title":"A"}]`, "Text", "created_at"},
		{"duplicate slug", `[{"slug":"article","title":"A"},{"slug":"article","title":"B"}]`, "Text", "duplicate"},
		{"invalid travel interval", `[{"slug":"article","title":"A","travel_start":"2025-10-27","travel_end":"2025-10-23"}]`, "Text", "travel_end"},
		{"invalid frontmatter", "", "---\ntitle: [not, a, title]\n---\nText", "frontmatter"},
		{"empty document", "", " \n ", "empty"},
		{"missing document", "", "", "no such file"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := newContentDirectory(t)
			if test.index == "" {
				writeIndex(t, dir, []content.Metadata{metadata("article", "2025-10-27")})
			} else {
				writeFile(t, filepath.Join(dir, "index.json"), test.index)
			}
			if test.body != "" {
				writeFile(t, filepath.Join(dir, "article.md"), test.body)
			}
			if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), test.needle) {
				t.Fatalf("expected error containing %q, got %v", test.needle, err)
			}
		})
	}
}

func TestCheckLocalImagesAndArticleLinksWithoutNetworkRequests(t *testing.T) {
	dir := newContentDirectory(t)
	writeIndex(t, dir, []content.Metadata{metadata("article", "2025-10-27"), metadata("other", "2025-10-26")})
	writeFile(t, filepath.Join(dir, "other.md"), "另一篇文章。")
	writeFile(t, filepath.Join(dir, "assets", "photo.jpg"), "image fixture")
	writeFile(t, filepath.Join(filepath.Dir(dir), "static", "img", "avatar.png"), "image fixture")
	valid := `![local](assets/photo.jpg)
![content route](/content/assets/photo.jpg)
![static](/static/img/avatar.png)
![remote](https://unreachable.invalid/photo.jpg)
[another](other.md#section)
[another route](/post/other)
[remote link](https://unreachable.invalid/post/does-not-exist)
![reference][image]

[image]: assets/photo.jpg

` + "```md\n![example](assets/not-real.jpg)\n[example](/post/not-real)\n```\n" + "`![inline example](assets/not-real.jpg)`\n"
	path := filepath.Join(dir, "article.md")
	writeFile(t, path, valid)
	if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, broken := range []string{
		"![broken](assets/missing.jpg)",
		"[broken](/post/missing)",
		"[broken](missing.md)",
		`<img src="assets/missing.jpg">`,
		"[broken](https://fenger.xin/post/missing)",
	} {
		writeFile(t, path, valid+"\n\n"+broken)
		if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted invalid local reference %q", broken)
		}
	}
}

func TestCheckDetectsForgottenIndexGeneration(t *testing.T) {
	dir := newContentDirectory(t)
	writeIndex(t, dir, []content.Metadata{})
	writeFile(t, filepath.Join(dir, "forgotten.md"), "新的正文。")
	if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "not in index.json") {
		t.Fatalf("expected unindexed article error, got %v", err)
	}
}

func TestCheckRejectsExistingImagesThatRuntimeCannotServe(t *testing.T) {
	dir := newContentDirectory(t)
	writeIndex(t, dir, []content.Metadata{metadata("article", "2025-10-27")})
	writeFile(t, filepath.Join(dir, "assets", "image.svg"), "<svg></svg>")
	writeFile(t, filepath.Join(dir, "photo.jpg"), "image fixture")
	for _, destination := range []string{"assets/image.svg", "photo.jpg", "/content/photo.jpg"} {
		writeFile(t, filepath.Join(dir, "article.md"), "正文。\n\n![unsupported]("+destination+")")
		if err := run([]string{"check", "--dir", dir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "supported") {
			t.Fatalf("existing but unservable image %q was accepted: %v", destination, err)
		}
	}
}

func newContentDirectory(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "content")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func metadata(slug, date string) content.Metadata {
	created, _ := time.Parse("2006-01-02", date)
	return content.Metadata{Slug: slug, Title: "Title", CreatedAt: created, UpdatedAt: created, ReadingMinutes: 1}
}

func writeIndex(t *testing.T, dir string, entries []content.Metadata) {
	t.Helper()
	if err := content.WriteIndex(filepath.Join(dir, "index.json"), entries); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
