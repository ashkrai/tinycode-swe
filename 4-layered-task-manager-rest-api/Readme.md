# 4. layered-task-manager-rest-api

Run it : 
# terminal 1
go mod tidy
go run .

# every request prints a log line:
# POST  /tasks  201  142µs
# GET   /tasks  200  38µs


# terminal 2 — try it manually
curl -s -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy milk"}' | jq .

# trigger the recovery middleware — server stays alive
curl -s http://localhost:8080/tasks/abc


# run all tests
go test -v ./...
```

---

## What each file is responsible for — and nothing else

| File | Only knows about |
|---|---|
| `task.go` | the data shape |
| `service.go` | the map, the rules, the errors — zero HTTP |
| `handler.go` | reading requests, calling service, writing responses |
| `middleware.go` | wrapping handlers, timing, panic catching |
| `main.go` | connecting all the pieces |

---

## How a request flows through all the layers now
```
POST /tasks  {"title":"Buy milk"}
        ↓
   Recovery middleware    defers a panic catcher
        ↓
   Logger middleware      starts a timer
        ↓
   Create handler         decodes JSON → calls svc.Create("Buy milk")
        ↓
   TaskService.Create     validates title → stores in map → returns Task
        ↓
   Create handler         calls writeJSON(201, task)
        ↓
   Logger middleware      prints  POST /tasks 201 142µs
        ↓
curl receives             {"id":1,"title":"Buy milk","done":false}


How table-driven tests work — the key idea :
tests := []struct {
    name       string
    body       map[string]any
    wantStatus int
}{
    {"valid task",    {"title":"Buy milk"}, 201},
    {"missing title", {"title":""},         400},
    {"empty body",    {},                   400},
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // same test logic, different inputs each time
    })
}
Adding a new test case is one line in the table. No new function, no copy-pasting test logic. That is the whole point.





Before code — here is what you are adding on top of the previous project.
Three new ideas only:

Request comes in
      ↓
  Logger middleware     ← prints: POST /tasks  201  2ms
      ↓
  Recovery middleware   ← if anything panics, catch it → send 500
      ↓
  Handler               ← reads JSON, calls the service
      ↓
  Service layer         ← the actual logic (create, list, delete...)
      ↓
  Store (same map as before)




Middleware is just a function that wraps your handler. It runs before and after. That's it.

Service layer is just moving the map logic out of the handler into its own struct. Handlers become thin — they only read the request and write the response. Business logic lives in the service.


Project structure
taskapi2/
  task.go          ← the Task struct (data shape)
  service.go       ← all business logic, the map store
  handler.go       ← HTTP only, calls the service
  middleware.go    ← logger + recovery
  main.go          ← wires everything together
  handler_test.go  ← all tests