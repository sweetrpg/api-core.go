package util

import (
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/sweetrpg/api-core.go/constants"
	"github.com/sweetrpg/common.go/logging"
)

// redisPoolConnectTimeout bounds a dial so a stalled Redis connection fails fast rather than
// hanging past a caller's own (e.g. readiness-probe) timeout.
const redisPoolConnectTimeout = 5 * time.Second

// RedisPool builds a redigo connection pool from REDIS_HOST / REDIS_PORT / REDIS_PASS, for use
// as the backing store of the shared per-client rate limiter (and any other Redis need a
// service has). It returns nil when REDIS_HOST is unset, so a service with no Redis configured
// runs without the dependency - callers that require Redis (e.g. ratelimit.Middleware) must
// treat a nil pool as a fatal misconfiguration, per the fail-closed contract.
func RedisPool() *redis.Pool {
	host, found := os.LookupEnv(constants.REDIS_HOST)
	if !found || host == "" {
		logging.Logger.Warn("REDIS_HOST is not set; no Redis pool created")
		return nil
	}

	port := GetEnv(constants.REDIS_PORT, "6379")
	pass := os.Getenv(constants.REDIS_PASS)
	addr := fmt.Sprintf("%s:%s", host, port)

	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr, redis.DialConnectTimeout(redisPoolConnectTimeout))
			if err != nil {
				return nil, err
			}
			if pass != "" {
				if _, err := c.Do("AUTH", pass); err != nil {
					_ = c.Close()
					return nil, err
				}
			}
			return c, nil
		},
	}
}
