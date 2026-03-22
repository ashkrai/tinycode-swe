package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gorilla/mux"

	"blog-api/internal/db"
	"blog-api/internal/handlers"
	"blog-api/internal/repository"
)

func main() {
	// ── flags ──────────────────────────────────────────────────────────────
	var (
		runMigrate   = flag.Bool("migrate", false, "run up-migrations then exit")
		runRollback  = flag.Bool("rollback", false, "rollback last migration then exit")
		showVersion  = flag.Bool("version", false, "print current migration version then exit")
		explainQuery = flag.Bool("explain", false, "run EXPLAIN ANALYZE on list-posts query then exit")
		addr         = flag.String("addr", ":8080", "HTTP listen address")
	)
	flag.Parse()

	// ── database ───────────────────────────────────────────────────────────
	cfg := db.ConfigFromEnv()
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer database.Close()

	migrationsDir := migrationsPath()

	// ── migration commands ─────────────────────────────────────────────────
	if *runMigrate {
		log.Println("running migrations...")
		if err := db.MigrateUp(database, migrationsDir); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
		log.Println("migrations applied successfully")
		os.Exit(0)
	}

	if *runRollback {
		log.Println("rolling back last migration...")
		if err := db.MigrateDown(database, migrationsDir); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		log.Println("rollback complete")
		os.Exit(0)
	}

	if *showVersion {
		v, dirty, err := db.MigrateVersion(database, migrationsDir)
		if err != nil {
			log.Printf("no migrations applied yet or error: %v", err)
		} else {
			log.Printf("schema version: %d  dirty: %v", v, dirty)
		}
		os.Exit(0)
	}

	if *explainQuery {
		postRepo := repository.NewPostRepository(database)
		plan, err := postRepo.ExplainListPublished(context.Background())
		if err != nil {
			log.Fatalf("explain: %v", err)
		}
		fmt.Println(plan)
		os.Exit(0)
	}

	// ── repositories ───────────────────────────────────────────────────────
	userRepo    := repository.NewUserRepository(database)
	postRepo    := repository.NewPostRepository(database)
	tagRepo     := repository.NewTagRepository(database)
	commentRepo := repository.NewCommentRepository(database)

	// ── handlers ───────────────────────────────────────────────────────────
	userH    := handlers.NewUserHandler(userRepo)
	postH    := handlers.NewPostHandler(postRepo)
	tagH     := handlers.NewTagHandler(tagRepo)
	commentH := handlers.NewCommentHandler(commentRepo)

	// ── router ─────────────────────────────────────────────────────────────
	r := mux.NewRouter()
	r.Use(loggingMiddleware)

	// Users
	r.HandleFunc("/api/users",      userH.List).Methods(http.MethodGet)
	r.HandleFunc("/api/users",      userH.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/users/{id}", userH.GetByID).Methods(http.MethodGet)

	// Posts
	r.HandleFunc("/api/posts",               postH.ListPublished).Methods(http.MethodGet)
	r.HandleFunc("/api/posts",               postH.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/posts/{slug}",        postH.GetBySlug).Methods(http.MethodGet)
	r.HandleFunc("/api/admin/explain-posts", postH.ExplainPlan).Methods(http.MethodGet)

	// Tags
	r.HandleFunc("/api/tags", tagH.List).Methods(http.MethodGet)
	r.HandleFunc("/api/tags", tagH.Create).Methods(http.MethodPost)

	// Comments
	r.HandleFunc("/api/posts/{postID}/comments", commentH.ListByPost).Methods(http.MethodGet)
	r.HandleFunc("/api/comments",                commentH.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/comments/{id}/approve",   commentH.Approve).Methods(http.MethodPatch)

	// Health
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := database.PingContext(r.Context()); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	}).Methods(http.MethodGet)

	log.Printf("blog-api listening on %s", *addr)
	if err := http.ListenAndServe(*addr, r); err != nil {
		log.Fatal(err)
	}
}

// migrationsPath returns an absolute path to the migrations directory.
func migrationsPath() string {
	if p := os.Getenv("MIGRATIONS_DIR"); p != "" {
		return p
	}
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
