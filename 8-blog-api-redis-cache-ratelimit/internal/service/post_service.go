package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ashkrai/blog-api/internal/cache"
	"github.com/ashkrai/blog-api/internal/model"
	"github.com/ashkrai/blog-api/internal/repository"
)

const (
	postListKey = "posts:list"
	postKeyFmt  = "posts:%d"
	cacheTTL    = cache.DefaultTTL // 60s
)

func postKey(id int) string { return fmt.Sprintf(postKeyFmt, id) }

// PostService contains business logic and orchestrates repo + cache.
type PostService struct {
	repo  *repository.PostRepository
	cache *cache.Client
}

func NewPostService(repo *repository.PostRepository, c *cache.Client) *PostService {
	return &PostService{repo: repo, cache: c}
}

// ListPosts returns all posts, served from cache when available.
func (s *PostService) ListPosts(ctx context.Context) ([]model.Post, error) {
	var posts []model.Post
	err := s.cache.GetJSON(ctx, postListKey, &posts)
	if err == nil {
		return posts, nil // cache hit
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		// Redis error — log and fall through to DB (fail open).
		fmt.Printf("cache get error (list): %v\n", err)
	}

	posts, err = s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	// Best-effort cache write; don't fail the request on cache errors.
	if cerr := s.cache.SetJSON(ctx, postListKey, posts, cacheTTL); cerr != nil {
		fmt.Printf("cache set error (list): %v\n", cerr)
	}
	return posts, nil
}

// GetPost returns a single post by ID, served from cache when available.
func (s *PostService) GetPost(ctx context.Context, id int) (*model.Post, error) {
	key := postKey(id)
	var post model.Post
	err := s.cache.GetJSON(ctx, key, &post)
	if err == nil {
		return &post, nil // cache hit
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		fmt.Printf("cache get error (post %d): %v\n", id, err)
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cerr := s.cache.SetJSON(ctx, key, p, cacheTTL); cerr != nil {
		fmt.Printf("cache set error (post %d): %v\n", id, cerr)
	}
	return p, nil
}

// CreatePost creates a post and busts the list cache.
func (s *PostService) CreatePost(ctx context.Context, req model.CreatePostRequest) (*model.Post, error) {
	p, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	s.bustListCache(ctx)
	return p, nil
}

// UpdatePost updates a post and busts both the list and the individual entry.
func (s *PostService) UpdatePost(ctx context.Context, id int, req model.CreatePostRequest) (*model.Post, error) {
	p, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	s.bustCache(ctx, id)
	return p, nil
}

// DeletePost deletes a post and busts caches.
func (s *PostService) DeletePost(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.bustCache(ctx, id)
	return nil
}

// BulkDeletePosts deletes many posts transactionally and busts all their caches.
func (s *PostService) BulkDeletePosts(ctx context.Context, ids []int) (int64, error) {
	n, err := s.repo.BulkDelete(ctx, ids)
	if err != nil {
		return 0, err
	}
	// Bust every individual key + the list.
	keys := make([]string, 0, len(ids)+1)
	keys = append(keys, postListKey)
	for _, id := range ids {
		keys = append(keys, postKey(id))
	}
	if cerr := s.cache.Delete(ctx, keys...); cerr != nil {
		fmt.Printf("cache bust error (bulk delete): %v\n", cerr)
	}
	return n, nil
}

// bustCache removes both the individual post cache and the list cache.
func (s *PostService) bustCache(ctx context.Context, id int) {
	if err := s.cache.Delete(ctx, postKey(id), postListKey); err != nil {
		fmt.Printf("cache bust error (post %d): %v\n", id, err)
	}
}

// bustListCache removes only the list cache key.
func (s *PostService) bustListCache(ctx context.Context) {
	if err := s.cache.Delete(ctx, postListKey); err != nil {
		fmt.Printf("cache bust error (list): %v\n", err)
	}
}

// HealthCheck pings redis and is used by the /healthz handler.
func (s *PostService) CacheHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	return s.cache.Ping(ctx) == nil
}
