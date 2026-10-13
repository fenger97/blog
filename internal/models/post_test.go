package models

import (
	"strings"
	"testing"
)

func TestPostGetHTMLContent(t *testing.T) {
	post := Post{Content: "# Heading\n\n**bold** [link](https://example.com)"}
	html := post.GetHTMLContent()

	for _, expected := range []string{"<h1 id=\"section-1\">Heading</h1>", "<strong>bold</strong>", "href=\"https://example.com\"", "target=\"_blank\"", "noopener"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("HTML %q does not contain %q", html, expected)
		}
	}
}

func TestPostHTMLSanitizesUnsafeContentAndHighlightsCode(t *testing.T) {
	post := Post{Content: "## Repeated\n\n## Repeated\n\n<script>alert('bad')</script>\n\n<img src=\"https://example.com/image.jpg\" onerror=\"alert(1)\">\n\n[unsafe](javascript:alert(1))\n\n```go\nfunc main() {}\n```"}
	html := post.GetHTMLContent()
	for _, bad := range []string{"<script", "onerror", "javascript:", "alert('bad')"} {
		if strings.Contains(html, bad) {
			t.Fatalf("unsafe output contains %q: %s", bad, html)
		}
	}
	for _, good := range []string{`id="section-1"`, `id="section-2"`, `class="chroma"`, `class="kd"`, `src="https://example.com/image.jpg"`} {
		if !strings.Contains(html, good) {
			t.Fatalf("output lacks %q: %s", good, html)
		}
	}
}

func TestPostExcerptAndReadingMinutes(t *testing.T) {
	if got := (&Post{EstimatedMinutes: 7}).ReadingMinutes(); got != 7 {
		t.Fatalf("metadata reading time = %d, want 7", got)
	}
	if got := (&Post{Content: "短文", EstimatedMinutes: 7}).ReadingMinutes(); got != 1 {
		t.Fatalf("body should override estimate, got %d", got)
	}
	if got := (&Post{Content: "<script>bad()</script>\n\n## 你好\n\n正文"}).GetExcerpt(); got != "你好 正文" {
		t.Fatalf("excerpt = %q", got)
	}
	for _, test := range []struct {
		body string
		want int
	}{
		{"", 0}, {"hello", 1}, {strings.Repeat("word ", 201), 2}, {strings.Repeat("文", 401), 2},
	} {
		if got := (&Post{Content: test.body}).ReadingMinutes(); got != test.want {
			t.Fatalf("ReadingMinutes() = %d, want %d", got, test.want)
		}
	}
}

func TestPostGetExcerpt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "removes Markdown markers", content: "# Title\n\n- **bold** [link](https://example.com)\n> quoted", want: "Title bold link quoted"},
		{name: "removes images", content: "Before ![image](https://example.com/image.png) after", want: "Before after"},
		{name: "truncates long content", content: strings.Repeat("文", 61), want: strings.Repeat("文", 60) + "..."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := (&Post{Content: test.content}).GetExcerpt(); got != test.want {
				t.Fatalf("excerpt = %q, want %q", got, test.want)
			}
		})
	}
}
