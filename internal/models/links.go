package models

import (
	"bytes"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// NormalizeArticleHTML resolves content-relative links in already sanitized
// Markdown HTML, identically for server-rendered pages and the JSON API.
func NormalizeArticleHTML(fragment string) string {
	root, err := html.Parse(strings.NewReader(fragment))
	if err != nil {
		return fragment
	}
	var body *html.Node
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "body" {
				body = n
			}
			if n.Data == "img" {
				n.Attr = append(n.Attr, html.Attribute{Key: "loading", Val: "lazy"}, html.Attribute{Key: "decoding", Val: "async"})
			}
			for i, a := range n.Attr {
				if (n.Data == "img" && a.Key == "src") || (n.Data == "a" && a.Key == "href") {
					u, err := url.Parse(a.Val)
					if err != nil || u.IsAbs() || u.Host != "" || strings.HasPrefix(a.Val, "/") || strings.HasPrefix(a.Val, "#") {
						continue
					}
					if n.Data == "img" && strings.HasPrefix(path.Clean(u.Path), "assets/") {
						u.Path = "/content/" + path.Clean(u.Path)
						n.Attr[i].Val = u.String()
					}
					if n.Data == "a" && strings.HasSuffix(u.Path, ".md") {
						slug := strings.TrimSuffix(path.Base(u.Path), ".md")
						if IsValidSlug(slug) {
							u.Path = "/post/" + slug
							n.Attr[i].Val = u.String()
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(root)
	if body == nil {
		return fragment
	}
	var out bytes.Buffer
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&out, c)
	}
	return out.String()
}
