# 9-product-catalog-mongo

A production-style Go REST API for a **product catalog** backed by **MongoDB**.

Builds on the patterns established in `8-blog-api` and `9-blog-api-redis-cache-ratelimit`:
same chi router, same middleware stack, same service/repository/handler layering —
but the persistence layer is MongoDB instead of Postgres.

---

## What's New in This Project

| Feature | Detail |
|---|---|
| **MongoDB** | Document store via `go.mongodb.org/mongo-driver` |
| **Compound indexes** | `(category, price)`, `(category, created_at)`, multikey on `tags` |
| **Aggregation pipeline** | Count, average price, total stock per category |
| **Flexible schema** | `attributes map[string]string` — arbitrary key/value per product |
| **Query filters** | `?category=`, `?tag=`, `?min_price=`, `?max_price=` |
| **Bulk delete** | `DeleteMany` with ObjectID array |

---

## Architecture

```
HTTP Request
     │
  chi Router
     │
  Middleware (Recoverer → Logger → RequestID)
     │
  ProductHandler
  (decode JSON, validate, extract query params)
     │
  ProductService
  (business logic)
     │
  ProductRepository
  (mongo-driver: Find, FindOne, InsertOne,
   FindOneAndUpdate, DeleteOne, DeleteMany, Aggregate)
     │
  MongoDB
```

---

## Quick Start

```bash
# 1. Start MongoDB + API
docker-compose up --build

# 2. Check health
curl http://localhost:8081/healthz

# 3. Create a product
curl -X POST http://localhost:8081/products/ \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "MacBook Pro",
    "category": "electronics",
    "price": 1999.99,
    "stock": 25,
    "tags": ["laptop", "apple", "featured"],
    "attributes": {"brand": "Apple", "ram": "16GB", "storage": "512GB"}
  }'

# 4. List all products
curl http://localhost:8081/products/

# 5. Filter by category and price range
curl "http://localhost:8081/products/?category=electronics&min_price=500&max_price=2000"

# 6. Aggregation — count + average price per category
curl http://localhost:8081/products/analytics/categories

# 7. Run the full test suite
chmod +x test.sh && ./test.sh
```

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | MongoDB liveness check |
| GET | `/products/` | List products (filterable) |
| POST | `/products/` | Create product |
| GET | `/products/:id` | Get product by ObjectID |
| PUT | `/products/:id` | Update product |
| DELETE | `/products/:id` | Delete product |
| DELETE | `/products/bulk` | Bulk delete (DeleteMany) |
| GET | `/products/analytics/categories` | Aggregation pipeline |

### Query Parameters for `GET /products/`

| Param | Example | Description |
|---|---|---|
| `category` | `?category=electronics` | Filter by category |
| `tag` | `?tag=sale` | Filter by tag (multikey index) |
| `min_price` | `?min_price=10.00` | Price lower bound |
| `max_price` | `?max_price=500.00` | Price upper bound |

---

## MongoDB Indexes

```javascript
// Compound: category + price — supports filtered listing + aggregation
{ category: 1, price: 1 }  →  idx_category_price

// Compound: category + created_at — supports sorted listing per category
{ category: 1, created_at: -1 }  →  idx_category_created_at

// Single: name lookup
{ name: 1 }  →  idx_name

// Multikey: tags array — one index entry per tag value
{ tags: 1 }  →  idx_tags

// Single: price range queries without category
{ price: 1 }  →  idx_price
```

---

## Aggregation Pipeline

`GET /products/analytics/categories` runs this pipeline:

```javascript
[
  // Stage 1 — group by category, compute metrics
  { $group: {
      _id: "$category",
      product_count: { $sum: 1 },
      average_price: { $avg: "$price" },
      total_stock:   { $sum: "$stock" }
  }},

  // Stage 2 — sort by most products first
  { $sort: { product_count: -1 } },

  // Stage 3 — round average_price to 2 decimal places
  { $project: {
      _id: 1,
      product_count: 1,
      total_stock: 1,
      average_price: { $round: ["$average_price", 2] }
  }}
]
```

