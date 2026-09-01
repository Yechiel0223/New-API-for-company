package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCalculateTextQuotaSummaryUsesSettledSeedreamPriceOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedream-5-0-pro-260628",
		PriceData: hosttypes.PriceData{
			Postpaid:   true,
			UsePrice:   true,
			ModelPrice: 0.75,
			GroupRatioInfo: hosttypes.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}

	summary := calculateTextQuotaSummary(ctx, info, &dto.Usage{PromptTokens: 1, TotalTokens: 1})

	require.Equal(t, common.QuotaFromFloat(0.75*common.QuotaPerUnit), summary.Quota)
}
