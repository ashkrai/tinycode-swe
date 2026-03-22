package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product is the core document stored in MongoDB.
type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
	Name        string             `bson:"name"            json:"name"`
	Description string             `bson:"description"     json:"description"`
	Category    string             `bson:"category"        json:"category"`
	Price       float64            `bson:"price"           json:"price"`
	Stock       int                `bson:"stock"           json:"stock"`
	Tags        []string           `bson:"tags"            json:"tags"`
	Attributes  map[string]string  `bson:"attributes"      json:"attributes"`
	CreatedAt   time.Time          `bson:"created_at"      json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"      json:"updated_at"`
}

// CreateProductRequest is the validated POST/PUT body.
type CreateProductRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Price       float64           `json:"price"`
	Stock       int               `json:"stock"`
	Tags        []string          `json:"tags"`
	Attributes  map[string]string `json:"attributes"`
}

// Validate returns a map of field → error message if anything is wrong.
func (r *CreateProductRequest) Validate() map[string]string {
	errs := map[string]string{}
	if len(r.Name) == 0 {
		errs["name"] = "required"
	}
	if len(r.Name) > 200 {
		errs["name"] = "max 200 characters"
	}
	if len(r.Category) == 0 {
		errs["category"] = "required"
	}
	if r.Price < 0 {
		errs["price"] = "must be >= 0"
	}
	if r.Stock < 0 {
		errs["stock"] = "must be >= 0"
	}
	return errs
}

// CategorySummary is the result of the aggregation pipeline.
type CategorySummary struct {
	Category     string  `bson:"_id"           json:"category"`
	ProductCount int     `bson:"product_count" json:"product_count"`
	AveragePrice float64 `bson:"average_price" json:"average_price"`
	TotalStock   int     `bson:"total_stock"   json:"total_stock"`
}

// ListFilter holds optional query params for listing products.
type ListFilter struct {
	Category string
	MinPrice float64
	MaxPrice float64
	Tag      string
}

// BulkDeleteRequest carries product IDs to delete.
type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

func (r *BulkDeleteRequest) Validate() map[string]string {
	errs := map[string]string{}
	if len(r.IDs) == 0 {
		errs["ids"] = "must contain at least one id"
	}
	return errs
}
