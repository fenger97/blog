package handlers

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"blog/internal/models"
)

// ListQuery is shared by the HTML list and the JSON API.
type ListQuery struct {
	Query, Tag, Year string
	Page, Limit      int
	Paginate         bool
}

func ParseListQuery(values url.Values) (ListQuery, error) {
	q := ListQuery{Query: strings.TrimSpace(values.Get("q")), Tag: strings.TrimSpace(values.Get("tag")), Year: strings.TrimSpace(values.Get("year")), Page: 1, Limit: 12}
	if len([]rune(q.Query)) > 200 || len([]rune(q.Tag)) > 64 {
		return q, fmt.Errorf("搜索词或标签过长")
	}
	if q.Year != "" {
		year, err := strconv.Atoi(q.Year)
		if err != nil || len(q.Year) != 4 || year < 1 || year > 9999 {
			return q, fmt.Errorf("年份格式不正确")
		}
	}
	for _, field := range []struct {
		name    string
		value   *int
		maximum int
	}{{"page", &q.Page, 1000000}, {"limit", &q.Limit, 100}} {
		if raw := values.Get(field.name); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > field.maximum {
				return q, fmt.Errorf("%s 参数无效", field.name)
			}
			*field.value = value
			q.Paginate = true
		}
	}
	return q, nil
}

func FilterPosts(posts []*models.Post, q ListQuery) []*models.Post {
	result := make([]*models.Post, 0, len(posts))
	needle := strings.ToLower(q.Query)
	for _, post := range posts {
		if post == nil {
			continue
		}
		if q.Year != "" && post.CreatedAt.Format("2006") != q.Year {
			continue
		}
		if q.Tag != "" {
			found := false
			for _, tag := range post.Tags {
				if tag == q.Tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		haystack := strings.ToLower(post.Title + " " + post.GetExcerpt() + " " + strings.Join(post.Tags, " "))
		if needle != "" && !strings.Contains(haystack, needle) {
			continue
		}
		result = append(result, post)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

func PagePosts(posts []*models.Post, page, limit int) []*models.Post {
	start := (page - 1) * limit
	if start < 0 || start >= len(posts) {
		return []*models.Post{}
	}
	end := start + limit
	if end > len(posts) {
		end = len(posts)
	}
	return posts[start:end]
}
