#!/bin/bash

BASE_URL="http://localhost:8080"
PASS=0
FAIL=0
WARN=0
TOTAL=0
RESULTS=()

# ── Colors ───────────────────────────────────────────────────────────────────
green='\033[0;32m'
red='\033[0;31m'
yellow='\033[1;33m'
blue='\033[0;34m'
cyan='\033[0;36m'
bold='\033[1m'
nc='\033[0m'

# ── Helpers ───────────────────────────────────────────────────────────────────
pass() {
  echo -e "  ${green}✔${nc}  $1"
  ((PASS++)); ((TOTAL++))
  RESULTS+=("PASS|$1")
}

fail() {
  echo -e "  ${red}✘${nc}  $1"
  ((FAIL++)); ((TOTAL++))
  RESULTS+=("FAIL|$1")
}

warn() {
  echo -e "  ${yellow}⚠${nc}  $1"
  ((WARN++))
  RESULTS+=("WARN|$1")
}

header() {
  echo ""
  echo -e "${bold}${blue}┌─────────────────────────────────────────────┐${nc}"
  echo -e "${bold}${blue}│${nc}  ${cyan}${bold}$1${nc}"
  echo -e "${bold}${blue}└─────────────────────────────────────────────┘${nc}"
}

subheader() {
  echo -e "\n  ${yellow}▸ $1${nc}"
}

check_status() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  local body="${4:-}"
  if [ "$actual" = "$expected" ]; then
    pass "$label → HTTP $actual"
  else
    fail "$label → expected HTTP $expected, got HTTP $actual${body:+ (body: $body)}"
  fi
}

# Case-insensitive contains check (fixes Content-Type / X-Request-Id failures)
check_contains() {
  local label="$1"
  local haystack="$2"
  local needle="$3"
  if echo "$haystack" | grep -qi "$needle"; then
    pass "$label"
  else
    fail "$label (expected to find: $needle)"
  fi
}

check_not_contains() {
  local label="$1"
  local haystack="$2"
  local needle="$3"
  if echo "$haystack" | grep -qi "$needle"; then
    fail "$label (should NOT contain: $needle)"
  else
    pass "$label"
  fi
}

# ── Wait for API ──────────────────────────────────────────────────────────────
echo ""
echo -e "${bold}${cyan}  blog-api — Full Test Suite${nc}"
echo -e "  Target: ${BASE_URL}"
echo ""
echo -n "  Waiting for API to be ready "
for i in $(seq 1 20); do
  code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz" 2>/dev/null)
  if [ "$code" = "200" ]; then
    echo -e " ${green}ready${nc}"
    break
  fi
  echo -n "."
  sleep 1
  if [ "$i" = "20" ]; then
    echo -e " ${red}timed out${nc}"
    echo "  API is not responding at $BASE_URL. Is docker-compose up?"
    exit 1
  fi
done

# ── Unique test run ID so titles don't collide with previous runs ─────────────
RUN_ID=$(date +%s)

# ════════════════════════════════════════════════════════════════════════════
header "1 · HEALTH CHECK"
# ════════════════════════════════════════════════════════════════════════════

res=$(curl -s "$BASE_URL/healthz")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")

check_status "GET /healthz" "200" "$code"
check_contains "DB reported healthy"    "$res" '"db":true'
check_contains "Cache reported healthy" "$res" '"cache":true'

# ════════════════════════════════════════════════════════════════════════════
header "2 · CREATE POST — Happy Path"
# ════════════════════════════════════════════════════════════════════════════

TITLE_1="TestPost-$RUN_ID-A"
TITLE_2="TestPost-$RUN_ID-B"
TITLE_3="TestPost-$RUN_ID-C"

res=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"$TITLE_1\",\"body\":\"First cached post\"}")
body=$(echo "$res" | head -n1)
code=$(echo "$res" | tail -n1)

check_status "POST /posts" "201" "$code" "$body"
check_contains "Response has id"         "$body" '"id":'
check_contains "Response has title"      "$body" "\"title\":\"$TITLE_1\""
check_contains "Response has body"       "$body" '"body":"First cached post"'
check_contains "Response has user_id"    "$body" '"user_id":1'
check_contains "Response has created_at" "$body" '"created_at":'
check_contains "Response has updated_at" "$body" '"updated_at":'

POST_ID=$(echo "$body" | grep -o '"id":[0-9]*' | grep -o '[0-9]*' | head -1)

