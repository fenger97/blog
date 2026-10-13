package models

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"io"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{0,118}[a-z0-9])?$`)

// IsValidSlug reports whether a content slug is safe to use as an article ID.
func IsValidSlug(slug string) bool { return slugPattern.MatchString(slug) }

// Post represents a published article and its reader-facing metadata.
type Post struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Excerpt          string    `json:"-"`
	Content          string    `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Tags             []string  `json:"tags,omitempty"`
	TravelStart      string    `json:"travel_start,omitempty"`
	TravelEnd        string    `json:"travel_end,omitempty"`
	EstimatedMinutes int       `json:"-"`
}

var contentPolicy = func() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("id").Matching(regexp.MustCompile(`^section-[0-9]+$`)).OnElements("h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)).OnElements("pre", "code", "span")
	policy.RequireNoFollowOnLinks(false)
	policy.RequireNoReferrerOnLinks(true)
	policy.AddTargetBlankToFullyQualifiedLinks(true)
	return policy
}()

// GetHTMLContent renders Markdown with stable heading anchors and syntax
// highlighting, then removes dangerous HTML, URL schemes and event handlers.
func (p *Post) GetHTMLContent() string {
	doc := parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs).Parse([]byte(p.Content))
	heading := 0
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if h, ok := node.(*ast.Heading); ok && entering {
			heading++
			h.HeadingID = fmt.Sprintf("section-%d", heading)
		}
		return ast.GoToNext
	})
	renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{
		Flags: mdhtml.CommonFlags | mdhtml.HrefTargetBlank,
		RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
			block, ok := node.(*ast.CodeBlock)
			if !ok {
				return ast.GoToNext, false
			}
			if !entering {
				return ast.GoToNext, true
			}
			if highlighted, err := highlight(block); err == nil {
				_, _ = io.WriteString(w, highlighted)
				return ast.GoToNext, true
			}
			return ast.GoToNext, false
		},
	})
	return contentPolicy.Sanitize(string(markdown.Render(doc, renderer)))
}

func highlight(block *ast.CodeBlock) (string, error) {
	language := ""
	if fields := strings.Fields(string(block.Info)); len(fields) > 0 {
		language = fields[0]
	}
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iterator, err := chroma.Coalesce(lexer).Tokenise(nil, string(block.Literal))
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	err = formatter.Format(&output, styles.Get("github"), iterator)
	return output.String(), err
}

// GetExcerpt returns explicit metadata when available, or a readable summary
// stripped from Markdown. Chinese text is truncated on rune boundaries.
func (p *Post) GetExcerpt() string {
	if strings.TrimSpace(p.Excerpt) != "" {
		return strings.TrimSpace(p.Excerpt)
	}
	excerpt := plainText(p.Content)
	runes := []rune(excerpt)
	if len(runes) <= 60 {
		return excerpt
	}
	short := string(runes[:60])
	lastPeriod := strings.LastIndex(short, "。")
	if lastPeriod >= 0 && len([]rune(short[:lastPeriod])) > 40 {
		return short[:lastPeriod+len("。")]
	}
	return short + "..."
}

// ReadingMinutes estimates Chinese characters at 400/min and non-CJK words at
// 200/min. Zero means that no body is currently available.
func (p *Post) ReadingMinutes() int {
	if p.Content == "" {
		return p.EstimatedMinutes
	}
	text := plainText(p.Content)
	if text == "" {
		return 0
	}
	cjk, words := 0, 0
	inWord := false
	for _, r := range text {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			cjk++
			inWord = false
		} else if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !inWord {
				words++
			}
			inWord = true
		} else {
			inWord = false
		}
	}
	return int(math.Max(1, math.Ceil(float64(cjk)/400+float64(words)/200)))
}

func plainText(content string) string {
	// A quote following a tight list is treated as literal text by the parser;
	// strip quote prefixes before extracting a summary as well.
	content = regexp.MustCompile(`(?m)^[ \t]*(?:>[ \t]*)+`).ReplaceAllString(content, "")
	doc := parser.NewWithExtensions(parser.CommonExtensions).Parse([]byte(content))
	var output strings.Builder
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}
		switch n := node.(type) {
		case *ast.Image:
			return ast.SkipChildren
		case *ast.Text:
			output.Write(n.Literal)
		case *ast.Code, *ast.CodeBlock:
			output.Write(node.AsLeaf().Literal)
			output.WriteByte(' ')
		case *ast.HTMLBlock, *ast.HTMLSpan:
			return ast.SkipChildren
		case *ast.Paragraph, *ast.Heading, *ast.ListItem, *ast.Hardbreak:
			output.WriteByte(' ')
		}
		return ast.GoToNext
	})
	return strings.Join(strings.Fields(stdhtml.UnescapeString(output.String())), " ")
}
