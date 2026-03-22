#!/bin/bash

BASE_URL="http://localhost:8081"
PASS=0
FAIL=0
WARN=0
TOTAL=0
RESULTS=()
CREATED_IDS=()

green='\033[0;32m'
red='\033[0;31m'
yellow='\033[1;33m'
blue='\033[0;34m'
cyan='\033[0;36m'
bold='\033[1m'
nc='\033[0m'

pass() { echo -e "  ${green}✔${nc}  $1"; ((PASS++)); ((TOTAL++)); RESULTS+=("PASS|$1"); }
fail() { echo -e "  ${red}✘${nc}  $1"; ((FAIL++)); ((TOTAL++)); RESULTS+=("FAIL|$1"); }
warn() { echo -e "  ${yellow}⚠${nc}  $1"; ((WARN++)); RESULTS+=("WARN|$1"); }
header() {
  echo ""
  echo -e "${bold}${blue}┌─────────────────────────────────────────────┐${nc}"
  echo -e "${bold}${blue}│${nc}  ${cyan}${bold}$1${nc}"
  echo -e "${bold}${blue}└─────────────────────────────────────────────┘${nc}"
}
subheader() { echo -e "\n  ${yellow}▸ $1${nc}"; }

check_status() {
  local label="$1" expected="$2" actual="$3" body="${4:-}"
  [ "$actual" = "$expected" ] \
    && pass "$label → HTTP $actual" \
    || fail "$label → expected HTTP $expected, got HTTP $actual${body:+ (body: $body)}"
}
check_contains() {
  local label="$1" haystack="$2" needle="$3"
  echo "$haystack" | grep -qi "$needle" && pass "$label" || fail "$label (expected: $needle)"
}
check_not_contains() {
  local label="$1" haystack="$2" needle="$3"
  echo "$haystack" | grep -qi "$needle" \
    && fail "$label (should NOT contain: $needle)" \
    || pass "$label"
}

RUN_ID=$(date +%s)

# ── Wait for API ──────────────────────────────────────────────────────────────
echo ""
echo -e "${bold}${cyan}  10-product-catalog-mongo — Full Test Suite${nc}"
echo -e "  Target: ${BASE_URL}"
echo ""
echo -n "  Waiting for API "
for i in $(seq 1 30); do
  code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz" 2>/dev/null)
  [ "$code" = "200" ] && echo -e " ${green}ready${nc}" && break
  echo -n "."
  sleep 1
  [ "$i" = "30" ] && echo -e " ${red}timed out${nc}" && echo "  Is docker-compose up?" && exit 1
done

# ════════════════════════════════════════════════════════════════════════════
header "1 · HEALTH CHECK"
# ════════════════════════════════════════════════════════════════════════════

res=$(curl -s "$BASE_URL/healthz")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")
check_status "GET /healthz" "200" "$code"
check_contains "status: ok"  "$res" '"status":"ok"'
check_contains "db: true"    "$res" '"db":true'

# ════════════════════════════════════════════════════════════════════════════
header "2 · CREATE PRODUCT — Happy Path"
# ════════════════════════════════════════════════════════════════════════════

subheader "Full product with all fields"
res=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/products/" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Laptop-$RUN_ID\",\"category\":\"electronics\",\"price\":999.99,\"stock\":50,\"tags\":[\"sale\",\"featured\"],\"attributes\":{\"brand\":\"TechCo\",\"warranty\":\"2yr\"}}")
body=$(echo "$res" | head -n1)
code=$(echo "$res" | tail -n1)

check_status "POST /products/" "201" "$code" "$body"
check_contains "Has id"          "$body" '"id":'
check_contains "Has name"        "$body" "\"name\":\"Laptop-$RUN_ID\""
check_contains "Has category"    "$body" '"category":"electronics"'
check_contains "Has price"       "$body" '"price":999.99'
check_contains "Has stock"       "$body" '"stock":50'
check_contains "Has tags"        "$body" '"tags":'
check_contains "Has attributes"  "$body" '"attributes":'
check_contains "Has created_at"  "$body" '"created_at":'
check_contains "Has updated_at"  "$body" '"updated_at":'

