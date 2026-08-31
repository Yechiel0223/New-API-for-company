package plugins_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoubaoSeedanceSubmitEstimateIncludesTerminalFrame(t *testing.T) {
	source, err := builtinplugins.Source("doubao")
	require.NoError(t, err)
	registry := jsplugin.NewRegistry()
	plugin, err := registry.RegisterFactory(source, jsplugin.Options{Key: "doubao"})
	require.NoError(t, err)

	testCases := []struct {
		name       string
		resolution string
		seconds    int
		tokens     float64
	}{
		{name: "480p 5s", resolution: "480p", seconds: 5, tokens: 48_437.8125},
		{name: "720p 4s", resolution: "720p", seconds: 4, tokens: 87_300},
		{name: "720p 5s", resolution: "720p", seconds: 5, tokens: 108_900},
		{name: "720p 10s", resolution: "720p", seconds: 10, tokens: 216_900},
		{name: "720p 30s", resolution: "720p", seconds: 30, tokens: 648_900},
		{name: "1080p 5s", resolution: "1080p", seconds: 5, tokens: 245_025},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			value, callErr := plugin.Engine.Call(t.Context(), "extractUsage", map[string]any{
				"model":         "doubao-seedance-2-5-260628",
				"upstreamModel": "doubao-seedance-2-5-260628",
				"requestBody": map[string]any{
					"seconds": float64(testCase.seconds),
					"metadata": map[string]any{
						"resolution": testCase.resolution,
						"content":    []any{},
					},
				},
			})
			require.NoError(t, callErr)
			encoded, marshalErr := common.Marshal(value)
			require.NoError(t, marshalErr)
			var facts map[string]any
			require.NoError(t, common.Unmarshal(encoded, &facts))
			assert.Equal(t, testCase.tokens, facts["tokens"])
		})
	}
}
