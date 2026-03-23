# 3. task-manager-rest-api

# terminal 1 — start the server
go mod tidy          # downloads the chi router
go run main.go

# terminal 2 — test it
# install jq first if you don't have it: sudo apt install jq
bash test.sh
```

---

## What you should see
```
── CREATE two tasks ─────────────────────────────
{ "id": 1, "title": "Buy milk",   "done": false }
{ "id": 2, "title": "Write code", "done": false }

── LIST all tasks ───────────────────────────────
[ { "id": 1, ... }, { "id": 2, ... } ]

── UPDATE task 1 ────────────────────────────────
{ "id": 1, "title": "Buy oat milk", "done": true }

── DELETE task 2 ────────────────────────────────
(no response body — 204 means deleted)

── GET a task that does not exist ───────────────
{ "error": "task not found" }
```

---

## HTTP status codes used — and why they matter

| Code | Name | When you use it |
|---|---|---|
| `200` | OK | Request worked, here's the data |
| `201` | Created | A new thing was made |
| `204` | No Content | Worked, but nothing to return (delete) |
| `400` | Bad Request | Client sent garbage (missing title, bad JSON) |
| `404` | Not Found | That ID doesn't exist |

These aren't just convention — `curl`, browsers, and every HTTP client treat `2xx` as success and `4xx` as the client's fault. If you always return `200` for everything, callers can't tell if something went wrong.

---

## How the request flows through the code
```
curl -X POST /tasks  {"title":"Buy milk"}
        ↓
   chi router        sees POST + /tasks
        ↓
  createTask()       decodes JSON body → Task struct
        ↓
   tasks map         tasks[1] = Task{ID:1, Title:"Buy milk"}
        ↓
  writeJSON()        encodes Task → JSON, writes 201 response
        ↓
curl receives        {"id":1,"title":"Buy milk","done":false}


Before code — here is exactly what you are building:

# create a task
curl -X POST http://localhost:8080/tasks \
  -d '{"title": "Buy milk"}'

# get all tasks
curl http://localhost:8080/tasks

# get one task
curl http://localhost:8080/tasks/1

# update a task
curl -X PUT http://localhost:8080/tasks/1 \
  -d '{"title": "Buy oat milk", "done": true}'

# delete a task
curl -X DELETE http://localhost:8080/tasks/1

A running HTTP server that stores tasks in memory. No database, no files — data lives in a Go map while the server is running, gone when you stop it.

The new concepts in this project :
HTTP handler — a Go function that receives a request and writes a response. That's all a web server is: a loop that calls your function for every incoming request.

Router — decides which handler to call based on the URL. /tasks → one function. /tasks/42 → a different function.

JSON — how data travels over HTTP. Your Go struct gets converted to JSON before sending, and JSON from the client gets converted back to a Go struct.

In-memory store — just a Go map. map[int]Task. No database needed yet. This lets you focus on HTTP without learning SQL at the same time.