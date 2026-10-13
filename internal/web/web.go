package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"blog/internal/content"
	"blog/internal/handlers"
	"blog/internal/models"
	"golang.org/x/net/html"
)

//go:embed templates/*.html
var templateFiles embed.FS

var pages = template.Must(template.New("pages").Funcs(template.FuncMap{
	"date": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02")
	},
	"isoDate": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339)
	},
	"postURL": func(id string) string { return "/post/" + url.PathEscape(id) },
	"tagURL":  func(tag string) string { return "/?tag=" + url.QueryEscape(tag) },
}).ParseFS(templateFiles, "templates/*.html"))

type Config struct {
	SiteURL, Version, Commit, BuildTime     string
	ContentMode, ContentDir, ContentBaseURL string
	StaticDir                               string
}
type Heading struct {
	ID, Text string
	Level    int
}
type Tag struct {
	Name  string
	Count int
}
type Year struct {
	Year  string
	Count int
}
type PageLink struct {
	Number  int
	URL     string
	Current bool
}
type View struct {
	SiteURL, PageTitle, Description, CanonicalURL, OGType, BuildVersion, ImageURL string
	Posts                                                                         []*models.Post
	Post                                                                          *models.Post
	ArticleHTML                                                                   template.HTML
	TOC                                                                           []Heading
	PrevPost, NextPost                                                            *models.Post
	Tags                                                                          []Tag
	Years                                                                         []Year
	Query, SelectedTag, SelectedYear                                              string
	Page, PageCount, Total                                                        int
	Pages                                                                         []PageLink
	ErrorCode                                                                     int
	ErrorMessage, TravelDates                                                     string
	NoIndex                                                                       bool
}

type site struct {
	store handlers.PostReader
	cfg   Config
}

func New(store handlers.PostReader, cfg Config) http.Handler {
	cfg.SiteURL = strings.TrimRight(cfg.SiteURL, "/")
	if cfg.SiteURL == "" {
		cfg.SiteURL = "https://fenger.xin"
	}
	if cfg.StaticDir == "" {
		cfg.StaticDir = "static"
	}
	if cfg.ContentDir == "" {
		cfg.ContentDir = "content"
	}
	if cfg.ContentMode == "" {
		cfg.ContentMode = "github"
	}
	s := &site{store: store, cfg: cfg}
	api := handlers.NewPostHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/post/", s.post)
	mux.HandleFunc("/posts", api.GetAllPosts)
	mux.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "Post ID is required", 400)
			return
		}
		api.GetPost(w, id)
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))
	mux.HandleFunc("/content/assets/", s.asset)
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/version", s.version)
	mux.HandleFunc("/robots.txt", s.robots)
	mux.HandleFunc("/sitemap.xml", s.sitemap)
	mux.HandleFunc("/rss.xml", s.rss)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", 405)
			return
		}
		if r.Method == http.MethodHead {
			w = headWriter{w}
		}
		mux.ServeHTTP(w, r)
	})
}

type headWriter struct{ http.ResponseWriter }

func (w headWriter) Write(p []byte) (int, error) { return len(p), nil }

