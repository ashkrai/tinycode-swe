package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ashkrai/product-catalog/internal/model"
)

const (
	dbName         = "product_catalog"
	collectionName = "products"
)

// ProductRepository handles all MongoDB operations.
type ProductRepository struct {
	col *mongo.Collection
}

// NewProductRepository creates the repo and ensures indexes exist.
func NewProductRepository(client *mongo.Client) (*ProductRepository, error) {
	col := client.Database(dbName).Collection(collectionName)
	r := &ProductRepository{col: col}
	if err := r.ensureIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure indexes: %w", err)
	}
	return r, nil
}

// ensureIndexes creates all required indexes.
// Compound indexes are used for the most common query patterns.
func (r *ProductRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		// Compound index: category + price — supports filtered listing and aggregation.
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "price", Value: 1},
			},
			Options: options.Index().SetName("idx_category_price"),
		},
		// Compound index: category + created_at — supports sorted listing per category.
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_category_created_at"),
		},
		// Single index on name for text search / exact lookup.
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetName("idx_name"),
		},
		// Multi-key index on tags array — enables efficient tag filtering.
		{
			Keys:    bson.D{{Key: "tags", Value: 1}},
			Options: options.Index().SetName("idx_tags"),
		},
		// Index on price alone — supports range queries without category filter.
		{
			Keys:    bson.D{{Key: "price", Value: 1}},
			Options: options.Index().SetName("idx_price"),
		},
	}

	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

// List returns products matching the optional filter.
func (r *ProductRepository) List(ctx context.Context, f model.ListFilter) ([]model.Product, error) {
	filter := bson.M{}

	if f.Category != "" {
		filter["category"] = f.Category
	}
	if f.Tag != "" {
		filter["tags"] = f.Tag
	}
	if f.MinPrice > 0 || f.MaxPrice > 0 {
		priceFilter := bson.M{}
		if f.MinPrice > 0 {
			priceFilter["$gte"] = f.MinPrice
		}
		if f.MaxPrice > 0 {
			priceFilter["$lte"] = f.MaxPrice
		}
		filter["price"] = priceFilter
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(100)

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find products: %w", err)
	}
	defer cur.Close(ctx)

	var products []model.Product
	if err := cur.All(ctx, &products); err != nil {
		return nil, fmt.Errorf("decode products: %w", err)
	}
	if products == nil {
		products = []model.Product{}
	}
	return products, nil
}

// GetByID returns a single product by its ObjectID hex string.
func (r *ProductRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	var p model.Product
	err = r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}
	return &p, nil
}

// Create inserts a new product and returns it with its generated ID.
func (r *ProductRepository) Create(ctx context.Context, req model.CreateProductRequest) (*model.Product, error) {
	now := time.Now().UTC()
	p := model.Product{
		ID:          primitive.NewObjectID(),
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		Stock:       req.Stock,
		Tags:        req.Tags,
		Attributes:  req.Attributes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.Attributes == nil {
		p.Attributes = map[string]string{}
	}

	_, err := r.col.InsertOne(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}
	return &p, nil
}

// Update replaces mutable fields on an existing product.
func (r *ProductRepository) Update(ctx context.Context, id string, req model.CreateProductRequest) (*model.Product, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	attrs := req.Attributes
	if attrs == nil {
		attrs = map[string]string{}
	}

	update := bson.M{
		"$set": bson.M{
			"name":        req.Name,
			"description": req.Description,
			"category":    req.Category,
			"price":       req.Price,
			"stock":       req.Stock,
			"tags":        tags,
			"attributes":  attrs,
			"updated_at":  time.Now().UTC(),
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var p model.Product
	err = r.col.FindOneAndUpdate(ctx, bson.M{"_id": oid}, update, opts).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	return &p, nil
}

// Delete removes a product by ID.
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// BulkDelete removes multiple products in one operation.
func (r *ProductRepository) BulkDelete(ctx context.Context, ids []string) (int64, error) {
	oids := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, fmt.Errorf("invalid id %q: %w", id, err)
		}
		oids = append(oids, oid)
	}
	res, err := r.col.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": oids}})
	if err != nil {
		return 0, fmt.Errorf("bulk delete: %w", err)
	}
	return res.DeletedCount, nil
}

// CategorySummary runs an aggregation pipeline that returns
// product count, average price, and total stock per category.
//
// Pipeline stages:
//  1. $group   — group by category, compute count/avg/sum
//  2. $sort    — sort by product count descending
//  3. $project — round average price to 2 decimal places
func (r *ProductRepository) CategorySummary(ctx context.Context) ([]model.CategorySummary, error) {
	pipeline := mongo.Pipeline{
		// Stage 1: group
		{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$category"},
				{Key: "product_count", Value: bson.D{{Key: "$sum", Value: 1}}},
				{Key: "average_price", Value: bson.D{{Key: "$avg", Value: "$price"}}},
				{Key: "total_stock", Value: bson.D{{Key: "$sum", Value: "$stock"}}},
			}},
		},
		// Stage 2: sort by product count desc
		{
			{Key: "$sort", Value: bson.D{
				{Key: "product_count", Value: -1},
			}},
		},
		// Stage 3: round average_price to 2 decimal places
		{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 1},
				{Key: "product_count", Value: 1},
				{Key: "total_stock", Value: 1},
				{Key: "average_price", Value: bson.D{
					{Key: "$round", Value: bson.A{"$average_price", 2}},
				}},
			}},
		},
	}

	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation: %w", err)
	}
	defer cur.Close(ctx)

	var summaries []model.CategorySummary
	if err := cur.All(ctx, &summaries); err != nil {
		return nil, fmt.Errorf("decode aggregation: %w", err)
	}
	if summaries == nil {
		summaries = []model.CategorySummary{}
	}
	return summaries, nil
}

// Ping checks MongoDB connectivity.
func (r *ProductRepository) Ping(ctx context.Context) error {
	return r.col.Database().Client().Ping(ctx, nil)
}
