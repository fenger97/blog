package content

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReadDocumentFrontmatter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trip.md")
	body := "# Trip\n\n你好，海风。\n"
	if err := os.WriteFile(path, []byte("---\nslug: trip\ntitle: Trip\ncreated_at: 2025-10-27\nupdated_at: '2026-10-10T00:00:00+08:00'\ntags: [旅行, 海边]\ntravel_start: 2025-10-23\ntravel_end: 2025-10-27\n---\n"+body), 0600); err != nil {
		t.Fatal(err)
	}
	meta, got, err := ReadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != body || meta.Slug != "trip" || meta.CreatedAt.Format("2006-01-02") != "2025-10-27" || meta.TravelStart != "2025-10-23" || !reflect.DeepEqual(meta.Tags, []string{"旅行", "海边"}) {
		t.Fatalf("unexpected document: %#v, %q", meta, got)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	meta, got, err = ReadDocument(path)
	if err != nil || got != body || !reflect.DeepEqual(meta, Metadata{}) {
		t.Fatalf("legacy: %#v %q %v", meta, got, err)
	}
}

func TestReadDocumentRejectsMalformedMetadata(t *testing.T) {
	for _, frontmatter := range []string{
		"created_at: yesterday", "travel_start: 2025-10-27\ntravel_end: 2025-10-23", "tags: [旅行, 旅行]", "unknown_key: value", "title: [not, text]",
	} {
		t.Run(frontmatter, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "post.md")
			if err := os.WriteFile(path, []byte("---\n"+frontmatter+"\n---\nBody"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := ReadDocument(path); err == nil {
				t.Fatal("expected frontmatter validation error")
			}
		})
	}
	if _, _, err := parseDocument([]byte("---\ntitle: Unclosed\nbody")); err == nil {
		t.Fatal("unterminated frontmatter accepted")
	}
	if _, _, err := parseDocument([]byte{0xff}); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
}

func TestWriteIndexValidatesBeforeReplacingAndSorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.json")
	entries := []Metadata{
		{Slug: "old", Title: "Old", CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Slug: "new", Title: "New", CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	if err := WriteIndex(path, entries); err != nil {
		t.Fatal(err)
	}
	got, err := LoadIndex(path)
	if err != nil || len(got) != 2 || got[0].Slug != "new" {
		t.Fatalf("LoadIndex = %#v, %v", got, err)
	}
	before, _ := os.ReadFile(path)
	if err := WriteIndex(path, append(entries, entries[0])); err == nil {
		t.Fatal("duplicate accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid write replaced original index")
	}
}

func TestLocalStoreReloadsFilesAndDoesNotHideUnavailableArticles(t *testing.T) {
	dir := t.TempDir()
	if err := WriteIndex(filepath.Join(dir, "index.json"), []Metadata{{Slug: "hello", Title: "Hello"}, {Slug: "missing", Title: "Missing"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hello.md"), []byte("---\ntags: [测试]\ntravel_start: 2025-10-23\ntravel_end: 2025-10-27\n---\nFirst"), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewLocalStore(dir)
	posts, err := store.GetAllPosts()
	if err != nil || len(posts) != 2 || posts[0].Content != "First" || posts[1].GetExcerpt() != unavailableExcerpt || !store.Status().Stale {
		t.Fatalf("posts/status: %#v %#v %v", posts, store.Status(), err)
	}
	if len(posts[0].Tags) != 1 || posts[0].TravelStart != "2025-10-23" {
		t.Fatalf("frontmatter not merged: %#v", posts[0])
	}
	if err := os.WriteFile(filepath.Join(dir, "hello.md"), []byte("Updated"), 0600); err != nil {
		t.Fatal(err)
	}
	post, err := store.GetPost("hello")
	if err != nil || post.Content != "Updated" {
		t.Fatalf("reload: %#v %v", post, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hello.md"), []byte("---\ntitle: New title\ntags: []\n---\nUpdated"), 0600); err != nil {
		t.Fatal(err)
	}
	post, err = store.GetPost("hello")
	if err != nil || post.Title != "New title" || len(post.Tags) != 0 {
		t.Fatalf("frontmatter preview should override stale index: %#v, %v", post, err)
	}
	if post, err := store.GetPost("../secret"); post != nil || err != nil {
		t.Fatalf("invalid slug: %#v %v", post, err)
	}
	if post, err := store.GetPost("missing"); post != nil || err != nil {
		t.Fatalf("missing body: %#v %v", post, err)
	}
	if _, err := NewLocalStore(t.TempDir()).GetAllPosts(); err == nil || !strings.Contains(err.Error(), "index.json") {
		t.Fatalf("missing index: %v", err)
	}
}

func TestLocalFileReadsAreBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "content.txt")
	if err := os.WriteFile(path, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedFile(path, 4); err == nil {
		t.Fatal("oversized file accepted")
	}
	if data, err := readBoundedFile(path, 5); err != nil || string(data) != "12345" {
		t.Fatalf("read = %q %v", data, err)
	}
}
