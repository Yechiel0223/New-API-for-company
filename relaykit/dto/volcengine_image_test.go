package dto

import (
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestImageRequestPreservesLayerDecomposition(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "enabled", raw: `{"model":"doubao-seedream-5-0-pro-260628","prompt":"split it","layer_decomposition":true}`, want: true},
		{name: "explicitly disabled", raw: `{"model":"doubao-seedream-5-0-pro-260628","prompt":"draw it","layer_decomposition":false}`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req ImageRequest
			require.NoError(t, kitutil.Unmarshal([]byte(tt.raw), &req))
			require.NotNil(t, req.LayerDecomposition)
			assert.Equal(t, tt.want, *req.LayerDecomposition)

			encoded, err := kitutil.Marshal(req)
			require.NoError(t, err)
			value := gjson.GetBytes(encoded, "layer_decomposition")
			require.True(t, value.Exists())
			assert.Equal(t, tt.want, value.Bool())
		})
	}
}

func TestUsageParsesVolcengineImageCounts(t *testing.T) {
	raw := []byte(`{"usage":{"input_images":2,"generated_images":4,"output_tokens":65472,"total_tokens":65472}}`)

	var response SimpleResponse
	require.NoError(t, kitutil.Unmarshal(raw, &response))
	assert.Equal(t, 2, response.Usage.InputImages)
	assert.Equal(t, 4, response.Usage.GeneratedImages)
	assert.Equal(t, 65472, response.Usage.OutputTokens)
	assert.Equal(t, 65472, response.Usage.TotalTokens)
}
