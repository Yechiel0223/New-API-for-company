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
		{name: "480p", resolution: "480p", tokens: 48_437.8125, wantTier: "480p_720p", wantCost: 3.390646875, wantQuota: 1_695_323},
		{name: "720p", resolution: "720p", tokens: 108_900, wantTier: "480p_720p", wantCost: 7.623, wantQuota: 3_811_500},
		{name: "1080p promo", resolution: "1080p", tokens: 245_025, wantTier: "1080p_promo", wantCost: 13.584186, wantQuota: 6_792_093},
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
				QuotaPerUnit:     500_000,
				ExprVersion:      1,
				TaskUsageBilling: true,
			}, TokenParams{}, request)
			require.NoError(t, err)
			assert.Equal(t, test.wantQuota, result.ActualQuotaAfterGroup)
		})
	}
}

func TestSeedance20PricingVectors(t *testing.T) {
	expression := `u("resolution") == "4k"
  ? (u("video_input") == "video"
    ? tier("4k_video_input", u("tokens") * 16 / 1000000)
    : tier("4k_no_video_input", u("tokens") * 26 / 1000000))
  : u("resolution") == "1080p"
    ? (u("video_input") == "video"
      ? tier("1080p_video_input", u("tokens") * 31 / 1000000)
      : tier("1080p_no_video_input", u("tokens") * 51 / 1000000))
    : (u("video_input") == "video"
      ? tier("480p_720p_video_input", u("tokens") * 28 / 1000000)
      : tier("480p_720p_no_video_input", u("tokens") * 46 / 1000000))`
	tests := []struct {
		name       string
		resolution string
		videoInput string
		tokens     float64
		wantTier   string
		wantCost   float64
		wantQuota  int
	}{
		{name: "real 720p 4s text to video", resolution: "720p", videoInput: "none", tokens: 87_300, wantTier: "480p_720p_no_video_input", wantCost: 4.0158, wantQuota: 2_007_900},
		{name: "720p video input", resolution: "720p", videoInput: "video", tokens: 100_000, wantTier: "480p_720p_video_input", wantCost: 2.8, wantQuota: 1_400_000},
		{name: "1080p no video input", resolution: "1080p", videoInput: "none", tokens: 100_000, wantTier: "1080p_no_video_input", wantCost: 5.1, wantQuota: 2_550_000},
		{name: "1080p video input", resolution: "1080p", videoInput: "video", tokens: 100_000, wantTier: "1080p_video_input", wantCost: 3.1, wantQuota: 1_550_000},
		{name: "4k no video input", resolution: "4k", videoInput: "none", tokens: 100_000, wantTier: "4k_no_video_input", wantCost: 2.6, wantQuota: 1_300_000},
		{name: "4k video input", resolution: "4k", videoInput: "video", tokens: 100_000, wantTier: "4k_video_input", wantCost: 1.6, wantQuota: 800_000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := RequestInput{Usage: map[string]any{
				"resolution":  test.resolution,
				"video_input": test.videoInput,
				"tokens":      test.tokens,
			}}
			cost, trace, err := RunExprWithRequest(expression, TokenParams{}, request)
			require.NoError(t, err)
			assert.InDelta(t, test.wantCost, cost, 0.000000001)
			assert.Equal(t, test.wantTier, trace.MatchedTier)

			result, err := ComputeTieredQuotaWithRequest(&BillingSnapshot{
				ExprString:       expression,
				ExprHash:         ExprHashString(expression),
				GroupRatio:       1,
				QuotaPerUnit:     500_000,
				ExprVersion:      1,
				TaskUsageBilling: true,
			}, TokenParams{}, request)
			require.NoError(t, err)
			assert.Equal(t, test.wantQuota, result.ActualQuotaAfterGroup)
		})
	}
}
