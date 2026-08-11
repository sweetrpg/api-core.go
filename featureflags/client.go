// Package featureflags provides a shared client for evaluating feature flags
// against the flagd sidecar injected into a pod by the OpenFeature Operator.
package featureflags

import (
	"context"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/sweetrpg/common.go/logging"
)

// Client evaluates feature flags. Every method fails open: if the flagd
// sidecar is unreachable or an evaluation errors, the caller-supplied default
// is returned rather than blocking or propagating the error - a briefly
// unavailable sidecar must degrade to "flag off," not "service won't start."
type Client struct {
	ofClient *openfeature.Client
}

// New creates a feature-flag client for serviceName, wired to the flagd
// sidecar injected into this pod by the OpenFeature Operator. The sidecar
// runs on localhost, so the provider uses its default RPC-mode configuration
// (host "localhost", port 8013) rather than a cluster-DNS address.
//
// Provider setup is asynchronous (openfeature.SetProvider does not block) so
// a slow-starting or unreachable sidecar never delays service startup;
// evaluations made before the provider is ready simply fail open.
func New(serviceName string) *Client {
	provider, err := flagd.NewProvider()
	if err != nil {
		logging.Logger.Warn("failed to construct flagd provider; feature flags will use caller-supplied defaults", "error", err.Error())
		return &Client{ofClient: openfeature.NewClient(serviceName)}
	}

	if err := openfeature.SetProvider(provider); err != nil {
		logging.Logger.Warn("failed to set flagd provider; feature flags will use caller-supplied defaults", "error", err.Error())
	}

	return &Client{ofClient: openfeature.NewClient(serviceName)}
}

// BoolFlag evaluates a boolean flag, returning defaultValue if flagd is
// unreachable or the evaluation otherwise errors.
func (c *Client) BoolFlag(ctx context.Context, key string, defaultValue bool) bool {
	value, err := c.ofClient.BooleanValue(ctx, key, defaultValue, openfeature.EvaluationContext{})
	if err != nil {
		logging.Logger.Warn("feature flag evaluation failed; using default", "flag", key, "default", defaultValue, "error", err.Error())
		return defaultValue
	}

	return value
}
