package main

// ── The data shape ────────────────────────────────────────────────────
//
// Same Task struct as before.
// Keeping it in its own file makes it easy to find.

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