func (s *site) view() View {
	version := s.cfg.Version
	if version == "" {
		version = "dev"
	}
	return View{SiteURL: s.cfg.SiteURL, PageTitle: "Fenger's Page", Description: "记录旅行、生活与思考。", CanonicalURL: s.cfg.SiteURL + "/", OGType: "website", BuildVersion: version, ImageURL: s.cfg.SiteURL + "/static/images/avatar-favicon.jpg"}
}
func (s *site) render(w http.ResponseWriter, name string, status int, view View) {
	var buf bytes.Buffer
	if err := pages.ExecuteTemplate(&buf, name, view); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "页面暂时无法显示", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
func (s *site) fail(w http.ResponseWriter, status int, message string) {
	v := s.view()
	v.PageTitle = fmt.Sprintf("%d · Fenger's Page", status)
	v.ErrorCode = status
	v.ErrorMessage = message
	v.NoIndex = true
	s.render(w, "error", status, v)
}
func (s *site) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.fail(w, 404, "这个页面不存在。")
		return
	}
	query, err := handlers.ParseListQuery(r.URL.Query())
	if err != nil {
		s.fail(w, 400, err.Error())
		return
	}
	all, err := s.store.GetAllPosts()
	if err != nil {
		log.Printf("list content: %v", err)
		s.fail(w, 502, "暂时无法读取文章，请稍后重试。")
		return
	}
	posts := handlers.FilterPosts(all, query)
	v := s.view()
	v.Query = query.Query
	v.SelectedTag = query.Tag
	v.SelectedYear = query.Year
	v.Total = len(posts)
	v.Page = query.Page
	v.PageCount = (len(posts) + query.Limit - 1) / query.Limit
	if v.PageCount == 0 {
		v.PageCount = 1
	}
	if query.Page > v.PageCount {
		s.fail(w, 404, "这一页没有文章。")
		return
	}
	v.Posts = handlers.PagePosts(posts, query.Page, query.Limit)
	tagCounts := map[string]int{}
	yearCounts := map[string]int{}
	for _, p := range all {
		if p == nil {
			continue
		}
		for _, tag := range p.Tags {
			tagCounts[tag]++
		}
		if !p.CreatedAt.IsZero() {
			yearCounts[p.CreatedAt.Format("2006")]++
		}
	}
	for tag, n := range tagCounts {
		v.Tags = append(v.Tags, Tag{tag, n})
	}
	sort.Slice(v.Tags, func(i, j int) bool { return v.Tags[i].Name < v.Tags[j].Name })
	for year, n := range yearCounts {
		v.Years = append(v.Years, Year{year, n})
	}
	sort.Slice(v.Years, func(i, j int) bool { return v.Years[i].Year > v.Years[j].Year })
	start, end := query.Page-2, query.Page+2
	if start < 1 {
		start = 1
	}
	if end > v.PageCount {
		end = v.PageCount
	}
	for i := start; i <= end; i++ {
		values := url.Values{}
		if query.Query != "" {
			values.Set("q", query.Query)
		}
		if query.Tag != "" {
			values.Set("tag", query.Tag)
		}
		if query.Year != "" {
			values.Set("year", query.Year)
		}
		values.Set("page", strconv.Itoa(i))
		values.Set("limit", strconv.Itoa(query.Limit))
		v.Pages = append(v.Pages, PageLink{i, "/?" + values.Encode(), i == query.Page})
	}
	v.NoIndex = query.Query != "" || query.Tag != "" || query.Year != "" || query.Page > 1
	s.render(w, "index", 200, v)
}
func (s *site) post(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/post/"), "/")
	if !models.IsValidSlug(id) {
		s.fail(w, 404, "这篇文章不存在。")
		return
	}
	post, err := s.store.GetPost(id)
	if err != nil {
		log.Printf("read post %s: %v", id, err)
		s.fail(w, 502, "文章暂时无法加载，请稍后重试。")
		return
	}
	if post == nil {
		s.fail(w, 404, "这篇文章不存在，可能已被移走。")
		return
	}
	v := s.view()
	v.Post = post
	v.PageTitle = post.Title + " · Fenger's Page"
	v.Description = post.GetExcerpt()
	v.CanonicalURL = s.cfg.SiteURL + "/post/" + url.PathEscape(id)
	v.OGType = "article"
	rendered, toc := prepareArticle(post.GetHTMLContent())
	v.ArticleHTML = template.HTML(rendered)
	v.TOC = toc
	if post.TravelStart != "" {
		v.TravelDates = post.TravelStart
		if post.TravelEnd != "" && post.TravelEnd != post.TravelStart {
			v.TravelDates += " — " + post.TravelEnd
		}
	}
	if all, err := s.store.GetAllPosts(); err == nil {
		all = handlers.FilterPosts(all, handlers.ListQuery{})
		for i, p := range all {
			if p.ID == id {
				if i > 0 {
					v.PrevPost = all[i-1]
				}
				if i+1 < len(all) {
					v.NextPost = all[i+1]
				}
				break
			}
		}
	}
	s.render(w, "post", 200, v)
}

