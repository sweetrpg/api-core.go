// Package ratelimit implements the platform-wide, Redis-backed, per-client, per-route-tier
// request rate limiter for sweetrpg Go HTTP services. Counters live in Redis (INCR + EXPIRE)
// rather than per-pod memory, so a client's limit is consistent across a service's replicas
// instead of being N times looser depending on how many pods are up. The limiter fails closed:
// if Redis is unreachable, requests subject to limiting are rejected with 503 rather than
// served as if unlimited.
package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"

	"github.com/sweetrpg/api-core.go/constants"
	"github.com/sweetrpg/api-core.go/util"
	"github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/common.go/logging"
)

// Tier names. Routes under /status/ use the looser cheap tier; everything else uses standard.
const (
	TierCheap    = "cheap"
	TierStandard = "standard"
)

// Tier describes a rate-limit budget: at most Limit requests per Window seconds.
type Tier struct {
	Limit  int
	Window int
}

// Limiter enforces per-client request limits per tier, backed by a Redis connection pool.
type Limiter struct {
	pool  *redis.Pool
	tiers map[string]Tier
}

// New builds a Limiter using the given Redis pool and tier definitions. Callers should pass at
// least a "standard" and "cheap" tier; Allow falls back to "standard" for an unknown tier name.
func New(pool *redis.Pool, tiers map[string]Tier) *Limiter {
	return &Limiter{pool: pool, tiers: tiers}
}

// Allow increments the request counter for clientKey within the named tier and reports whether
// the request is within that tier's limit. A non-nil error means the Redis backend was
// unreachable - callers must treat that as fail-closed (reject the request), not fail-open.
func (l *Limiter) Allow(ctx context.Context, clientKey, tierName string) (bool, error) {
	tier, ok := l.tiers[tierName]
	if !ok {
		tier = l.tiers[TierStandard]
	}

	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("ratelimit: get redis connection: %w", err)
	}
	defer func() { _ = conn.Close() }()

	key := fmt.Sprintf("ratelimit:%s:%s", tierName, clientKey)
	count, err := redis.Int(conn.Do("INCR", key))
	if err != nil {
		return false, fmt.Errorf("ratelimit: incr: %w", err)
	}
	if count == 1 {
		if _, err := conn.Do("EXPIRE", key, tier.Window); err != nil {
			return false, fmt.Errorf("ratelimit: expire: %w", err)
		}
	}

	return count <= tier.Limit, nil
}

// Ping verifies the Redis backend is reachable, for use in startup/readiness checks.
func Ping(ctx context.Context, pool *redis.Pool) error {
	conn, err := pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	_, err = conn.Do("PING")
	return err
}

// TierFor groups routes into the cheap tier (shallow /status endpoints) and the standard tier
// (everything else), keyed by route cost rather than one entry per resource.
func TierFor(path string) string {
	if strings.HasPrefix(path, "/status/") {
		return TierCheap
	}
	return TierStandard
}

// ClientKey identifies the caller for per-client limiting: the API key if the client sent an
// X-API-Key header, otherwise the client IP. It does not introduce an auth mechanism - it keys
// off whatever identity already exists on the request. The key:/ip: prefixes keep a client
// that sends an API key and one that does not from colliding on the same source address.
func ClientKey(c *gin.Context) string {
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return "key:" + apiKey
	}
	return "ip:" + c.ClientIP()
}

// Options configures the middleware's tier budgets.
type Options struct {
	Cheap    Tier
	Standard Tier
}

// DefaultOptions reads the tier budgets from the standard env vars
// (RATE_LIMIT_CHEAP / RATE_LIMIT_CHEAP_WINDOW_SECONDS / RATE_LIMIT_STANDARD /
// RATE_LIMIT_STANDARD_WINDOW_SECONDS), falling back to the catalog-api-validated defaults.
func DefaultOptions() Options {
	return Options{
		Cheap: Tier{
			Limit:  util.GetEnvInt(constants.RATE_LIMIT_CHEAP, constants.RateLimitCheapDefault),
			Window: util.GetEnvInt(constants.RATE_LIMIT_CHEAP_WINDOW, constants.RateLimitCheapWindowDefault),
		},
		Standard: Tier{
			Limit:  util.GetEnvInt(constants.RATE_LIMIT_STANDARD, constants.RateLimitStandardDefault),
			Window: util.GetEnvInt(constants.RATE_LIMIT_STANDARD_WINDOW, constants.RateLimitStandardWindowDefault),
		},
	}
}

// Middleware returns a gin handler that enforces per-client, per-tier limits backed by the
// given Redis pool. It aborts with 503 (ErrorRateLimitUnavailable) when the pool is nil or the
// backend is unreachable, and with 429 (ErrorRateLimited) when a client is over its tier limit.
func Middleware(pool *redis.Pool, opts Options) gin.HandlerFunc {
	limiter := New(pool, map[string]Tier{
		TierCheap:    opts.Cheap,
		TierStandard: opts.Standard,
	})

	return func(c *gin.Context) {
		if pool == nil {
			logging.Logger.Error("Rate-limit store not configured; rejecting request (fail closed)")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, vo.ErrorVO{
				Error:   constants.ErrorRateLimitUnavailable,
				Message: "Rate limiting is temporarily unavailable",
			})
			return
		}

		tier := TierFor(c.Request.URL.Path)
		clientKey := ClientKey(c)

		allowed, err := limiter.Allow(c.Request.Context(), clientKey, tier)
		if err != nil {
			logging.Logger.Error("Rate-limit backend unreachable; rejecting request (fail closed)", "error", err.Error())
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, vo.ErrorVO{
				Error:   constants.ErrorRateLimitUnavailable,
				Message: "Rate limiting is temporarily unavailable",
			})
			return
		}
		if !allowed {
			logging.Logger.Warn("Rate limit exceeded", "client", clientKey, "tier", tier, "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, vo.ErrorVO{
				Error:   constants.ErrorRateLimited,
				Message: "Limit exceeded",
			})
			return
		}
		c.Next()
	}
}
