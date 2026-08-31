package billingexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedance25POCPricingVectors(t *testing.T) {
	expression := `u("resolution") == "1080p"
  ? tier("1080p_promo", u("tokens") * 55.44 / 1000000)
  : tier("480p_720p", u("tokens") * 70 / 1000000)`
	tests := []struct {
		name       string
		resolution string
		tokens     float64
		wantTier   string
		wantCost   float64
		wantQuota  int
	}{
		{name: "480p", resolution: "480p", tokens: 48_038, wantTier: "480p_720p", wantCost: 3.36266, wantQuota: 1_681_330},
		{name: "720p", resolution: "720p", tokens: 108_000, wantTier: "480p_720p", wantCost: 7.56, wantQuota: 3_780_000},
		{name: "1080p promo", resolution: "1080p", tokens: 243_000, wantTier: "1080p_promo", wantCost: 13.47192, wantQuota: 6_735_960},
		{name: "zero tokens", resolution: "720p", tokens: 0, wantTier: "480p_720p", wantCost: 0, wantQuota: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := RequestInput{Usage: map[string]any{
				"resolution": test.resolution,
				"tokens":     test.tokens,
			}}
			cost, trace, err := RunExprWithRequest(expression, TokenParams{}, request)
			require.NoError(t, err)
			assert.InDelta(t, test.wantCost, cost, 0.000000001)
			assert.Equal(t, test.wantTier, trace.MatchedTier)

			result, err := ComputeTieredQuotaWithRequest(&BillingSnapshot{
				ExprString:       expression,
				ExprHash:         ExprHashString(expression),
				GroupRatio:       1,
				QuotaPerUnit:      500_000,
				ExprVersion:      1,
				TaskUsageBilling: true,
			}, TokenParams{}, request)
			require.NoError(t, err)
			assert.Equal(t, test.wantQuota, result.ActualQuotaAfterGroup)
		})
	}
}