// prepareArticle operates only on HTML already sanitized by the Markdown renderer.
func prepareArticle(fragment string) (string, []Heading) {
	fragment = models.NormalizeArticleHTML(fragment)
	root, err := html.Parse(strings.NewReader(fragment))
	if err != nil {
		return fragment, nil
	}
	var toc []Heading
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				var id string
				for _, a := range n.Attr {
					if a.Key == "id" {
						id = a.Val
					}
				}
				if id != "" {
					toc = append(toc, Heading{id, nodeText(n), int(n.Data[1] - '0')})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(root)
	var body *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(root)
	if body == nil {
		return fragment, toc
	}
	var out bytes.Buffer
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&out, c)
	}
	return out.String(), toc
}
func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func (s *site) asset(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, "/content/")
	clean := path.Clean(rel)
	if !strings.HasPrefix(clean, "assets/") || clean != rel {
		http.NotFound(w, r)
		return
	}
	ext := strings.ToLower(path.Ext(clean))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif":
	default:
		http.NotFound(w, r)
		return
	}
	if s.cfg.ContentMode != "local" {
		base, err := url.Parse(s.cfg.ContentBaseURL)
		if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
			http.NotFound(w, r)
			return
		}
		base.Path = strings.TrimRight(base.Path, "/") + "/" + clean
		http.Redirect(w, r, base.String(), http.StatusTemporaryRedirect)
		return
	}
	dir, err := filepath.EvalSymlinks(s.cfg.ContentDir)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	file, err := filepath.EvalSymlinks(filepath.Join(dir, filepath.FromSlash(clean)))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	file, err = filepath.Abs(file)
	if err != nil || !strings.HasPrefix(file, dir+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(file)
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, file)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *site) version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"version": s.cfg.Version, "commit": s.cfg.Commit, "build_time": s.cfg.BuildTime})
}
func (s *site) health(w http.ResponseWriter, r *http.Request) {
	posts, err := s.store.GetAllPosts()
	status := http.StatusOK
	var state content.Status
	if source, ok := s.store.(interface{ Status() content.Status }); ok {
		state = source.Status()
	}
	if err != nil {
		status = http.StatusServiceUnavailable
		log.Printf("health content: %v", err)
	}
	writeJSON(w, status, map[string]any{"ok": err == nil, "degraded": state.Stale || state.LastError != "", "content_mode": s.cfg.ContentMode, "article_count": len(posts), "last_success": state.LastSuccess, "version": s.cfg.Version})
}
func (s *site) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /api/\nDisallow: /version\nDisallow: /healthz\nSitemap: %s/sitemap.xml\n", s.cfg.SiteURL)
}

type sitemapURL struct {
	Location     string `xml:"loc"`
	LastModified string `xml:"lastmod,omitempty"`
}
type sitemapSet struct {
	XMLName   xml.Name     `xml:"urlset"`
	Namespace string       `xml:"xmlns,attr"`
	URLs      []sitemapURL `xml:"url"`
}

func (s *site) sitemap(w http.ResponseWriter, r *http.Request) {
	posts, err := s.store.GetAllPosts()
	if err != nil {
		s.fail(w, 502, "暂时无法生成站点地图。")
		return
	}
	doc := sitemapSet{Namespace: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: []sitemapURL{{Location: s.cfg.SiteURL + "/"}}}
	for _, p := range posts {
		updated := p.UpdatedAt
		if updated.IsZero() {
			updated = p.CreatedAt
		}
		stamp := ""
		if !updated.IsZero() {
			stamp = updated.Format(time.RFC3339)
		}
		doc.URLs = append(doc.URLs, sitemapURL{s.cfg.SiteURL + "/post/" + url.PathEscape(p.ID), stamp})
	}
	writeXML(w, doc, "application/xml; charset=utf-8")
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
	Published   string `xml:"pubDate,omitempty"`
}
type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []rssItem `xml:"item"`
}
type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

func (s *site) rss(w http.ResponseWriter, r *http.Request) {
	posts, err := s.store.GetAllPosts()
	if err != nil {
		s.fail(w, 502, "暂时无法生成订阅内容。")
		return
	}
	posts = handlers.FilterPosts(posts, handlers.ListQuery{})
	doc := rssDocument{Version: "2.0", Channel: rssChannel{Title: "Fenger's Page", Link: s.cfg.SiteURL + "/", Description: s.view().Description, Language: "zh-CN"}}
	if len(posts) > 50 {
		posts = posts[:50]
	}
	for _, p := range posts {
		link := s.cfg.SiteURL + "/post/" + url.PathEscape(p.ID)
		published := ""
		if !p.CreatedAt.IsZero() {
			published = p.CreatedAt.Format(time.RFC1123Z)
		}
		doc.Channel.Items = append(doc.Channel.Items, rssItem{p.Title, link, p.GetExcerpt(), link, published})
	}
	writeXML(w, doc, "application/rss+xml; charset=utf-8")
}
func writeXML(w http.ResponseWriter, value any, kind string) {
	data, err := xml.MarshalIndent(value, "", "  ")
	if err != nil {
		http.Error(w, "无法生成内容", 500)
		return
	}
	w.Header().Set("Content-Type", kind)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(data)
}