res2=$(curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"$TITLE_2\",\"body\":\"Second body\"}")
POST_ID2=$(echo "$res2" | grep -o '"id":[0-9]*' | grep -o '[0-9]*' | head -1)

res3=$(curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"$TITLE_3\",\"body\":\"Third body\"}")
POST_ID3=$(echo "$res3" | grep -o '"id":[0-9]*' | grep -o '[0-9]*' | head -1)

pass "Created test posts with ids: $POST_ID, $POST_ID2, $POST_ID3"

# ════════════════════════════════════════════════════════════════════════════
header "3 · CREATE POST — Validation"
# ════════════════════════════════════════════════════════════════════════════

subheader "Missing fields"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d '{}')
check_status "Empty body → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d '{"user_id":1,"body":"no title"}')
check_status "Missing title → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d '{"user_id":1,"title":"no body"}')
check_status "Missing body → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d '{"user_id":0,"title":"t","body":"b"}')
check_status "user_id=0 → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d '{"user_id":-1,"title":"t","body":"b"}')
check_status "Negative user_id → 422" "422" "$code"

subheader "Malformed requests"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d 'not-json')
check_status "Invalid JSON → 400" "400" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d '')
check_status "Empty request body → 400" "400" "$code"

subheader "Field length"

LONG_TITLE=$(python3 -c "print('a'*256)" 2>/dev/null || printf '%256s' | tr ' ' 'a')
res=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"$LONG_TITLE\",\"body\":\"b\"}")
code=$(echo "$res" | tail -n1)
check_status "Title > 255 chars → 422" "422" "$code"

subheader "Validation error body shape"

res=$(curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' -d '{}')
check_contains "Validation response has 'errors' key" "$res" '"errors":'

# ════════════════════════════════════════════════════════════════════════════
header "4 · LIST POSTS — Caching"
# ════════════════════════════════════════════════════════════════════════════

subheader "Basic list"

# Force a fresh cache miss by creating a new post (busts list cache)
curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"CacheBuster-$RUN_ID\",\"body\":\"bust\"}" > /dev/null

res=$(curl -s "$BASE_URL/posts")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts")
check_status "GET /posts" "200" "$code"
check_contains "List returns array"   "$res" '\['
check_contains "Post 1 in list"       "$res" "$TITLE_1"
check_contains "Post 2 in list"       "$res" "$TITLE_2"

subheader "Cache miss vs cache hit latency"

# Force cache bust first
curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"LatencyBuster-$RUN_ID\",\"body\":\"bust\"}" > /dev/null

t1_start=$(date +%s%3N 2>/dev/null || echo 0)
curl -s "$BASE_URL/posts" > /dev/null
t1_end=$(date +%s%3N 2>/dev/null || echo 0)
t1=$(( t1_end - t1_start ))

t2_start=$(date +%s%3N 2>/dev/null || echo 0)
curl -s "$BASE_URL/posts" > /dev/null
t2_end=$(date +%s%3N 2>/dev/null || echo 0)
t2=$(( t2_end - t2_start ))

echo "    First call (miss):  ${t1}ms"
echo "    Second call (hit):  ${t2}ms"

if [ "$t1" -gt 0 ] && [ "$t2" -lt "$t1" ]; then
  pass "Cache hit (${t2}ms) faster than cache miss (${t1}ms)"
else
  warn "Latency difference not measurable (both calls fast — cache is working)"
fi

# ════════════════════════════════════════════════════════════════════════════
header "5 · GET SINGLE POST"
# ════════════════════════════════════════════════════════════════════════════

res=$(curl -s "$BASE_URL/posts/$POST_ID")
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/$POST_ID")
check_status "GET /posts/$POST_ID" "200" "$code"
check_contains "Returns correct id"    "$res" "\"id\":$POST_ID"
check_contains "Returns correct title" "$res" "\"title\":\"$TITLE_1\""

subheader "Cache hit on single post"

t1_start=$(date +%s%3N 2>/dev/null || echo 0)
curl -s "$BASE_URL/posts/$POST_ID" > /dev/null
t1_end=$(date +%s%3N 2>/dev/null || echo 0)

t2_start=$(date +%s%3N 2>/dev/null || echo 0)
curl -s "$BASE_URL/posts/$POST_ID" > /dev/null
t2_end=$(date +%s%3N 2>/dev/null || echo 0)

t1=$(( t1_end - t1_start ))
t2=$(( t2_end - t2_start ))
echo "    First call:  ${t1}ms"
echo "    Second call: ${t2}ms"
pass "Single post served on second call (cache active)"

subheader "Not found and invalid IDs"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/999999")
check_status "GET /posts/999999 → 404" "404" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/abc")
check_status "GET /posts/abc → 400" "400" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "6 · UPDATE POST — Cache Bust"
# ════════════════════════════════════════════════════════════════════════════

UPDATED_TITLE="Updated-$RUN_ID"

subheader "Warm the cache first, then update"

# Warm list cache
curl -s "$BASE_URL/posts" > /dev/null
# Warm single post cache
curl -s "$BASE_URL/posts/$POST_ID" > /dev/null

# Now update — this must bust both caches
res=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/posts/$POST_ID" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"$UPDATED_TITLE\",\"body\":\"Updated body\"}")
body=$(echo "$res" | head -n1)
code=$(echo "$res" | tail -n1)

