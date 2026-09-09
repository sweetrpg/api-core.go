package ratelimit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"

	"github.com/sweetrpg/api-core.go/constants"
	"github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/common.go/logging"
)

func TestMain(m *testing.M) {
	logging.Init()
	os.Exit(m.Run())
}

func newTestPool(t *testing.T) (*redis.Pool, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", mr.Addr())
		},
	}
	t.Cleanup(func() { _ = pool.Close() })

	return pool, mr
}

func unreachablePool(t *testing.T) *redis.Pool {
	t.Helper()

	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()

	// A fresh pool pointed at the now-closed address, with a bounded dial so the closed
	// remote end surfaces as an immediate connection refusal rather than a long TCP timeout.
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, redis.DialConnectTimeout(2*time.Second))
		},
	}
	t.Cleanup(func() { _ = pool.Close() })

	return pool
}

func TestAllowWithinLimit(t *testing.T) {
	pool, _ := newTestPool(t)
	limiter := New(pool, map[string]Tier{TierStandard: {Limit: 3, Window: 60}})

	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(context.Background(), "client-a", TierStandard)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if !allowed {
			t.Fatalf("Allow() request %d = false, want true (within limit)", i+1)
		}
	}
}

func TestAllowRejectsOverLimit(t *testing.T) {
	pool, _ := newTestPool(t)
	limiter := New(pool, map[string]Tier{TierStandard: {Limit: 2, Window: 60}})

	for i := 0; i < 2; i++ {
		if allowed, err := limiter.Allow(context.Background(), "client-a", TierStandard); err != nil || !allowed {
			t.Fatalf("Allow() request %d = (%v, %v), want (true, nil)", i+1, allowed, err)
		}
	}

	allowed, err := limiter.Allow(context.Background(), "client-a", TierStandard)
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if allowed {
		t.Fatal("Allow() over limit = true, want false")
	}
}

func TestAllowTracksClientsIndependently(t *testing.T) {
	pool, _ := newTestPool(t)
	limiter := New(pool, map[string]Tier{TierStandard: {Limit: 1, Window: 60}})

	if allowed, err := limiter.Allow(context.Background(), "client-a", TierStandard); err != nil || !allowed {
		t.Fatalf("Allow(client-a) = (%v, %v), want (true, nil)", allowed, err)
	}
	if allowed, err := limiter.Allow(context.Background(), "client-b", TierStandard); err != nil || !allowed {
		t.Fatalf("Allow(client-b) = (%v, %v), want (true, nil) - separate client budget", allowed, err)
	}
	if allowed, _ := limiter.Allow(context.Background(), "client-a", TierStandard); allowed {
		t.Fatal("Allow(client-a) second request = true, want false - client-a is over its own limit")
	}
}

func TestAllowTracksTiersIndependently(t *testing.T) {
	pool, _ := newTestPool(t)
	limiter := New(pool, map[string]Tier{
		TierCheap:    {Limit: 5, Window: 60},
		TierStandard: {Limit: 1, Window: 60},
	})

	if allowed, err := limiter.Allow(context.Background(), "client-a", TierStandard); err != nil || !allowed {
		t.Fatalf("Allow(standard) = (%v, %v), want (true, nil)", allowed, err)
	}
	if allowed, _ := limiter.Allow(context.Background(), "client-a", TierStandard); allowed {
		t.Fatal("Allow(standard) second request = true, want false")
	}
	if allowed, err := limiter.Allow(context.Background(), "client-a", TierCheap); err != nil || !allowed {
		t.Fatalf("Allow(cheap) = (%v, %v), want (true, nil) - cheap tier has its own budget", allowed, err)
	}
}

func TestAllowUnknownTierFallsBackToStandard(t *testing.T) {
	pool, _ := newTestPool(t)
	limiter := New(pool, map[string]Tier{TierStandard: {Limit: 1, Window: 60}})

	if allowed, err := limiter.Allow(context.Background(), "client-a", "unknown-tier"); err != nil || !allowed {
		t.Fatalf("Allow(unknown-tier) = (%v, %v), want (true, nil) via standard fallback", allowed, err)
	}
	if allowed, _ := limiter.Allow(context.Background(), "client-a", "unknown-tier"); allowed {
		t.Fatal("Allow(unknown-tier) second request = true, want false - fell back to standard's limit of 1")
	}
}

func TestAllowFailsClosedWhenBackendUnreachable(t *testing.T) {
	limiter := New(unreachablePool(t), map[string]Tier{TierStandard: {Limit: 10, Window: 60}})

	allowed, err := limiter.Allow(context.Background(), "client-a", TierStandard)
	if err == nil {
		t.Fatal("Allow() with unreachable backend error = nil, want non-nil")
	}
	if allowed {
		t.Fatal("Allow() with unreachable backend = true, want false (fail closed)")
	}
}

