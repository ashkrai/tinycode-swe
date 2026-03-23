# 10-protobuf-schema-serialisation-versioning

Protocol Buffers 101 — schema design, serialisation, versioning, and binary vs JSON comparison.

Continues the series from `10-product-catalog-mongo`.


This project is correctly a CLI demo program that:
Builds a User struct
Serialises to binary + JSON
Compares sizes
Deserialises and verifies round-trip
Shows v1/v2 backward compat


---

## What This Project Covers

| Topic | Where |
|---|---|
| `.proto` schema with enums + nested messages | `proto/user_v1.proto`, `proto/user_v2.proto` |
| Wire format serialisation (binary) | `internal/user/user.go` — `Marshal()` |
| Wire format deserialisation | `internal/user/user.go` — `Unmarshal()` |
| JSON serialisation | `cmd/demo/main.go` — `json.Marshal` |
| Binary vs JSON size comparison | `cmd/demo/main.go` — Step 3 |
| Proto versioning (v1 → v2) | `proto/user_v2.proto` — comments explain rules |
| Backward compat: v1 reader reads v2 binary | `internal/user/user.go` — `UnmarshalMaxField(b, 13)` |
| Forward compat: v2 reader reads v1 binary | `cmd/demo/main.go` — Step 6 |
| Table-driven unit tests | `internal/user/user_test.go` |

---

## Quick Start

```bash
cd 11-protobuf-user
go mod tidy
go run ./cmd/demo
```

Expected output:
```
╔══════════════════════════════════╗
║  11 · Protocol Buffers Demo      ║
╚══════════════════════════════════╝

── Step 1 — Serialise to Binary ───────────────────────────────

  Binary size: 312 bytes
  First 64 bytes (hex): 0a1e7573725f30314...

── Step 2 — Serialise to JSON ─────────────────────────────────

  JSON size: 891 bytes
  ...

── Step 3 — Binary vs JSON Size Comparison ────────────────────

  ┌──────────────────────────────────────────────┐
  │  Format    │  Size     │  Ratio              │
  ├──────────────────────────────────────────────┤
  │  Protobuf  │   312 B   │  1.00× (baseline)   │
  │  JSON      │   891 B   │  2.86× larger        │
  ├──────────────────────────────────────────────┤
  │  Protobuf is 65% smaller than JSON           │
  └──────────────────────────────────────────────┘
```

---

## Run Tests

```bash
go test ./... -v
```

---

## Proto Schema Design

### User v1 (`proto/user_v1.proto`)

```
User
├── id, username, first_name, last_name   (string, fields 1–4)
├── role                                  (enum Role, field 5)
├── status                                (enum AccountStatus, field 6)
├── contact                               (message ContactInfo, field 7)
│   ├── email, phone
│   └── social_links []string
├── address                               (message Address, field 8)
│   └── street, city, state, country, zip
├── preferences                           (message Preferences, field 9)
│   ├── language, timezone
│   └── email_newsletter bool, dark_mode bool
├── group_ids                             (repeated string, field 10)
├── metadata                              (map<string,string>, field 11)
├── created_at_unix                       (int64, field 12)
└── updated_at_unix                       (int64, field 13)
```

### User v2 (`proto/user_v2.proto`) — backward-compatible additions

```
User (v2 additions, field numbers 14–17)
├── subscription_tier   (enum SubscriptionTier, field 14)  ← NEW
├── audit               (message AuditInfo, field 15)      ← NEW
│   ├── last_login_ip, last_login_unix
│   └── login_count int32, mfa_enabled bool
├── display_name        (string, field 16)                 ← NEW
└── badge_ids           (repeated string, field 17)        ← NEW

ContactInfo v2: + website (field 4)
Address v2:     + formatted (field 6)
Preferences v2: + theme_color (field 5)
```

---

## Wire Format Explained

Protobuf binary format uses **tag-value pairs**:

```
Tag = (field_number << 3) | wire_type

Wire types:
  0 = Varint  — int32, int64, bool, enum
  2 = Len     — string, bytes, embedded message, repeated string
```

Example: field 1 (id), wire type 2 (string):
```
Tag byte: (1 << 3) | 2 = 0x0a
Then: length varint + UTF-8 bytes
```

The binary has **no field names**, **no quotes**, **no braces** — just compact tag+value pairs. That's why it's 2–4× smaller than JSON.

---

## Versioning Rules (How We Added v2 Fields)

### Rules Followed

| Rule | Why |
|---|---|
| Never remove a field | Old binary has the bytes; new reader must decode |
| Never renumber a field | Tag = field_number; changing breaks decoders |
| Never change a field type | Old encoder wrote wrong wire type |
| Add new fields with NEW numbers | Old readers skip unknown numbers |
| Keep proto3 defaults (0/empty) | Absent fields = zero, never panic |

### Backward Compatibility (v1 reads v2 binary)

```
v2 binary wire:
  field 1  (id)              → v1 reads ✔
  field 7  (contact)         → v1 reads ✔
  field 14 (subscription_tier) → v1 SKIPS (unknown)
  field 15 (audit)           → v1 SKIPS (unknown)
  field 16 (display_name)    → v1 SKIPS (unknown)
  field 17 (badge_ids)       → v1 SKIPS (unknown)

Result: v1 reader gets a valid User with v1 fields intact ✔
```

### Forward Compatibility (v2 reads v1 binary)

```
v1 binary wire:
  field 1–13 present
  fields 14–17 absent

v2 reader: missing fields = proto3 zero values
  subscription_tier → SUBSCRIPTION_TIER_UNSPECIFIED
  audit             → nil
  display_name      → ""
  badge_ids         → []

Result: v2 reader gets a valid User, no crash ✔
```

---

## Generating Code from .proto (Optional)

The project ships `internal/user/user.go` which implements the same types
without requiring `protoc`. If you want the official generated code:

```bash
# macOS
brew install protobuf
make install-tools
make generate
```

This runs `protoc --go_out=...` and produces `internal/user/user_v2.pb.go`.
You can then delete `user.go` and use the generated file instead.

---

## Binary vs JSON: When to Use Each

| Use Protobuf binary | Use JSON |
|---|---|
| Internal service-to-service (gRPC) | Public APIs, REST endpoints |
| High-throughput event streaming (Kafka) | Human-readable logs/configs |
| Mobile apps with bandwidth constraints | Browser/JS interoperability |
| Strict schema enforcement needed | Rapid prototyping |
| 2–4× size savings matter at scale | Debuggability matters more |

---

## Project Structure

```
11-protobuf-user/
├── cmd/demo/main.go           # demo: serialise, compare, compat
├── proto/
│   ├── user_v1.proto          # original schema
│   └── user_v2.proto          # backward-compatible additions
├── internal/user/
│   ├── user.go                # types + Marshal/Unmarshal (wire format)
│   └── user_test.go           # table-driven tests
├── go.mod
├── Makefile
└── README.md
```
