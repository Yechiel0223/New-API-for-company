package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type modelUsageAnalyticsResponse struct {
	Success bool                         `json:"success"`
	Message string                       `json:"message"`
	Data    service.ModelAnalyticsResult `json:"data"`
}

type modelUsageHealthResponse struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message"`
	Data    service.ModelHealthResult `json:"data"`
}

func setupModelUsageAnalyticsControllerTestDB(t *testing.T) {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ModelUsageEvent{}, &model.ModelUsageAttempt{}))
}

func TestGetModelUsageAnalyticsRejectsInvalidParameters(t *testing.T) {
	for _, target := range []string{
		"/analytics?start_timestamp=20&end_timestamp=10",
		"/analytics?start_timestamp=0&end_timestamp=10",
		"/analytics?start_timestamp=10&end_timestamp=20&granularity=month",
		"/analytics?start_timestamp=10&end_timestamp=20&models=model-a,",
	} {
		t.Run(target, func(t *testing.T) {
			r := gin.New()
			r.GET("/analytics", GetModelUsageAnalytics)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestGetModelUsageAnalyticsReturnsServiceResult(t *testing.T) {
	setupModelUsageAnalyticsControllerTestDB(t)
	start := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	require.NoError(t, model.DB.Create(&model.ModelUsageEvent{
		EventKey: "request:analytics", ModelName: "model-a", Username: "alice",
		Status: model.ModelUsageStatusSuccess, SubmittedAt: start.Add(time.Minute).Unix(),
		CompletedAt: start.Add(2 * time.Minute).Unix(), UpdatedAt: start.Add(2 * time.Minute).Unix(),
	}).Error)

	r := gin.New()
	r.GET("/analytics", GetModelUsageAnalytics)
	rec := httptest.NewRecorder()
	target := "/analytics?start_timestamp=" + strconv.FormatInt(start.Unix(), 10) + "&end_timestamp=" + strconv.FormatInt(start.Add(time.Hour).Unix(), 10) + "&username=alice&models=model-a"
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var payload modelUsageAnalyticsResponse
	require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &payload))
	require.True(t, payload.Success, payload.Message)
	assert.Equal(t, "Asia/Shanghai", payload.Data.Range.Timezone)
	assert.Equal(t, "hour", string(payload.Data.Range.Granularity))
	assert.Equal(t, int64(1), payload.Data.Summary.TotalCalls)
	assert.NotNil(t, payload.Data.Series)
	assert.NotNil(t, payload.Data.Models)
	assert.Equal(t, []string{"model-a"}, payload.Data.AvailableModels)
	assert.NotZero(t, payload.Data.UpdatedAt)
}

func TestGetModelUsageHealthAcceptsBoundedWindows(t *testing.T) {
	setupModelUsageAnalyticsControllerTestDB(t)
	r := gin.New()
	r.GET("/health", GetModelUsageHealth)

	for _, hours := range []string{"1", "24", "168", "720"} {
		t.Run(hours, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health?hours="+hours, nil))
			require.Equal(t, http.StatusOK, rec.Code)
			var payload modelUsageHealthResponse
			require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &payload))
			require.True(t, payload.Success, payload.Message)
		})
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health?hours=721", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
