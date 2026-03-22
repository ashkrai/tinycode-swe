package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const DefaultTTL = 60 * time.Second

// ErrCacheMiss is returned when a key is not present in Redis.
var ErrCacheMiss = errors.New("cache miss")

// Client wraps a go-redis client with typed helpers.
type Client struct {
	rdb *redis.Client
}

// New creates a Redis client from the given URL (redis://:pass@host:port/db).
func New(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// Close closes the underlying Redis connection.
func (c *Client) Close() error { return c.rdb.Close() }

// Ping checks connectivity.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// GetJSON fetches key and unmarshals JSON into dst.
// Returns ErrCacheMiss when the key does not exist.
func (c *Client) GetJSON(ctx context.Context, key string, dst any) error {
	val, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("redis GET %s: %w", key, err)
	}
	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return fmt.Errorf("unmarshal cache value for %s: %w", key, err)
	}
	return nil
}

// SetJSON marshals src to JSON and stores it with the given TTL.
func (c *Client) SetJSON(ctx context.Context, key string, src any, ttl time.Duration) error {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("marshal cache value for %s: %w", key, err)
	}
	if err := c.rdb.Set(ctx, key, b, ttl).Err(); err != nil {
		return fmt.Errorf("redis SET %s: %w", key, err)
	}
	return nil
}

// Delete removes one or more keys from Redis.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis DEL: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sliding-window rate limiter
// ---------------------------------------------------------------------------

// RateLimitResult holds the outcome of an Allow check.
type RateLimitResult struct {
	Allowed    bool
	Count      int64
	RetryAfter time.Duration
}

// Allow implements a sliding-window rate limiter using INCR + EXPIRE.
//
// key    — usually "ratelimit:<ip>"
// limit  — max requests allowed in the window
// window — duration of the window (e.g. 1 minute)
//
// Algorithm:
//  1. INCR key  → new count (atomically creates key on first call)
//  2. If count == 1, set EXPIRE so the key auto-resets after the window.
//  3. If count > limit → deny and return TTL as RetryAfter.
func (c *Client) Allow(ctx context.Context, key string, limit int64, window time.Duration) (RateLimitResult, error) {
	pipe := c.rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	// We always call EXPIRE; it is a no-op if the key already has a TTL that
	// was set in step 1 of a previous call — we use NX variant to set only if
	// no expiry exists yet (requires Redis ≥ 7.0).  For broader compatibility
	// we use a plain EXPIRE which resets the window on every request inside the
	// same window — an acceptable trade-off for simplicity.
	ttlCmd := pipe.TTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return RateLimitResult{}, fmt.Errorf("rate limit pipeline: %w", err)
	}

	count := incrCmd.Val()

	// Set expiry only on the first increment (TTL == -1 means no expiry set).
	if ttlCmd.Val() == -1 {
		if err := c.rdb.Expire(ctx, key, window).Err(); err != nil {
			return RateLimitResult{}, fmt.Errorf("rate limit expire: %w", err)
		}
	}

	if count > limit {
		// Return the remaining TTL so callers can set Retry-After.
		remaining, err := c.rdb.TTL(ctx, key).Result()
		if err != nil {
			remaining = window
		}
		return RateLimitResult{Allowed: false, Count: count, RetryAfter: remaining}, nil
	}

	return RateLimitResult{Allowed: true, Count: count}, nil
}
