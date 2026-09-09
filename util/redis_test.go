package util

import (
	"testing"

	"github.com/sweetrpg/api-core.go/constants"
)

func TestRedisPoolNilWithoutHost(t *testing.T) {
	t.Setenv(constants.REDIS_HOST, "")
	if pool := RedisPool(); pool != nil {
		t.Fatal("RedisPool() without REDIS_HOST = non-nil, want nil")
	}
}

func TestRedisPoolBuiltWithHost(t *testing.T) {
	t.Setenv(constants.REDIS_HOST, "localhost")
	t.Setenv(constants.REDIS_PORT, "6390")

	pool := RedisPool()
	if pool == nil {
		t.Fatal("RedisPool() with REDIS_HOST = nil, want a pool")
	}
	t.Cleanup(func() { _ = pool.Close() })

	// The pool is lazy - no connection is dialed until first use - so building one against a
	// host with nothing listening must not error here.
	if pool.Dial == nil {
		t.Fatal("RedisPool() returned a pool with no Dial func")
	}
}