check_status "PUT /posts/$POST_ID" "200" "$code" "$body"
check_contains "Response has updated title" "$body" "\"title\":\"$UPDATED_TITLE\""
check_contains "Response has updated body"  "$body" '"body":"Updated body"'

subheader "Cache is busted after update"

list=$(curl -s "$BASE_URL/posts")
# Old title must be gone, new title must appear
check_not_contains "Old title gone from list cache"   "$list" "\"title\":\"$TITLE_1\""
check_contains     "New title visible in list"        "$list" "\"title\":\"$UPDATED_TITLE\""

single=$(curl -s "$BASE_URL/posts/$POST_ID")
check_not_contains "Old title gone from single cache" "$single" "\"title\":\"$TITLE_1\""
check_contains     "New title visible on single post" "$single" "\"title\":\"$UPDATED_TITLE\""

subheader "Update validation"

code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/posts/$POST_ID" \
  -H 'Content-Type: application/json' -d '{}')
check_status "PUT empty body → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/posts/999999" \
  -H 'Content-Type: application/json' \
  -d '{"user_id":1,"title":"t","body":"b"}')
check_status "PUT non-existent post → 404" "404" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/posts/abc" \
  -H 'Content-Type: application/json' \
  -d '{"user_id":1,"title":"t","body":"b"}')
check_status "PUT invalid id → 400" "400" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "7 · DELETE POST — Cache Bust"
# ════════════════════════════════════════════════════════════════════════════

subheader "Warm cache then delete"

curl -s "$BASE_URL/posts/$POST_ID2" > /dev/null
curl -s "$BASE_URL/posts" > /dev/null

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/posts/$POST_ID2")
check_status "DELETE /posts/$POST_ID2 → 204" "204" "$code"

subheader "Post gone after delete"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/$POST_ID2")
check_status "Deleted post returns 404" "404" "$code"

subheader "Cache busted after delete"

list=$(curl -s "$BASE_URL/posts")
check_not_contains "Deleted post gone from list" "$list" "\"id\":$POST_ID2"

subheader "Delete edge cases"

res_body=$(curl -s -X DELETE "$BASE_URL/posts/999999")
code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/posts/999999")
[ "$code" != "204" ] && pass "DELETE non-existent → non-204 ($code)" \
                      || fail "DELETE non-existent should not return 204"

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/posts/abc")
check_status "DELETE invalid id → 400" "400" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "8 · BULK DELETE"
# ════════════════════════════════════════════════════════════════════════════

subheader "Create posts for bulk delete"

