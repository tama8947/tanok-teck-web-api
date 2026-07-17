package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/models"
)

func (h *Handler) TrackEvent(c *gin.Context) {
	var req models.TrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields: type, path"})
		return
	}

	if req.Type == "" || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields: type, path"})
		return
	}

	validTypes := map[models.EventType]bool{
		models.EventPageview: true,
		models.EventClick:    true,
		models.EventScroll:   true,
		models.EventDuration: true,
	}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid event type. Must be one of: pageview, click, scroll, duration")})
		return
	}

	visitorID := c.GetHeader("x-visitor-id")
	if visitorID == "" {
		userAgent := c.GetHeader("User-Agent")
		forwardedFor := c.GetHeader("x-forwarded-for")
		ip := "unknown"
		if forwardedFor != "" {
			commaIdx := -1
			for i, ch := range forwardedFor {
				if ch == ',' {
					commaIdx = i
					break
				}
			}
			if commaIdx > 0 {
				ip = forwardedFor[:commaIdx]
			} else {
				ip = forwardedFor
			}
		}
		hash := md5.Sum([]byte(ip + "-" + userAgent))
		visitorID = fmt.Sprintf("%x", hash)
	}

	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" {
		userAgent = "unknown"
	}

	geo := buildGeoJSON(c)

	result, err := h.Services.TrackEvent(c.Request.Context(), req, visitorID, userAgent, geo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func buildGeoJSON(c *gin.Context) string {
	country := c.GetHeader("cf-ipcountry")
	if country == "" {
		country = c.GetHeader("x-vercel-ip-country")
	}
	if country == "" {
		country = "unknown"
	}

	city := c.GetHeader("x-vercel-ip-city")
	if city == "" {
		city = "unknown"
	}

	region := c.GetHeader("x-vercel-country-region")
	if region == "" {
		region = "unknown"
	}

	data := map[string]string{
		"country": country,
		"city":    city,
		"region":  region,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func (h *Handler) GetStats(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil || days < 1 {
		days = 7
	}

	result, err := h.Services.GetStats(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetEvents(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	typeFilter := c.Query("type")
	pathFilter := c.Query("path")

	events, err := h.Services.GetEvents(c.Request.Context(), limit, typeFilter, pathFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *Handler) GetTopElements(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil || days < 1 {
		days = 7
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "5"))
	if err != nil || limit < 1 {
		limit = 5
	}

	result, err := h.Services.GetTopElements(c.Request.Context(), days, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top elements"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetEngagement(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil || days < 1 {
		days = 7
	}

	result, err := h.Services.GetEngagement(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch engagement data"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetHeatmap(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query param is required"})
		return
	}

	points, err := h.Services.GetHeatmap(c.Request.Context(), path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch heatmap data"})
		return
	}

	c.JSON(http.StatusOK, points)
}

func (h *Handler) GetTrafficSources(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil || days < 1 {
		days = 7
	}

	result, err := h.Services.GetTrafficSources(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch traffic sources"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Analyze(c *gin.Context) {
	var req models.AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	result, err := h.Services.AnalyzeWithAI(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze"})
		return
	}

	c.JSON(http.StatusOK, result)
}
