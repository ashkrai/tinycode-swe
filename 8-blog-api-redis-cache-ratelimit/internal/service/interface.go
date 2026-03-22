package service

import (
	"context"

	"github.com/ashkrai/blog-api/internal/model"
)

// PostServiceIface is the interface the HTTP handler depends on.
// The concrete *PostService satisfies it; tests supply a stub.
type PostServiceIface interface {
	ListPosts(ctx context.Context) ([]model.Post, error)
	GetPost(ctx context.Context, id int) (*model.Post, error)
	CreatePost(ctx context.Context, req model.CreatePostRequest) (*model.Post, error)
	UpdatePost(ctx context.Context, id int, req model.CreatePostRequest) (*model.Post, error)
	DeletePost(ctx context.Context, id int) error
	BulkDeletePosts(ctx context.Context, ids []int) (int64, error)
	CacheHealthy(ctx context.Context) bool
}

// Compile-time assertion.
var _ PostServiceIface = (*PostService)(nil)
