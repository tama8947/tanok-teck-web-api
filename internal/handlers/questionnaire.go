package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/models"
)

func (h *Handler) SendWebQuestionnaire(c *gin.Context) {
	var req models.QuestionnaireRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and email are required"})
		return
	}

	if req.Name == "" || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and email are required"})
		return
	}

	if err := h.Services.SendWebQuestionnaire(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send questionnaire"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
