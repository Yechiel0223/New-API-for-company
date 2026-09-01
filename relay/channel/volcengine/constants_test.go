package volcengine

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestModelListIncludesCompanyArkModels(t *testing.T) {
	assert.Contains(t, ModelList, constant.ModelDoubaoSeedance20)
	assert.Contains(t, ModelList, constant.ModelDoubaoSeedance25)
	assert.Contains(t, ModelList, constant.ModelDoubaoSeedream5Pro)
}