Sample response:
```json
[
  { "category": "electronics", "product_count": 5, "average_price": 749.99, "total_stock": 230 },
  { "category": "tools",       "product_count": 3, "average_price": 44.99,  "total_stock": 450 },
  { "category": "appliances",  "product_count": 2, "average_price": 64.99,  "total_stock": 70  }
]
```

---

## When to Use MongoDB vs PostgreSQL

This is the central design decision for any new service. Here's the framework:

### Use PostgreSQL (like in `8-blog-api`) when:

| Scenario | Reason |
|---|---|
| **Relationships matter** | Users → Posts → Comments → Tags — FKs enforce integrity |
| **Transactions are required** | `BEGIN / COMMIT / ROLLBACK` across multiple tables |
| **Schema is stable and known upfront** | Posts always have title, body, user_id |
| **Complex joins are common** | "Give me all posts with their authors and comment counts" |
| **Aggregate queries need accuracy** | Financial sums, inventory counts — no approximation |
| **Compliance / audit trail** | ACID guarantees, WAL, point-in-time recovery |
| **Data is highly normalized** | Avoid duplication, enforce constraints |

**In this project:** Blog users, posts, tags, and comments are all relational.
A post *belongs to* a user. A comment *belongs to* a post and a user.
Deleting a user should cascade to their posts. This is inherently relational.

---

### Use MongoDB (like in this project) when:

| Scenario | Reason |
|---|---|
| **Schema varies per document** | A laptop has `ram`, `storage`; a hammer has `material`, `handle_type` |
| **Deeply nested / hierarchical data** | Product → variants → images → reviews, all in one document |
| **Write-heavy, high-throughput** | Event logs, IoT sensor data, clickstreams |
| **Flexible evolution** | New product types with new attributes — no `ALTER TABLE` |
| **Aggregation pipelines** | `$group`, `$lookup`, `$unwind` — powerful built-in analytics |
| **Array fields are first-class** | Tags, image URLs, related IDs — multikey indexes on arrays |
| **Geo-spatial queries** | `$near`, `$geoWithin` built into the query engine |
| **Read replicas + sharding** | Horizontal scaling built into the architecture |

**In this project:** Products have an `attributes map[string]string` field.
A laptop has `{brand, ram, storage, display}`. A hammer has `{material, handle_type, weight}`.
Forcing these into Postgres would require either a generic `JSONB` column (losing type safety)
or separate `product_attributes` tables (complex joins for every read).
MongoDB stores each product as a self-contained document — the query for a laptop
returns everything needed in a single round trip with no joins.

---

### The Decision Flowchart

```
Does your data have strict relationships
with referential integrity requirements?
          │
         YES → PostgreSQL
          │
         NO
          │
Does your schema vary significantly
between records (product catalog,
CMS content, user-generated data)?
          │
         YES → MongoDB
          │
         NO
          │
Do you need ACID transactions across
multiple collections/documents?
          │
         YES → PostgreSQL (or MongoDB 4.x+ with caution)
          │
         NO
          │
Is write throughput > 10k/sec or do
you need horizontal sharding?
          │
         YES → MongoDB
          │
         NO → Either works; choose based on team familiarity
```

---

### Side-by-Side: Blog (Postgres) vs Product Catalog (Mongo)

```
Blog API (Postgres)                  Product Catalog (Mongo)
─────────────────────────────────    ────────────────────────────────────
Fixed schema: title, body, user_id   Flexible: attributes per product type
FK: posts.user_id → users.id         No FK: products are self-contained
JOIN: post + author in one query      Single document fetch: all data inside
Migration: ALTER TABLE for changes    No migration: add field, just write it
Transactions: bulk delete with TX     DeleteMany: no TX needed, atomic op
Aggregate: SQL GROUP BY               Aggregate: $group pipeline stage
index on user_id (FK)                 Compound (category, price) + multikey tags
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection string |
| `PORT` | `8081` | HTTP listen port |

---

## Connection Pool Settings

```go
clientOpts := options.Client().
    SetMaxPoolSize(25).           // max concurrent connections
    SetMinPoolSize(5).            // keep alive minimum
    SetMaxConnIdleTime(5 * time.Minute).  // recycle idle connections
    SetConnectTimeout(10 * time.Second).
    SetServerSelectionTimeout(5 * time.Second)
```
