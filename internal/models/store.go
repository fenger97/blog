package models

import "sync"

// Store 是一个简单的内存存储
type Store struct {
	posts  map[int]*Post
	nextID int
	mu     sync.RWMutex
}

// NewStore 创建一个新的存储实例
func NewStore() *Store {
	return &Store{
		posts:  make(map[int]*Post),
		nextID: 1,
	}
}

// CreatePost 创建一篇新文章
func (s *Store) CreatePost(post *Post) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	post.ID = s.nextID
	s.posts[post.ID] = post
	s.nextID++
	return post.ID
}

// GetPost 获取指定ID的文章
func (s *Store) GetPost(id int) *Post {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.posts[id]
}

// GetAllPosts 获取所有文章
func (s *Store) GetAllPosts() []*Post {
	s.mu.RLock()
	defer s.mu.RUnlock()

	posts := make([]*Post, 0, len(s.posts))
	for _, post := range s.posts {
		posts = append(posts, post)
	}
	return posts
} 