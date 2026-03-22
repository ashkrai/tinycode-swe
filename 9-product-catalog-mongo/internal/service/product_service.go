package service

import (
	"context"

	"github.com/ashkrai/product-catalog/internal/model"
	"github.com/ashkrai/product-catalog/internal/repository"
)

// ProductServiceIface is the contract the HTTP handler depends on.
// Enables stub injection in tests.
type ProductServiceIface interface {
	ListProducts(ctx context.Context, f model.ListFilter) ([]model.Product, error)
	GetProduct(ctx context.Context, id string) (*model.Product, error)
	CreateProduct(ctx context.Context, req model.CreateProductRequest) (*model.Product, error)
	UpdateProduct(ctx context.Context, id string, req model.CreateProductRequest) (*model.Product, error)
	DeleteProduct(ctx context.Context, id string) error
	BulkDeleteProducts(ctx context.Context, ids []string) (int64, error)
	CategorySummary(ctx context.Context) ([]model.CategorySummary, error)
	DBHealthy(ctx context.Context) bool
}

// ProductService contains all business logic.
type ProductService struct {
	repo *repository.ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// Compile-time check.
var _ ProductServiceIface = (*ProductService)(nil)

func (s *ProductService) ListProducts(ctx context.Context, f model.ListFilter) ([]model.Product, error) {
	return s.repo.List(ctx, f)
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) CreateProduct(ctx context.Context, req model.CreateProductRequest) (*model.Product, error) {
	return s.repo.Create(ctx, req)
}

func (s *ProductService) UpdateProduct(ctx context.Context, id string, req model.CreateProductRequest) (*model.Product, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProductService) BulkDeleteProducts(ctx context.Context, ids []string) (int64, error) {
	return s.repo.BulkDelete(ctx, ids)
}

func (s *ProductService) CategorySummary(ctx context.Context) ([]model.CategorySummary, error) {
	return s.repo.CategorySummary(ctx)
}

func (s *ProductService) DBHealthy(ctx context.Context) bool {
	return s.repo.Ping(ctx) == nil
}
