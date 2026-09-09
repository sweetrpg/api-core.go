package constants

// Environment variable names
const (
	BIND_ADDRESS      = "BIND_ADDRESS"
	ENV               = "ENV"
	INGRESS_BASE_PATH = "INGRESS_BASE_PATH"
	INGRESS_HOST      = "INGRESS_HOST"
	INGRESS_SCHEMES   = "INGRESS_SCHEMES"
	PORT              = "PORT"
	RATE_LIMIT        = "RATE_LIMIT"

	// Per-client/IP rate-limit tiers, read by ratelimit.DefaultOptions. A limit is the
	// maximum number of requests allowed per its window (in seconds).
	RATE_LIMIT_CHEAP           = "RATE_LIMIT_CHEAP"
	RATE_LIMIT_CHEAP_WINDOW    = "RATE_LIMIT_CHEAP_WINDOW_SECONDS"
	RATE_LIMIT_STANDARD        = "RATE_LIMIT_STANDARD"
	RATE_LIMIT_STANDARD_WINDOW = "RATE_LIMIT_STANDARD_WINDOW_SECONDS"

	REDIS_DB        = "REDIS_DB"
	REDIS_HOST      = "REDIS_HOST"
	REDIS_PASS      = "REDIS_PASS"
	REDIS_PORT      = "REDIS_PORT"
	SENTRY_DEBUG    = "SENTRY_DEBUG"
	SENTRY_DSN      = "SENTRY_DSN"
	TRACING_NAME    = "TRACING_NAME"
	VERSION         = "VERSION"
	ZIPKIN_ENDPOINT = "ZIPKIN_ENDPOINT"
)

// Rate-limit tier defaults, applied when the matching env var is unset. These match
// catalog-api's validated values: a looser "cheap" tier for shallow /status routes and a
// stricter "standard" tier for data routes.
const (
	RateLimitCheapDefault          = 120
	RateLimitCheapWindowDefault    = 60
	RateLimitStandardDefault       = 30
	RateLimitStandardWindowDefault = 60
)

// Value constants
const (
	PageStartOption   = "start"
	PageLimitOption   = "limit"
	ErrorUnauthorized = "unauthorized"
	ErrorForbidden    = "forbidden"
	ErrorRateLimited  = "rate_limited"

	// ErrorRateLimitUnavailable is the error code returned with a 503 when the rate-limit
	// backend is unreachable (fail closed).
	ErrorRateLimitUnavailable = "rate_limit_unavailable"
)
