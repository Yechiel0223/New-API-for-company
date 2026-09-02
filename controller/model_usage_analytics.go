package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetModelUsageAnalytics(c *gin.Context) {
	query, message := parseModelUsageAnalyticsQuery(c)
	if message != "" {
		modelUsageAnalyticsError(c, http.StatusBadRequest, message)
		return
	}

	result, err := service.QueryModelAnalytics(c.Request.Context(), query)
	if err != nil {
		modelUsageAnalyticsError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": result})
}

func GetModelUsageHealth(c *gin.Context) {
	hours, message := parseModelUsageHealthHours(c)
	if message != "" {
		modelUsageAnalyticsError(c, http.StatusBadRequest, message)
		return
	}

	result, err := service.QueryModelHealth(c.Request.Context(), hours, time.Time{})
	if err != nil {
		modelUsageAnalyticsError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": result})
}

func parseModelUsageAnalyticsQuery(c *gin.Context) (service.ModelAnalyticsQuery, string) {
	start, ok := parsePositiveUnixSeconds(c.Query("start_timestamp"))
	if !ok {
		return service.ModelAnalyticsQuery{}, "invalid start_timestamp"
	}
	end, ok := parsePositiveUnixSeconds(c.Query("end_timestamp"))
	if !ok {
		return service.ModelAnalyticsQuery{}, "invalid end_timestamp"
	}
	if end < start {
		return service.ModelAnalyticsQuery{}, "invalid time range"
	}

	query := service.ModelAnalyticsQuery{StartTimestamp: start, EndTimestamp: end}
	if granularity, present := c.GetQuery("granularity"); present {
		query.Granularity = service.AnalyticsGranularity(granularity)
		if query.Granularity != service.AnalyticsGranularityHour && query.Granularity != service.AnalyticsGranularityDay && query.Granularity != service.AnalyticsGranularityWeek {
			return service.ModelAnalyticsQuery{}, "invalid granularity"
		}
	}
	if username, present := c.GetQuery("username"); present {
		if username != strings.TrimSpace(username) {
			return service.ModelAnalyticsQuery{}, "invalid username"
		}
		query.Username = username
	}
	if rawModels, present := c.GetQuery("models"); present {
		models := strings.Split(rawModels, ",")
		for _, modelName := range models {
			if modelName == "" || modelName != strings.TrimSpace(modelName) {
				return service.ModelAnalyticsQuery{}, "invalid models"
			}
		}
		query.Models = models
	}
	return query, ""
}

func parseModelUsageHealthHours(c *gin.Context) (int, string) {
	rawHours, present := c.GetQuery("hours")
	if !present {
		return 24, ""
	}
	hours, err := strconv.Atoi(rawHours)
	if err != nil || rawHours != strconv.Itoa(hours) || (hours != 1 && hours != 24 && hours != 168) {
		return 0, "invalid hours"
	}
	return hours, ""
}

func parsePositiveUnixSeconds(raw string) (int64, bool) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 || raw != strconv.FormatInt(value, 10) {
		return 0, false
	}
	return value, true
}

func modelUsageAnalyticsError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": false, "message": message})
}
