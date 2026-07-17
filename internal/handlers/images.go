package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GenerateCoverImage(c *gin.Context) {
	var body struct {
		Title  string `json:"title" binding:"required"`
		Locale string `json:"locale" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and locale are required"})
		return
	}

	url, err := h.Services.GenerateCoverImage(c.Request.Context(), body.Title, body.Locale)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coverImage": url})
}
