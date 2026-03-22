package main

import (
	"fmt"
	"net/http"
	"time"
)

// ── Middleware ────────────────────────────────────────────────────────
//
// A middleware is a function that:
//   1. Receives the next handler as an argument
//   2. Does something BEFORE calling it   (e.g. start a timer)
//   3. Calls the next handler             (the actual work happens)
//   4. Does something AFTER it returns    (e.g. print how long it took)
//
// Shape of every middleware:
//
//   func MyMiddleware(next http.Handler) http.Handler {
//       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//           // before
//           next.ServeHTTP(w, r)
//           // after
//       })
//   }

// ── Logger ────────────────────────────────────────────────────────────
//
// Wraps every request and prints:   POST /tasks  201  1.23ms
//
// To capture the status code we need a small wrapper around
// ResponseWriter because the standard one doesn't expose it after
// it's been written.

// responseRecorder wraps http.ResponseWriter so we can read
// the status code after the handler writes it.
type responseRecorder struct {
	http.ResponseWriter        // embed the real one
	statusCode          int    // we capture this ourselves
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default if handler never calls WriteHeader
	}
}

// WriteHeader intercepts the status code before passing it through.
func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := newResponseRecorder(w)

		next.ServeHTTP(rr, r) // ← run the actual handler

		duration := time.Since(start)
		fmt.Printf("%s  %s  %d  %s\n",
			r.Method,
			r.URL.Path,
			rr.statusCode,
			duration.Round(time.Microsecond),
		)
	})
}

// ── Recovery ──────────────────────────────────────────────────────────
//
// If any handler calls panic(), this catches it and returns a 500
// instead of crashing the whole server.
//
// In Go, recover() only works inside a deferred function.
// That's why we use defer here.

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic to the terminal.
				fmt.Printf("PANIC recovered: %v\n", err)
				// Tell the client something went wrong server-side.
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