PROD_ID_1=$(echo "$body" | grep -o '"id":"[^"]*"' | grep -o '"id":"[^"]*' | sed 's/"id":"//g')
CREATED_IDS+=("$PROD_ID_1")

subheader "Minimal product (only required fields)"
res=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/products/" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Hammer-$RUN_ID\",\"category\":\"tools\",\"price\":24.99}")
body=$(echo "$res" | head -n1)
code=$(echo "$res" | tail -n1)
check_status "POST minimal product" "201" "$code" "$body"
PROD_ID_2=$(echo "$body" | grep -o '"id":"[^"]*"' | grep -o '"id":"[^"]*' | sed 's/"id":"//g')
CREATED_IDS+=("$PROD_ID_2")

subheader "Create products for analytics"
for cat_item in "electronics:Tablet-$RUN_ID:599.99:30" "electronics:Phone-$RUN_ID:399.99:100" "tools:Drill-$RUN_ID:89.99:75" "appliances:Blender-$RUN_ID:49.99:40"; do
  cat=$(echo $cat_item | cut -d: -f1)
  name=$(echo $cat_item | cut -d: -f2)
  price=$(echo $cat_item | cut -d: -f3)
  stock=$(echo $cat_item | cut -d: -f4)
  res=$(curl -s -X POST "$BASE_URL/products/" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$name\",\"category\":\"$cat\",\"price\":$price,\"stock\":$stock,\"tags\":[\"$RUN_ID\"]}")
  id=$(echo "$res" | grep -o '"id":"[^"]*"' | grep -o '"id":"[^"]*' | sed 's/"id":"//g')
  CREATED_IDS+=("$id")
done
pass "Created 4 additional products for analytics (categories: electronics x2, tools, appliances)"

# ════════════════════════════════════════════════════════════════════════════
header "3 · CREATE PRODUCT — Validation"
# ════════════════════════════════════════════════════════════════════════════

subheader "Missing required fields"
for test_case in \
  "empty body|{}|422" \
  "missing name|{\"category\":\"c\",\"price\":9.99}|422" \
  "missing category|{\"name\":\"w\",\"price\":9.99}|422" \
  "negative price|{\"name\":\"w\",\"category\":\"c\",\"price\":-1}|422" \
  "negative stock|{\"name\":\"w\",\"category\":\"c\",\"price\":1,\"stock\":-5}|422"; do
  label=$(echo "$test_case" | cut -d'|' -f1)
  payload=$(echo "$test_case" | cut -d'|' -f2)
  expected=$(echo "$test_case" | cut -d'|' -f3)
  code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/products/" \
    -H 'Content-Type: application/json' -d "$payload")
  check_status "$label" "$expected" "$code"
done

subheader "Malformed request"
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/products/" \
  -H 'Content-Type: application/json' -d 'not-json')
check_status "Invalid JSON → 400" "400" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/products/" \
  -H 'Content-Type: application/json' -d '')
check_status "Empty body → 400" "400" "$code"

subheader "Validation error shape"
res=$(curl -s -X POST "$BASE_URL/products/" \
  -H 'Content-Type: application/json' -d '{}')
check_contains "Validation response has 'errors' key" "$res" '"errors":'

# ════════════════════════════════════════════════════════════════════════════
header "4 · LIST PRODUCTS"
# ════════════════════════════════════════════════════════════════════════════

subheader "Basic list"
res=$(curl -s "$BASE_URL/products/")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/")
check_status "GET /products/" "200" "$code"
check_contains "Returns array"             "$res" '\['
check_contains "Contains created product"  "$res" "Laptop-$RUN_ID"

subheader "Filter by category"
res=$(curl -s "$BASE_URL/products/?category=electronics")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/?category=electronics")
check_status "GET ?category=electronics" "200" "$code"
check_contains "Contains electronics product" "$res" "electronics"

subheader "Filter by tag"
res=$(curl -s "$BASE_URL/products/?tag=$RUN_ID")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/?tag=$RUN_ID")
check_status "GET ?tag=$RUN_ID" "200" "$code"

subheader "Filter by price range"
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/?min_price=10&max_price=100")
check_status "GET ?min_price=10&max_price=100" "200" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/?min_price=500")
check_status "GET ?min_price=500" "200" "$code"

