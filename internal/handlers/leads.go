package handlers

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/models"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func (h *Handler) CaptureLead(c *gin.Context) {
	var req models.LeadCaptureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and email are required"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	normalizedEmail := strings.ToLower(email)
	if !emailRegex.MatchString(normalizedEmail) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}

	req.Name = name
	req.Email = normalizedEmail

	result, err := h.Services.CaptureLead(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	statusCode := http.StatusCreated
	if result.Status == "existing" {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, result)
}

func (h *Handler) ListLeads(c *gin.Context) {
	page := atoiOrDefault(c.DefaultQuery("page", "1"), 1)
	if page < 1 {
		page = 1
	}

	limit := atoiOrDefault(c.DefaultQuery("limit", "20"), 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	result, err := h.Services.ListLeads(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leads"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetLead(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	lead, err := h.Services.GetLeadByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"lead": lead})
}

func (h *Handler) GetMagnetBySlug(c *gin.Context) {
	slug := c.Query("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug is required"})
		return
	}

	magnet, err := h.Services.GetMagnetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Magnet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"magnet": magnet})
}
