package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/ashkrai/blog-api/internal/cache"
	"github.com/ashkrai/blog-api/internal/handler"
	"github.com/ashkrai/blog-api/internal/middleware"
	internalmigrations "github.com/ashkrai/blog-api/internal/migrations"
	"github.com/ashkrai/blog-api/internal/repository"
	"github.com/ashkrai/blog-api/internal/service"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")
	port := getEnv("PORT", "8080")

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	defer db.Close()

	runMigrations(dbURL)

	redisClient, err := cache.New(redisURL)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	postRepo := repository.NewPostRepository(db)
	postSvc := service.NewPostService(postRepo, redisClient)
	postH := handler.NewPostHandler(postSvc)
	healthH := handler.NewHealthzHandler(db, postSvc)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(chimw.RequestID)
	r.Use(middleware.RateLimiter(redisClient, 100, time.Minute))

	r.Get("/healthz", healthH.Healthz)

	r.Route("/posts", func(r chi.Router) {
		r.Get("/", postH.ListPosts)
		r.Post("/", postH.CreatePost)
		r.Delete("/bulk", postH.BulkDeletePosts)
		r.Get("/{id}", postH.GetPost)
		r.Put("/{id}", postH.UpdatePost)
		r.Delete("/{id}", postH.DeletePost)
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

func runMigrations(dbURL string) {
	d, err := iofs.New(internalmigrations.FS, "sql")
	if err != nil {
		log.Fatalf("migrate iofs: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate up: %v", err)
	}
	log.Println("migrations: up to date")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