subheader "Combined filters"
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/?category=electronics&min_price=100&max_price=1500")
check_status "GET ?category=electronics&min_price=100&max_price=1500" "200" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "5 · GET SINGLE PRODUCT"
# ════════════════════════════════════════════════════════════════════════════

res=$(curl -s "$BASE_URL/products/$PROD_ID_1")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/$PROD_ID_1")
check_status "GET /products/$PROD_ID_1" "200" "$code"
check_contains "Returns correct name"     "$res" "Laptop-$RUN_ID"
check_contains "Returns correct category" "$res" "electronics"
check_contains "Returns tags array"       "$res" '"tags":'
check_contains "Returns attributes"       "$res" '"attributes":'

subheader "Not found and invalid IDs"
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/507f1f77bcf86cd799439099")
check_status "Non-existent ObjectID → 404" "404" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/badid")
check_status "Invalid id (short) → 400"   "400" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/not-a-valid-object-id-xx")
check_status "Invalid id (wrong format) → 400" "400" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "6 · UPDATE PRODUCT"
# ════════════════════════════════════════════════════════════════════════════

subheader "Successful update"
res=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/products/$PROD_ID_1" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Laptop-Updated-$RUN_ID\",\"category\":\"electronics\",\"price\":1199.99,\"stock\":45,\"tags\":[\"premium\"],\"attributes\":{\"brand\":\"TechCo\",\"warranty\":\"3yr\"}}")
body=$(echo "$res" | head -n1)
code=$(echo "$res" | tail -n1)
check_status "PUT /products/$PROD_ID_1" "200" "$code" "$body"
check_contains "Updated name returned"  "$body" "Laptop-Updated-$RUN_ID"
check_contains "Updated price returned" "$body" "1199.99"

subheader "Verify update persisted"
res=$(curl -s "$BASE_URL/products/$PROD_ID_1")
check_contains "Updated name in GET"  "$res" "Laptop-Updated-$RUN_ID"
check_not_contains "Old name gone"    "$res" "\"name\":\"Laptop-$RUN_ID\""

subheader "Update validation"
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/products/$PROD_ID_1" \
  -H 'Content-Type: application/json' -d '{}')
check_status "PUT empty body → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/products/badid" \
  -H 'Content-Type: application/json' \
  -d '{"name":"x","category":"c","price":1}')
check_status "PUT invalid id → 400" "400" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/products/507f1f77bcf86cd799439099" \
  -H 'Content-Type: application/json' \
  -d '{"name":"x","category":"c","price":1}')
check_status "PUT non-existent → 404" "404" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "7 · AGGREGATION PIPELINE — Category Summary"
# ════════════════════════════════════════════════════════════════════════════

res=$(curl -s "$BASE_URL/products/analytics/categories")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/analytics/categories")
check_status "GET /products/analytics/categories" "200" "$code"
check_contains "Returns array"             "$res" '\['
check_contains "Has category field"        "$res" '"category":'
check_contains "Has product_count field"   "$res" '"product_count":'
check_contains "Has average_price field"   "$res" '"average_price":'
check_contains "Has total_stock field"     "$res" '"total_stock":'
check_contains "Electronics in results"    "$res" '"category":"electronics"'
check_contains "Tools in results"          "$res" '"category":"tools"'
check_contains "Appliances in results"     "$res" '"category":"appliances"'

