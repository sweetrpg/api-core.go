package ratelimit

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/sweetrpg/api-core.go/constants"
)

var testOptions = Options{Cheap: Tier{Limit: 100, Window: 60}, Standard: Tier{Limit: 100, Window: 60}}

func errorCount(tier string) float64 {
	return testutil.ToFloat64(backendErrors.WithLabelValues(tier))
}

func TestStatusRoutesFailOpenWhenThePoolIsNil(t *testing.T) {
	r := newMiddlewareRouter(Middleware(nil, testOptions))
	before := errorCount(TierCheap)

	if w := doGet(r, "/status/ping"); w.Code != http.StatusOK {
		t.Fatalf("/status/ping with a nil pool = %d, want 200", w.Code)
	}
	if got := errorCount(TierCheap) - before; got != 1 {
		t.Errorf("ratelimit_backend_errors_total{tier=cheap} rose by %v, want 1", got)
	}
}

func TestStatusRoutesFailOpenWhenTheStoreErrors(t *testing.T) {
	r := newMiddlewareRouter(Middleware(unreachablePool(t), testOptions))
	before := errorCount(TierCheap)

	if w := doGet(r, "/status/ping"); w.Code != http.StatusOK {
		t.Fatalf("/status/ping with an erroring store = %d, want 200", w.Code)
	}
	if got := errorCount(TierCheap) - before; got != 1 {
		t.Errorf("counter rose by %v, want 1 (one error, counted once)", got)
	}
}

func TestStandardRoutesStillFailClosedAndAreCounted(t *testing.T) {
	for name, pool := range map[string]func(*testing.T) gin.HandlerFunc{
		"nil pool":      func(*testing.T) gin.HandlerFunc { return Middleware(nil, testOptions) },
		"erroring pool": func(t *testing.T) gin.HandlerFunc { return Middleware(unreachablePool(t), testOptions) },
	} {
		t.Run(name, func(t *testing.T) {
			r := newMiddlewareRouter(pool(t))
			before := errorCount(TierStandard)
			if w := doGet(r, "/volumes"); w.Code != http.StatusServiceUnavailable {
				t.Fatalf("standard route = %d, want 503", w.Code)
			}
			if got := errorCount(TierStandard) - before; got != 1 {
				t.Errorf("counter{tier=standard} rose by %v, want 1", got)
			}
		})
	}
}

func TestStatusRouteStillRateLimitedWhileTheStoreWorks(t *testing.T) {
	pool, _ := newTestPool(t)
	r := newMiddlewareRouter(Middleware(pool, Options{Cheap: Tier{Limit: 2, Window: 60}, Standard: Tier{Limit: 100, Window: 60}}))

	for i := 0; i < 2; i++ {
		if w := doGet(r, "/status/ping"); w.Code != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i+1, w.Code)
		}
	}
	w := doGet(r, "/status/ping")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("over-limit status request = %d, want 429 (fail-open is for store faults only)", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("429 on a status route has no Retry-After")
	}
}

func TestDefaultsAreTwelveHundredCheapAndThreeHundredStandard(t *testing.T) {
	opts := DefaultOptions()
	if opts.Cheap != (Tier{Limit: 1200, Window: 60}) || opts.Standard != (Tier{Limit: 300, Window: 60}) {
		t.Errorf("defaults = %+v, want cheap 1200/60 and standard 300/60", opts)
	}
	if constants.RateLimitCheapDefault != 1200 || constants.RateLimitStandardDefault != 300 {
		t.Errorf("constants = %d/%d", constants.RateLimitCheapDefault, constants.RateLimitStandardDefault)
	}
}

func TestEvery429CarriesRetryAfterWithTheSecondsLeftInTheWindow(t *testing.T) {
	pool, mr := newTestPool(t)
	r := newMiddlewareRouter(Middleware(pool, Options{Cheap: Tier{Limit: 100, Window: 60}, Standard: Tier{Limit: 1, Window: 60}}))

	if w := doGet(r, "/volumes"); w.Code != http.StatusOK {
		t.Fatalf("first request = %d", w.Code)
	}
	w := doGet(r, "/volumes")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second request = %d, want 429", w.Code)
	}
	got, err := strconv.Atoi(w.Header().Get("Retry-After"))
	if err != nil || got < 1 || got > 60 {
		t.Fatalf("Retry-After = %q, want an integer 1..60", w.Header().Get("Retry-After"))
	}
	if got != 60 {
		t.Errorf("Retry-After right after the window opened = %d, want 60", got)
	}

	mr.FastForward(45 * time.Second)
	w = doGet(r, "/volumes")
	if got, _ := strconv.Atoi(w.Header().Get("Retry-After")); got != 15 {
		t.Errorf("Retry-After 45s into a 60s window = %d, want 15", got)
	}
}

func TestRetryAfterIsAtLeastOneSecond(t *testing.T) {
	pool, mr := newTestPool(t)
	r := newMiddlewareRouter(Middleware(pool, Options{Cheap: Tier{Limit: 100, Window: 60}, Standard: Tier{Limit: 1, Window: 60}}))
	doGet(r, "/volumes")
	mr.FastForward(59*time.Second + 600*time.Millisecond)

	w := doGet(r, "/volumes")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", w.Code)
	}
	if got, _ := strconv.Atoi(w.Header().Get("Retry-After")); got != 1 {
		t.Errorf("Retry-After with under a second left = %d, want 1", got)
	}
}

func TestACounterWithNoExpiryIsRepairedAndAnswersWithTheFullWindow(t *testing.T) {
	pool, mr := newTestPool(t)
	r := newMiddlewareRouter(Middleware(pool, Options{Cheap: Tier{Limit: 100, Window: 60}, Standard: Tier{Limit: 1, Window: 30}}))
	// A crash between INCR and EXPIRE leaves a counter that never resets.
	if err := mr.Set("ratelimit:standard:ip:198.51.100.5", "5"); err != nil {
		t.Fatal(err)
	}

	w := doGet(r, "/volumes")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if got, _ := strconv.Atoi(w.Header().Get("Retry-After")); got != 30 {
		t.Errorf("Retry-After = %d, want the full window 30", got)
	}
	if ttl := mr.TTL("ratelimit:standard:ip:198.51.100.5"); ttl <= 0 {
		t.Errorf("the stuck counter was not given an expiry (ttl %v)", ttl)
	}
}

func TestAllowStillReportsWithinLimit(t *testing.T) {
	pool, _ := newTestPool(t)
	l := New(pool, map[string]Tier{TierStandard: {Limit: 1, Window: 60}})
	res, err := l.Check(t.Context(), "k", TierStandard)
	if err != nil || !res.Allowed {
		t.Fatalf("first check = %+v err = %v", res, err)
	}
	res, err = l.Check(t.Context(), "k", TierStandard)
	if err != nil || res.Allowed || res.RetryAfter != 60*time.Second {
		t.Fatalf("second check = %+v err = %v, want denied with 60s", res, err)
	}
}
