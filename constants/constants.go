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
	REDIS_DB          = "REDIS_DB"
	REDIS_HOST        = "REDIS_HOST"
	REDIS_PASS        = "REDIS_PASS"
	REDIS_PORT        = "REDIS_PORT"
	SENTRY_DEBUG      = "SENTRY_DEBUG"
	SENTRY_DSN        = "SENTRY_DSN"
	TRACING_NAME      = "TRACING_NAME"
	VERSION           = "VERSION"
	ZIPKIN_ENDPOINT   = "ZIPKIN_ENDPOINT"
)

// Value constants
const (
	PageStartOption   = "start"
	PageLimitOption   = "limit"
	ErrorUnauthorized = "unauthorized"
	ErrorForbidden    = "forbidden"
	ErrorRateLimited  = "rate_limited"
)
