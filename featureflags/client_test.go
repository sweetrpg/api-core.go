package featureflags

import (
	"context"
	"os"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/sweetrpg/common.go/logging"
)

func TestMain(m *testing.M) {
	logging.Init()
	os.Exit(m.Run())
}

// fakeProvider is a controllable openfeature.FeatureProvider used to drive
// the client through the scenarios BoolFlag must fail open on, without a
// real flagd sidecar.
type fakeProvider struct {
	initErr error
	value   bool
	evalErr *openfeature.ResolutionError
}

func (p *fakeProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: "fake"}
}

func (p *fakeProvider) Hooks() []openfeature.Hook { return nil }

func (p *fakeProvider) Init(_ openfeature.EvaluationContext) error {
	return p.initErr
}

func (p *fakeProvider) Shutdown() {}

func (p *fakeProvider) BooleanEvaluation(_ context.Context, _ string, defaultValue bool, _ openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	if p.evalErr != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: *p.evalErr,
				Reason:          openfeature.ErrorReason,
			},
		}
	}
	return openfeature.BoolResolutionDetail{
		Value: p.value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
		},
	}
}

func (p *fakeProvider) StringEvaluation(_ context.Context, _ string, defaultValue string, _ openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{Value: defaultValue}
}

func (p *fakeProvider) FloatEvaluation(_ context.Context, _ string, defaultValue float64, _ openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{Value: defaultValue}
}

func (p *fakeProvider) IntEvaluation(_ context.Context, _ string, defaultValue int64, _ openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{Value: defaultValue}
}

func (p *fakeProvider) ObjectEvaluation(_ context.Context, _ string, defaultValue any, _ openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{Value: defaultValue}
}

// newTestClient installs provider as the default openfeature provider and
// returns a featureflags.Client wired to it, waiting for provider init to
// settle so each test observes a deterministic state.
func newTestClient(t *testing.T, provider *fakeProvider) *Client {
	t.Helper()
	_ = openfeature.SetProviderAndWait(provider)
	return &Client{ofClient: openfeature.NewClient(t.Name())}
}

func TestBoolFlag_NormalEvaluation(t *testing.T) {
	provider := &fakeProvider{value: true}
	client := newTestClient(t, provider)

	result := client.BoolFlag(context.Background(), "profiling-enabled", false)

	assert.True(t, result, "expected the provider's configured value, not the default")
}

func TestBoolFlag_UnreachableAtInit_ReturnsDefault(t *testing.T) {
	provider := &fakeProvider{
		initErr: assert.AnError,
		evalErr: ptr(openfeature.NewProviderNotReadyResolutionError("flagd sidecar unreachable during init")),
	}
	client := newTestClient(t, provider)

	result := client.BoolFlag(context.Background(), "profiling-enabled", false)

	assert.False(t, result, "expected the caller-supplied default when flagd is unreachable at init")
}

func TestBoolFlag_UnreachableAfterInit_ReturnsDefault(t *testing.T) {
	provider := &fakeProvider{
		evalErr: ptr(openfeature.NewGeneralResolutionError("flagd sidecar connection lost")),
	}
	client := newTestClient(t, provider)

	result := client.BoolFlag(context.Background(), "profiling-enabled", true)

	assert.True(t, result, "expected the caller-supplied default when flagd becomes unreachable after init")
}

func TestBoolFlag_EvaluationError_ReturnsDefault(t *testing.T) {
	provider := &fakeProvider{
		evalErr: ptr(openfeature.NewParseErrorResolutionError("malformed flag definition")),
	}
	client := newTestClient(t, provider)

	result := client.BoolFlag(context.Background(), "profiling-enabled", true)

	assert.True(t, result, "expected the caller-supplied default on any evaluation error, not just connectivity errors")
}

func ptr[T any](v T) *T { return &v }
