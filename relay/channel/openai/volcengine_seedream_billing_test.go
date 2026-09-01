package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyVolcengineSeedreamBilling(t *testing.T) {
	layerEnabled := true
	tests := []struct {
		name          string
		request       *dto.ImageRequest
		usage         dto.Usage
		response      string
		wantPrice     float64
		wantInput     int
		wantGenerated int
		wantLow       int
		wantHigh      int
		wantFallback  int
		wantScene     string
	}{
		{
			name:          "text to image high pixel output",
			request:       &dto.ImageRequest{Model: constant.ModelDoubaoSeedream5Pro},
			usage:         dto.Usage{GeneratedImages: 1},
			response:      `{"data":[{"size":"2048x2048"}],"usage":{"generated_images":1}}`,
			wantPrice:     0.60,
			wantGenerated: 1,
			wantHigh:      1,
			wantScene:     "standard",
		},
		{
			name:          "image to image includes additional input image price",
			request:       &dto.ImageRequest{Model: constant.ModelDoubaoSeedream5Pro, Image: []byte(`["a","b"]`)},
			usage:         dto.Usage{InputImages: 2, GeneratedImages: 1},
			response:      `{"data":[{"size":"1024x1024"}],"usage":{"input_images":2,"generated_images":1}}`,
			wantPrice:     0.32,
			wantInput:     2,
			wantGenerated: 1,
			wantLow:       1,
			wantScene:     "standard",
		},
		{
			name:    "layer decomposition prices every returned layer",
			request: &dto.ImageRequest{Model: constant.ModelDoubaoSeedream5Pro, Image: []byte(`"source"`), LayerDecomposition: &layerEnabled},
			usage:   dto.Usage{InputImages: 1, GeneratedImages: 4},
			response: `{"data":[
				{"size":"2048x2048"},
				{"size":"1297x1462"},
				{"size":"1989x919"},
				{"size":"1502x1492"}
			],"usage":{"input_images":1,"generated_images":4}}`,
			wantPrice:     0.75,
			wantInput:     1,
			wantGenerated: 4,
			wantLow:       3,
			wantHigh:      1,
			wantScene:     "layer_decomposition",
		},
		{
			name:          "missing output size uses high tier fallback",
			request:       &dto.ImageRequest{Model: constant.ModelDoubaoSeedream5Pro},
			usage:         dto.Usage{GeneratedImages: 2},
			response:      `{"data":[{"size":"1024x1024"},{}],"usage":{"generated_images":2}}`,
			wantPrice:     0.90,
			wantGenerated: 2,
			wantLow:       1,
			wantHigh:      1,
			wantFallback:  1,
			wantScene:     "standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName: constant.ModelDoubaoSeedream5Pro,
				Request:         tt.request,
				ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeVolcEngine},
				PriceData:       types.PriceData{UsePrice: true, Postpaid: true, ModelPrice: 0.60},
			}

			require.True(t, applyVolcengineSeedreamBilling(info, &tt.usage, []byte(tt.response)))
			assert.InDelta(t, tt.wantPrice, info.PriceData.ModelPrice, 0.0000001)
			assert.Nil(t, info.PriceData.OtherRatios())
			require.NotNil(t, info.SeedreamBilling)
			assert.Equal(t, tt.wantInput, info.SeedreamBilling.InputImages)
			assert.Equal(t, tt.wantGenerated, info.SeedreamBilling.GeneratedImages)
			assert.Equal(t, tt.wantLow, info.SeedreamBilling.LowPixelImages)
			assert.Equal(t, tt.wantHigh, info.SeedreamBilling.HighPixelImages)
			assert.Equal(t, tt.wantFallback, info.SeedreamBilling.FallbackImages)
			assert.Equal(t, tt.wantScene, info.SeedreamBilling.Scene)
		})
	}
}

func TestApplyVolcengineSeedreamBillingIgnoresOtherModels(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "other-image-model",
		Request:         &dto.ImageRequest{Model: "other-image-model"},
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeVolcEngine},
		PriceData:       types.PriceData{UsePrice: true, ModelPrice: 1.25},
	}

	assert.False(t, applyVolcengineSeedreamBilling(info, &dto.Usage{GeneratedImages: 1}, []byte(`{"data":[{"size":"2048x2048"}]}`)))
	assert.Equal(t, 1.25, info.PriceData.ModelPrice)
	assert.Nil(t, info.SeedreamBilling)
}

func TestUpdateOpenAIImageCountDoesNotMultiplySettledSeedreamPrice(t *testing.T) {
	info := &relaycommon.RelayInfo{
		SeedreamBilling: &relaycommon.SeedreamBillingDetails{TotalPrice: 0.75},
		PriceData:       types.PriceData{UsePrice: true, Postpaid: true, ModelPrice: 0.75},
	}

	updateOpenAIImageCount(info, 4)

	assert.Nil(t, info.PriceData.OtherRatios())
}

func TestApplyVolcengineSeedreamStreamBilling(t *testing.T) {
	layerEnabled := true
	tests := []struct {
		name          string
		request       *dto.ImageRequest
		usage         dto.Usage
		sizes         []string
		completed     int
		wantPrice     float64
		wantGenerated int
		wantLow       int
		wantHigh      int
		wantFallback  int
	}{
		{
			name:          "completed events without usage still bill every image",
			request:       &dto.ImageRequest{Model: constant.ModelDoubaoSeedream5Pro},
			sizes:         []string{"1024x1024", ""},
			completed:     2,
			wantPrice:     0.90,
			wantGenerated: 2,
			wantLow:       1,
			wantHigh:      1,
			wantFallback:  1,
		},
		{
			name:          "usage count without sizes uses high tier fallback",
			request:       &dto.ImageRequest{Model: constant.ModelDoubaoSeedream5Pro},
			usage:         dto.Usage{GeneratedImages: 2},
			completed:     2,
			wantPrice:     1.20,
			wantGenerated: 2,
			wantHigh:      2,
			wantFallback:  2,
		},
		{
			name:          "layer stream preserves mixed size prices",
			request:       &dto.ImageRequest{Model: constant.ModelDoubaoSeedream5Pro, LayerDecomposition: &layerEnabled},
			sizes:         []string{"2048x2048", "1297x1462", "1989x919", "1502x1492"},
			completed:     4,
			wantPrice:     0.75,
			wantGenerated: 4,
			wantLow:       3,
			wantHigh:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName: constant.ModelDoubaoSeedream5Pro,
				Request:         tt.request,
				ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeVolcEngine},
				PriceData:       types.PriceData{UsePrice: true, Postpaid: true, ModelPrice: 0.60},
			}

			require.True(t, applyVolcengineSeedreamStreamBilling(info, &tt.usage, tt.sizes, tt.completed))
			assert.InDelta(t, tt.wantPrice, info.PriceData.ModelPrice, 0.0000001)
			require.NotNil(t, info.SeedreamBilling)
			assert.Equal(t, tt.wantGenerated, info.SeedreamBilling.GeneratedImages)
			assert.Equal(t, tt.wantLow, info.SeedreamBilling.LowPixelImages)
			assert.Equal(t, tt.wantHigh, info.SeedreamBilling.HighPixelImages)
			assert.Equal(t, tt.wantFallback, info.SeedreamBilling.FallbackImages)
		})
	}
}