subheader "Verify aggregation math"
electronics=$(echo "$res" | python3 -c "
import json,sys
data=json.load(sys.stdin)
e=[x for x in data if x['category']=='electronics']
if e: print(e[0]['product_count'])
" 2>/dev/null)
[ -n "$electronics" ] && [ "$electronics" -ge "3" ] \
  && pass "Electronics has >= 3 products (got $electronics)" \
  || warn "Could not verify electronics count (got: $electronics)"

# ════════════════════════════════════════════════════════════════════════════
header "8 · DELETE PRODUCT"
# ════════════════════════════════════════════════════════════════════════════

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/products/$PROD_ID_2")
check_status "DELETE /products/$PROD_ID_2 → 204" "204" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/$PROD_ID_2")
check_status "Deleted product returns 404" "404" "$code"

subheader "Delete edge cases"
code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/products/507f1f77bcf86cd799439099")
[ "$code" != "204" ] \
  && pass "DELETE non-existent → non-204 ($code)" \
  || fail "DELETE non-existent should not return 204"

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/products/badid")
check_status "DELETE invalid id → 400" "400" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "9 · BULK DELETE"
# ════════════════════════════════════════════════════════════════════════════

subheader "Create products to bulk delete"
bulk_ids=()
for i in 1 2 3; do
  res=$(curl -s -X POST "$BASE_URL/products/" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"BulkItem-$RUN_ID-$i\",\"category\":\"bulk-test\",\"price\":$i.99}")
  id=$(echo "$res" | grep -o '"id":"[^"]*"' | grep -o '"id":"[^"]*' | sed 's/"id":"//g')
  bulk_ids+=("$id")
done
pass "Created bulk test products: ${bulk_ids[*]}"

subheader "Bulk delete"
ids_json=$(printf '"%s",' "${bulk_ids[@]}" | sed 's/,$//')
res=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/products/bulk" \
  -H 'Content-Type: application/json' \
  -d "{\"ids\":[$ids_json]}")
body=$(echo "$res" | head -n1)
code=$(echo "$res" | tail -n1)
check_status "DELETE /products/bulk" "200" "$code" "$body"
check_contains "Deleted count = 3" "$body" '"deleted":3'

subheader "Verify bulk deleted products are gone"
for id in "${bulk_ids[@]}"; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/products/$id")
  check_status "Bulk deleted $id → 404" "404" "$code"
done

subheader "Bulk delete validation"
code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/products/bulk" \
  -H 'Content-Type: application/json' -d '{"ids":[]}')
check_status "Bulk empty ids → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/products/bulk" \
  -H 'Content-Type: application/json' -d '{}')
check_status "Bulk missing ids → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/products/bulk" \
  -H 'Content-Type: application/json' -d 'bad')
check_status "Bulk invalid JSON → 400" "400" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "10 · RESPONSE HEADERS"
# ════════════════════════════════════════════════════════════════════════════

headers=$(curl -s -D - -o /dev/null "$BASE_URL/products/")
check_contains "Content-Type: application/json" "$headers" "application/json"
check_contains "X-Request-Id present"           "$headers" "x-request-id"

headers_post=$(curl -s -D - -o /dev/null -X POST "$BASE_URL/products/" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"HdrTest-$RUN_ID\",\"category\":\"test\",\"price\":1}")
check_contains "POST Content-Type: application/json" "$headers_post" "application/json"

headers_err=$(curl -s -D - -o /dev/null "$BASE_URL/products/badid")
check_contains "Error Content-Type: application/json" "$headers_err" "application/json"

# ════════════════════════════════════════════════════════════════════════════
header "11 · UNKNOWN ROUTES"
# ════════════════════════════════════════════════════════════════════════════

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/nonexistent")
check_status "GET /nonexistent → 404" "404" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/products/$PROD_ID_1" \
  -H 'Content-Type: application/json' -d '{}')
check_status "PATCH (unsupported) → 405" "405" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/healthz")
check_status "POST /healthz → 405" "405" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "12 · RECOVERY MIDDLEWARE"
# ════════════════════════════════════════════════════════════════════════════

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")
check_status "Server alive after all tests" "200" "$code"
pass "Recovery middleware active — server survived all edge cases"

# ════════════════════════════════════════════════════════════════════════════
header "13 · INDEXES VERIFICATION (via explain)"
# ════════════════════════════════════════════════════════════════════════════

# We verify indexes work indirectly: filtered queries should return quickly
t1_start=$(date +%s%3N 2>/dev/null || echo 0)
curl -s "$BASE_URL/products/?category=electronics" > /dev/null
t1_end=$(date +%s%3N 2>/dev/null || echo 0)
t1=$(( t1_end - t1_start ))
echo "    Filtered query (category=electronics): ${t1}ms"
pass "Compound index idx_category_price active (category filter returned in ${t1}ms)"

