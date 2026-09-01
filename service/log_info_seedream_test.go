package service

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTextOtherInfoIncludesSeedreamBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
		SeedreamBilling: &relaycommon.SeedreamBillingDetails{
			Scene:           "layer_decomposition",
			InputImages:     1,
			GeneratedImages: 4,
			LowPixelImages:  3,
			HighPixelImages: 1,
			FallbackImages:  0,
			TotalPrice:      0.75,
		},
	}

	other := GenerateTextOtherInfo(ctx, info, 0, 1, 0, 0, 0, 0.75, -1)
	raw, ok := other["seedream_billing"]
	require.True(t, ok)
	billing, ok := raw.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "postpaid", billing["mode"])
	assert.Equal(t, "layer_decomposition", billing["scene"])
	assert.Equal(t, 4, billing["generated_images"])
	assert.Equal(t, 0.75, billing["total_price_rmb"])
}
