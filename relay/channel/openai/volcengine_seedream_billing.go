package openai

import (
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/shopspring/decimal"
)

const seedreamLowPixelLimit int64 = 2_610_000

var (
	seedreamAdditionalInputPrice = decimal.RequireFromString("0.02")
	seedreamStandardLowPrice     = decimal.RequireFromString("0.30")
	seedreamStandardHighPrice    = decimal.RequireFromString("0.60")
	seedreamLayerLowPrice        = decimal.RequireFromString("0.15")
	seedreamLayerHighPrice       = decimal.RequireFromString("0.30")
)

func applyVolcengineSeedreamBilling(info *relaycommon.RelayInfo, usage *dto.Usage, responseBody []byte) bool {
	if info == nil || usage == nil || info.ChannelMeta == nil ||
		info.ChannelType != constant.ChannelTypeVolcEngine ||
		info.OriginModelName != constant.ModelDoubaoSeedream5Pro {
		return false
	}

	request, _ := info.Request.(*dto.ImageRequest)
	details := &relaycommon.SeedreamBillingDetails{Scene: "standard"}
	if request != nil && request.LayerDecomposition != nil && *request.LayerDecomposition {
		details.Scene = "layer_decomposition"
	}

	details.InputImages = usage.InputImages
	if details.InputImages == 0 && request != nil {
		details.InputImages = countSeedreamInputImages(request)
	}

	var payload struct {
		Data []struct {
			Size string `json:"size"`
		} `json:"data"`
	}
	_ = common.Unmarshal(responseBody, &payload)

	details.GeneratedImages = usage.GeneratedImages
	if len(payload.Data) > details.GeneratedImages {
		details.GeneratedImages = len(payload.Data)
	}

	lowPrice, highPrice := seedreamStandardLowPrice, seedreamStandardHighPrice
	if details.Scene == "layer_decomposition" {
		lowPrice, highPrice = seedreamLayerLowPrice, seedreamLayerHighPrice
	}

	totalPrice := decimal.Zero
	if details.InputImages > 1 {
		totalPrice = totalPrice.Add(seedreamAdditionalInputPrice.Mul(decimal.NewFromInt(int64(details.InputImages - 1))))
	}
	for index := 0; index < details.GeneratedImages; index++ {
		if index < len(payload.Data) {
			if pixels, ok := parseSeedreamImagePixels(payload.Data[index].Size); ok {
				if pixels <= seedreamLowPixelLimit {
					details.LowPixelImages++
					totalPrice = totalPrice.Add(lowPrice)
				} else {
					details.HighPixelImages++
					totalPrice = totalPrice.Add(highPrice)
				}
				continue
			}
		}
		details.HighPixelImages++
		details.FallbackImages++
		totalPrice = totalPrice.Add(highPrice)
	}

	details.TotalPrice, _ = totalPrice.Float64()
	info.SeedreamBilling = details
	info.PriceData.Postpaid = true
	info.PriceData.UsePrice = true
	info.PriceData.FreeModel = false
	info.PriceData.ModelPrice = details.TotalPrice
	info.PriceData.ReplaceOtherRatios(nil)
	return true
}

func applyVolcengineSeedreamStreamBilling(info *relaycommon.RelayInfo, usage *dto.Usage, sizes []string, completedImages int) bool {
	if info == nil || usage == nil || info.ChannelMeta == nil ||
		info.ChannelType != constant.ChannelTypeVolcEngine ||
		info.OriginModelName != constant.ModelDoubaoSeedream5Pro {
		return false
	}

	generatedImages := usage.GeneratedImages
	if completedImages > generatedImages {
		generatedImages = completedImages
	}
	if len(sizes) > generatedImages {
		generatedImages = len(sizes)
	}
	usage.GeneratedImages = generatedImages

	payload := struct {
		Data []struct {
			Size string `json:"size,omitempty"`
		} `json:"data"`
	}{
		Data: make([]struct {
			Size string `json:"size,omitempty"`
		}, generatedImages),
	}
	for index := range payload.Data {
		if index < len(sizes) {
			payload.Data[index].Size = sizes[index]
		}
	}

	responseBody, err := common.Marshal(payload)
	if err != nil {
		return applyVolcengineSeedreamBilling(info, usage, nil)
	}
	return applyVolcengineSeedreamBilling(info, usage, responseBody)
}

func countSeedreamInputImages(request *dto.ImageRequest) int {
	if request == nil {
		return 0
	}
	if count := countJSONImages(request.Image); count > 0 {
		return count
	}
	return countJSONImages(request.Images)
}

func countJSONImages(raw []byte) int {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return 0
	}
	var items []interface{}
	if err := common.Unmarshal(raw, &items); err == nil {
		return len(items)
	}
	var item string
	if err := common.Unmarshal(raw, &item); err == nil && item != "" {
		return 1
	}
	return 0
}

func parseSeedreamImagePixels(size string) (int64, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, false
	}
	width, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || width <= 0 {
		return 0, false
	}
	height, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || height <= 0 || width > math.MaxInt64/height {
		return 0, false
	}
	return width * height, true
}
