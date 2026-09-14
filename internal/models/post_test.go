package models

import (
	"strings"
	"testing"
)

func TestPostGetHTMLContent(t *testing.T) {
	post := Post{Content: "# Heading\n\n**bold** [link](https://example.com)"}
	html := post.GetHTMLContent()

	for _, expected := range []string{"<h1>Heading</h1>", "<strong>bold</strong>", "href=\"https://example.com\"", "target=\"_blank\""} {
		if !strings.Contains(html, expected) {
			t.Fatalf("HTML %q does not contain %q", html, expected)
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
