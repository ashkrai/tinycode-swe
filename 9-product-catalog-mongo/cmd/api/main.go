package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/ashkrai/product-catalog/internal/handler"
	"github.com/ashkrai/product-catalog/internal/middleware"
	"github.com/ashkrai/product-catalog/internal/repository"
	"github.com/ashkrai/product-catalog/internal/service"
)

func main() {
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	port := getEnv("PORT", "8081")

	// ── MongoDB client ────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(25).
		SetMinPoolSize(5).
		SetMaxConnIdleTime(5 * time.Minute).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("mongo ping: %v", err)
	}
	log.Println("connected to MongoDB")

	defer func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		if err := client.Disconnect(shutCtx); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}()

	// ── Wire layers ───────────────────────────────────────────────────────
	productRepo, err := repository.NewProductRepository(client)
	if err != nil {
		log.Fatalf("create product repository: %v", err)
	}

	productSvc := service.NewProductService(productRepo)
	productH := handler.NewProductHandler(productSvc)
	healthH := handler.NewHealthzHandler(productSvc)

	// ── Router ────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)

	r.Get("/healthz", healthH.Healthz)

	r.Route("/products", func(r chi.Router) {
		// Analytics — must be registered BEFORE /{id} to avoid route conflict
		r.Get("/analytics/categories", productH.CategorySummary)

		r.Get("/", productH.ListProducts)
		r.Post("/", productH.CreateProduct)
		r.Delete("/bulk", productH.BulkDeleteProducts)
		r.Get("/{id}", productH.GetProduct)
		r.Put("/{id}", productH.UpdateProduct)
		r.Delete("/{id}", productH.DeleteProduct)
	})

	addr := fmt.Sprintf(":%s", port)
	log.Printf("listening on %s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
