// Package ratelimit implements the platform-wide, Redis-backed, per-client, per-route-tier
// request rate limiter for sweetrpg Go HTTP services. Counters live in Redis (INCR + EXPIRE)
// rather than per-pod memory, so a client's limit is consistent across a service's replicas
// instead of being N times looser depending on how many pods are up.
//
// If the counter store is unreachable, standard-tier routes fail closed (503) rather than being
// served as if unlimited, while /status/* routes fail open so a cache fault cannot fail the
// liveness and readiness probes that call them (PADR-0036). Every store error is logged at ERROR
// and counted in ratelimit_backend_errors_total{tier}. Every 429 carries a Retry-After header.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

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

// backendErrors counts failures to reach or use the counter store, by tier, whether the request
// was then let through (cheap) or rejected (standard). Alert on it: a failing store no longer
// shows up as failing probes.
var backendErrors = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "ratelimit_backend_errors_total",
	Help: "Rate-limit counter store errors, by tier.",
}, []string{"tier"})

func init() {
	// Export both series at zero so an alert can tell "no errors" from "no data".
	backendErrors.WithLabelValues(TierCheap)
	backendErrors.WithLabelValues(TierStandard)
}

var errNoPool = errors.New("ratelimit: no redis pool configured")

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

// Result is the outcome of one limit check. RetryAfter is how long until the client's window
// resets; it is only meaningful when Allowed is false, and is at least one second.
type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Allow increments the request counter for clientKey within the named tier and reports whether
// the request is within that tier's limit. A non-nil error means the Redis backend was
// unreachable; Middleware decides per tier whether that fails open or closed.
func (l *Limiter) Allow(ctx context.Context, clientKey, tierName string) (bool, error) {
	res, err := l.Check(ctx, clientKey, tierName)
	return res.Allowed, err
}

// Check is Allow plus, for a rejected request, how long the client should wait.
func (l *Limiter) Check(ctx context.Context, clientKey, tierName string) (Result, error) {
	tier, ok := l.tiers[tierName]
	if !ok {
		tier = l.tiers[TierStandard]
	}

	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit: get redis connection: %w", err)
	}
	defer func() { _ = conn.Close() }()

	key := fmt.Sprintf("ratelimit:%s:%s", tierName, clientKey)
	count, err := redis.Int(conn.Do("INCR", key))
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit: incr: %w", err)
	}
	if count == 1 {
		if _, err := conn.Do("EXPIRE", key, tier.Window); err != nil {
			return Result{}, fmt.Errorf("ratelimit: expire: %w", err)
		}
	}
	if count <= tier.Limit {
		return Result{Allowed: true}, nil
	}

	retry, err := retryAfter(conn, key, tier.Window)
	if err != nil {
		return Result{}, err
	}
	return Result{RetryAfter: retry}, nil
}

// retryAfter is the time left on the counter's window, rounded up to whole seconds and never
// under one. A counter with no expiry (the process died between INCR and EXPIRE) would block the
// client forever, so it is given a fresh window and reports the full window.
func retryAfter(conn redis.Conn, key string, window int) (time.Duration, error) {
	ms, err := redis.Int64(conn.Do("PTTL", key))
	if err != nil {
		return 0, fmt.Errorf("ratelimit: pttl: %w", err)
	}
	if ms < 0 {
		if _, err := conn.Do("EXPIRE", key, window); err != nil {
			return 0, fmt.Errorf("ratelimit: expire: %w", err)
		}
		return time.Duration(window) * time.Second, nil
	}
	secs := int64(math.Ceil(float64(ms) / 1000))
	return time.Duration(max(secs, 1)) * time.Second, nil
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
// given Redis pool. It aborts with 429 (ErrorRateLimited, plus Retry-After) when a client is over
// its tier limit. When the pool is nil or the store errors it logs, counts, and then either lets
// the request through (/status/*, so probes survive a cache fault) or aborts with 503
// (ErrorRateLimitUnavailable) for every other route.
func Middleware(pool *redis.Pool, opts Options) gin.HandlerFunc {
	limiter := New(pool, map[string]Tier{
		TierCheap:    opts.Cheap,
		TierStandard: opts.Standard,
	})

	return func(c *gin.Context) {
		tier := TierFor(c.Request.URL.Path)
		if pool == nil {
			storeUnavailable(c, tier, errNoPool)
			return
		}

		clientKey := ClientKey(c)
		res, err := limiter.Check(c.Request.Context(), clientKey, tier)
		if err != nil {
			storeUnavailable(c, tier, err)
			return
		}
		if !res.Allowed {
			logging.Logger.Warn("Rate limit exceeded", "client", clientKey, "tier", tier, "path", c.Request.URL.Path)
			c.Header("Retry-After", strconv.Itoa(int(res.RetryAfter/time.Second)))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, vo.ErrorVO{
				Error:   constants.ErrorRateLimited,
				Message: "Limit exceeded",
			})
			return
		}
		c.Next()
	}
}

// storeUnavailable handles one counter-store failure: it is logged once at ERROR and counted,
// then /status/* (the cheap tier) continues and every other route is rejected.
func storeUnavailable(c *gin.Context, tier string, err error) {
	backendErrors.WithLabelValues(tier).Inc()
	if tier == TierCheap {
		logging.Logger.Error("Rate-limit store unavailable; allowing request (fail open)", "tier", tier, "path", c.Request.URL.Path, "error", err.Error())
		c.Next()
		return
	}
	logging.Logger.Error("Rate-limit store unavailable; rejecting request (fail closed)", "tier", tier, "path", c.Request.URL.Path, "error", err.Error())
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, vo.ErrorVO{
		Error:   constants.ErrorRateLimitUnavailable,
		Message: "Rate limiting is temporarily unavailable",
	})
}
