package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriceDataShouldPreConsume(t *testing.T) {
	assert.True(t, (PriceData{}).ShouldPreConsume())
	assert.False(t, (PriceData{FreeModel: true}).ShouldPreConsume())
	assert.False(t, (PriceData{Postpaid: true}).ShouldPreConsume())
}