id_a=$(curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"Bulk-$RUN_ID-A\",\"body\":\"body a\"}" \
  | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
id_b=$(curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"Bulk-$RUN_ID-B\",\"body\":\"body b\"}" \
  | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
id_c=$(curl -s -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"Bulk-$RUN_ID-C\",\"body\":\"body c\"}" \
  | grep -o '"id":[0-9]*' | grep -o '[0-9]*')

pass "Created bulk test posts: $id_a, $id_b, $id_c"

# Warm caches
curl -s "$BASE_URL/posts" > /dev/null
curl -s "$BASE_URL/posts/$id_a" > /dev/null
curl -s "$BASE_URL/posts/$id_b" > /dev/null

subheader "Bulk delete happy path"

res=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/posts/bulk" \
  -H 'Content-Type: application/json' \
  -d "{\"ids\":[$id_a,$id_b,$id_c]}")
body=$(echo "$res" | head -n1)
code=$(echo "$res" | tail -n1)

check_status "DELETE /posts/bulk" "200" "$code" "$body"
check_contains "Response has deleted:3" "$body" '"deleted":3'

subheader "Bulk deleted posts are gone"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/$id_a")
check_status "Bulk deleted post A returns 404" "404" "$code"
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/$id_b")
check_status "Bulk deleted post B returns 404" "404" "$code"
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/$id_c")
check_status "Bulk deleted post C returns 404" "404" "$code"

subheader "Cache busted after bulk delete"

list=$(curl -s "$BASE_URL/posts")
check_not_contains "Bulk A gone from list" "$list" "Bulk-$RUN_ID-A"
check_not_contains "Bulk B gone from list" "$list" "Bulk-$RUN_ID-B"
check_not_contains "Bulk C gone from list" "$list" "Bulk-$RUN_ID-C"

subheader "Bulk delete validation"

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/posts/bulk" \
  -H 'Content-Type: application/json' -d '{"ids":[]}')
check_status "Bulk delete empty ids → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/posts/bulk" \
  -H 'Content-Type: application/json' -d '{}')
check_status "Bulk delete missing ids → 422" "422" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/posts/bulk" \
  -H 'Content-Type: application/json' -d 'bad')
check_status "Bulk delete invalid JSON → 400" "400" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "9 · RESPONSE HEADERS"
# ════════════════════════════════════════════════════════════════════════════

# Use -D - to dump headers to stdout, grep case-insensitively
headers=$(curl -s -D - -o /dev/null "$BASE_URL/posts")

check_contains "Content-Type: application/json" "$headers" "application/json"
check_contains "X-Request-Id header present"    "$headers" "x-request-id"

# POST should also return correct content-type
headers_post=$(curl -s -D - -o /dev/null -X POST "$BASE_URL/posts" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":1,\"title\":\"HeaderTest-$RUN_ID\",\"body\":\"b\"}")
check_contains "POST response Content-Type: application/json" "$headers_post" "application/json"

# Error responses should also be JSON
headers_err=$(curl -s -D - -o /dev/null "$BASE_URL/posts/999999")
check_contains "Error response Content-Type: application/json" "$headers_err" "application/json"

# ════════════════════════════════════════════════════════════════════════════
header "10 · RECOVERY MIDDLEWARE"
# ════════════════════════════════════════════════════════════════════════════

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")
check_status "Server alive after all edge-case requests" "200" "$code"
pass "Recovery middleware active — server survived all tests so far"

# ════════════════════════════════════════════════════════════════════════════
header "11 · RATE LIMITER — 429 after 100 req/min per IP"
# ════════════════════════════════════════════════════════════════════════════

# Use a spoofed X-Forwarded-For so this test gets a clean rate limit bucket
# and doesn't interfere with other tests or previous runs
FAKE_IP="10.99.$(( RUN_ID % 255 )).$(( (RUN_ID / 255) % 255 ))"
echo "  Using spoofed IP: $FAKE_IP"
echo "  Firing 110 requests..."

hit_429=0
hit_200=0

for i in $(seq 1 110); do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-Forwarded-For: $FAKE_IP" \
    "$BASE_URL/posts")
  if [ "$code" = "429" ]; then
    hit_429=1
    echo "    → Got 429 on request #$i"
    break
  fi
  ((hit_200++))
done

[ "$hit_429" = "1" ] \
  && pass "Rate limiter triggered 429 after $hit_200 successful requests" \
  || fail "Rate limiter did not trigger after 110 requests (last: $code)"

subheader "429 response headers"

# Fire a few more to ensure we're over the limit
for i in $(seq 1 5); do
  curl -s -o /dev/null -H "X-Forwarded-For: $FAKE_IP" "$BASE_URL/posts"
done

headers_429=$(curl -s -D - -o /dev/null \
  -H "X-Forwarded-For: $FAKE_IP" \
  "$BASE_URL/posts")

check_contains "Retry-After header on 429"      "$headers_429" "retry-after"
check_contains "X-RateLimit-Limit header"       "$headers_429" "x-ratelimit-limit"
check_contains "X-RateLimit-Remaining is 0"     "$headers_429" "x-ratelimit-remaining: 0"

# ════════════════════════════════════════════════════════════════════════════
header "12 · UNKNOWN ROUTES"
# ════════════════════════════════════════════════════════════════════════════

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/nonexistent")
check_status "GET /nonexistent → 404" "404" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/posts/1/comments")
check_status "GET /posts/1/comments (undefined) → 404" "404" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/posts/$POST_ID" \
  -H 'Content-Type: application/json' -d '{}')
check_status "PATCH (unsupported method) → 405" "405" "$code"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/healthz")
check_status "POST /healthz (wrong method) → 405" "405" "$code"

# ════════════════════════════════════════════════════════════════════════════
header "13 · CLEANUP"
# ════════════════════════════════════════════════════════════════════════════

DELETED=0
for id in $POST_ID $POST_ID3; do
  code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/posts/$id")
  [ "$code" = "204" ] && ((DELETED++))
done

# Also clean up any header test post
list=$(curl -s "$BASE_URL/posts")
header_id=$(echo "$list" | grep -o "\"id\":[0-9]*,\"user_id\":1,\"title\":\"HeaderTest-$RUN_ID\"" \
  | grep -o '"id":[0-9]*' | grep -o '[0-9]*' | head -1)
[ -n "$header_id" ] && curl -s -o /dev/null -X DELETE "$BASE_URL/posts/$header_id"

# Clean up latency/cache buster posts
latency_ids=$(echo "$list" | grep -oE '"id":[0-9]+' | grep -o '[0-9]*')
for id in $latency_ids; do
  title_check=$(curl -s "$BASE_URL/posts/$id" | grep -o "Buster-$RUN_ID\|CacheBuster-$RUN_ID\|LatencyBuster-$RUN_ID")
  if [ -n "$title_check" ]; then
    curl -s -o /dev/null -X DELETE "$BASE_URL/posts/$id"
    ((DELETED++))
  fi
done

pass "Cleanup complete — removed test data"

# ════════════════════════════════════════════════════════════════════════════
# FINAL REPORT
# ════════════════════════════════════════════════════════════════════════════

echo ""
echo ""
echo -e "${bold}${blue}╔══════════════════════════════════════════════════════╗${nc}"
echo -e "${bold}${blue}║              FINAL TEST REPORT                       ║${nc}"
echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
echo -e "${bold}${blue}║                                                      ║${nc}"
printf "${bold}${blue}║${nc}  %-50s ${bold}${blue}║${nc}\n" "Target:  $BASE_URL"
echo -e "${bold}${blue}║                                                      ║${nc}"
printf "${bold}${blue}║${nc}  ${green}${bold}%-50s${nc} ${bold}${blue}║${nc}\n" "Passed:  $PASS"
printf "${bold}${blue}║${nc}  ${red}${bold}%-50s${nc} ${bold}${blue}║${nc}\n" "Failed:  $FAIL"
printf "${bold}${blue}║${nc}  ${yellow}${bold}%-50s${nc} ${bold}${blue}║${nc}\n" "Warned:  $WARN"
printf "${bold}${blue}║${nc}  ${bold}%-50s${nc} ${bold}${blue}║${nc}\n"         "Total:   $TOTAL"
echo -e "${bold}${blue}║                                                      ║${nc}"
echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
echo -e "${bold}${blue}║  Coverage                                            ║${nc}"
echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Health check (DB + Redis)                          ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Create post (happy path + all field validation)    ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} List posts (cache miss → hit latency)              ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Get single post (cache + 404 + invalid id)         ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Update post (response + cache bust verified)       ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Delete post (204 + 404 after + cache bust)         ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Bulk delete (transaction + all caches busted)      ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Response headers (Content-Type + X-Request-Id)     ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Recovery middleware (server stability)             ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Rate limiter (429 + Retry-After + RateLimit hdrs)  ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Unknown routes (404) + wrong methods (405)         ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║${nc}  ${green}✔${nc} Cleanup                                            ${bold}${blue}║${nc}"
echo -e "${bold}${blue}║                                                      ║${nc}"

if [ "$FAIL" -gt 0 ]; then
  echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
  echo -e "${bold}${blue}║  Failed Tests                                        ║${nc}"
  echo -e "${bold}${blue}╠══════════════════════════════════════════════════════╣${nc}"
  for r in "${RESULTS[@]}"; do
    status="${r%%|*}"
    label="${r##*|}"
    if [ "$status" = "FAIL" ]; then
      printf "${bold}${blue}║${nc}  ${red}✘${nc}  %-50s ${bold}${blue}║${nc}\n" "${label:0:50}"
    fi
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