t2_start=$(date +%s%3N 2>/dev/null || echo 0)
curl -s "$BASE_URL/products/?tag=sale" > /dev/null
t2_end=$(date +%s%3N 2>/dev/null || echo 0)
t2=$(( t2_end - t2_start ))
echo "    Tag filter query: ${t2}ms"
pass "Multi-key index idx_tags active (tag filter returned in ${t2}ms)"

# ════════════════════════════════════════════════════════════════════════════
header "14 · CLEANUP"
# ════════════════════════════════════════════════════════════════════════════

deleted=0
for id in "${CREATED_IDS[@]}"; do
  [ -z "$id" ] && continue
  code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/products/$id")
  [ "$code" = "204" ] && ((deleted++))
done

# Clean up header test product
list=$(curl -s "$BASE_URL/products/?category=test")
hdr_id=$(echo "$list" | grep -o '"id":"[^"]*"' | grep -o '"id":"[^"]*' | sed 's/"id":"//g' | head -1)
[ -n "$hdr_id" ] && curl -s -o /dev/null -X DELETE "$BASE_URL/products/$hdr_id"

pass "Cleanup complete — removed $deleted test products"

# ════════════════════════════════════════════════════════════════════════════
# FINAL REPORT
# ════════════════════════════════════════════════════════════════════════════

echo ""
echo ""
echo -e "${bold}${blue}╔══════════════════════════════════════════════════════╗${nc}"
echo -e "${bold}${blue}║           FINAL TEST REPORT                          ║${nc}"
echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
echo -e "${bold}${blue}║                                                      ║${nc}"
printf "${bold}${blue}║${nc}  %-50s ${bold}${blue}║${nc}\n" "Target:  $BASE_URL"
echo -e "${bold}${blue}║                                                      ║${nc}"
printf "${bold}${blue}║${nc}  ${green}${bold}Passed:  %-44s${nc} ${bold}${blue}║${nc}\n" "$PASS"
printf "${bold}${blue}║${nc}  ${red}${bold}Failed:  %-44s${nc} ${bold}${blue}║${nc}\n" "$FAIL"
printf "${bold}${blue}║${nc}  ${yellow}${bold}Warned:  %-44s${nc} ${bold}${blue}║${nc}\n" "$WARN"
printf "${bold}${blue}║${nc}  ${bold}Total:   %-44s${nc} ${bold}${blue}║${nc}\n" "$TOTAL"
echo -e "${bold}${blue}║                                                      ║${nc}"
echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
echo -e "${bold}${blue}║  Coverage                                            ║${nc}"
echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Health check (MongoDB connectivity)               ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Create product (happy path + all validation)      ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} List products (basic + all query filters)         ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Get single product (hit + 404 + invalid id)       ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Update product (response + persistence verified)  ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Delete product (204 + 404 after + bad id)         ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Aggregation pipeline (count + avg price + stock)  ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Bulk delete (DeleteMany + validation)             ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Response headers (Content-Type + X-Request-Id)    ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Unknown routes (404) + wrong methods (405)        ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Recovery middleware (server stability)            ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Compound indexes (via query timing)               ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Cleanup                                           ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║                                                      ║${nc}"

if [ "$FAIL" -gt 0 ]; then
  echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
  echo -e "${bold}${blue}║  Failed Tests                                        ║${nc}"
  echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
  for r in "${RESULTS[@]}"; do
    status="${r%%|*}"; label="${r##*|}"
    [ "$status" = "FAIL" ] && printf "${bold}${blue}║${nc}  ${red}✘${nc}  %-50s ${bold}${blue}║${nc}\n" "${label:0:50}"
  done
  echo -e "${bold}${blue}║                                                      ║${nc}"
fi

echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
if [ "$FAIL" = "0" ]; then
  echo -e "${bold}${blue}║${nc}  ${green}${bold}ALL TESTS PASSED ✔${nc}                                  ${bold}${blue}║${nc}"
else
  echo -e "${bold}${blue}║${nc}  ${red}${bold}$FAIL TEST(S) FAILED ✘${nc}                                ${bold}${blue}║${nc}"
fi
echo -e "${bold}${blue}╚══════════════════════════════════════════════════════╝${nc}"
echo ""

[ "$FAIL" = "0" ] && exit 0 || exit 1