func TestPingSucceedsWhenReachable(t *testing.T) {
	pool, _ := newTestPool(t)

	if err := Ping(context.Background(), pool); err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
}

func TestPingFailsWhenUnreachable(t *testing.T) {
	if err := Ping(context.Background(), unreachablePool(t)); err == nil {
		t.Fatal("Ping() with unreachable backend error = nil, want non-nil")
	}
}

func TestTierFor(t *testing.T) {
	cases := map[string]string{
		"/status/ping":   TierCheap,
		"/status/health": TierCheap,
		"/volumes":       TierStandard,
		"/":              TierStandard,
		"/statusicons":   TierStandard, // not under /status/
	}
	for path, want := range cases {
		if got := TierFor(path); got != want {
			t.Errorf("TierFor(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestClientKeyPrefersAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/volumes", nil)
	c.Request.Header.Set("X-API-Key", "abc123")

	if got := ClientKey(c); got != "key:abc123" {
		t.Errorf("ClientKey() with header = %q, want %q", got, "key:abc123")
	}
}

func TestClientKeyFallsBackToIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/volumes", nil)
	c.Request.RemoteAddr = "203.0.113.7:54321"

	if got := ClientKey(c); got != "ip:203.0.113.7" {
		t.Errorf("ClientKey() without header = %q, want %q", got, "ip:203.0.113.7")
	}
}

func newMiddlewareRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/volumes", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/status/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func doGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "198.51.100.5:12345"
	r.ServeHTTP(w, req)
	return w
}

func TestMiddlewareAllowsUnderLimitThen429s(t *testing.T) {
	pool, _ := newTestPool(t)
	r := newMiddlewareRouter(Middleware(pool, Options{
		Cheap:    Tier{Limit: 100, Window: 60},
		Standard: Tier{Limit: 2, Window: 60},
	}))

	for i := 0; i < 2; i++ {
		if w := doGet(r, "/volumes"); w.Code != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i+1, w.Code)
		}
	}

	w := doGet(r, "/volumes")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("over-limit request = %d, want 429", w.Code)
	}
	var body vo.ErrorVO
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("429 body not JSON: %v", err)
	}
	if body.Error != constants.ErrorRateLimited {
		t.Errorf("429 body error = %q, want %q", body.Error, constants.ErrorRateLimited)
	}
}

func TestMiddlewareCheapTierSurvivesStandardExhaustion(t *testing.T) {
	pool, _ := newTestPool(t)
	r := newMiddlewareRouter(Middleware(pool, Options{
		Cheap:    Tier{Limit: 100, Window: 60},
		Standard: Tier{Limit: 1, Window: 60},
	}))

	if w := doGet(r, "/volumes"); w.Code != http.StatusOK {
		t.Fatalf("first standard request = %d, want 200", w.Code)
	}
	if w := doGet(r, "/volumes"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("second standard request = %d, want 429", w.Code)
	}
	if w := doGet(r, "/status/ping"); w.Code != http.StatusOK {
		t.Fatalf("cheap-tier request while standard exhausted = %d, want 200", w.Code)
	}
}

func TestMiddlewareFailsClosedOnUnreachableBackend(t *testing.T) {
	r := newMiddlewareRouter(Middleware(unreachablePool(t), Options{
		Cheap:    Tier{Limit: 100, Window: 60},
		Standard: Tier{Limit: 100, Window: 60},
	}))

	w := doGet(r, "/volumes")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("request with unreachable backend = %d, want 503", w.Code)
	}
	var body vo.ErrorVO
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("503 body not JSON: %v", err)
	}
	if body.Error != constants.ErrorRateLimitUnavailable {
		t.Errorf("503 body error = %q, want %q", body.Error, constants.ErrorRateLimitUnavailable)
	}
}

func TestMiddlewareFailsClosedOnNilPool(t *testing.T) {
	r := newMiddlewareRouter(Middleware(nil, DefaultOptions()))

	if w := doGet(r, "/volumes"); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("request with nil pool = %d, want 503", w.Code)
	}
}

func TestDefaultOptionsUsesEnvThenDefaults(t *testing.T) {
	if opts := DefaultOptions(); opts.Standard.Limit != constants.RateLimitStandardDefault ||
		opts.Cheap.Limit != constants.RateLimitCheapDefault ||
		opts.Standard.Window != constants.RateLimitStandardWindowDefault ||
		opts.Cheap.Window != constants.RateLimitCheapWindowDefault {
		t.Fatalf("DefaultOptions() without env = %+v, want catalog-api defaults", opts)
	}

	t.Setenv(constants.RATE_LIMIT_STANDARD, "7")
	t.Setenv(constants.RATE_LIMIT_STANDARD_WINDOW, "11")
	if opts := DefaultOptions(); opts.Standard.Limit != 7 || opts.Standard.Window != 11 {
		t.Fatalf("DefaultOptions() with env = %+v, want Standard{7, 11}", opts.Standard)
	}
}
