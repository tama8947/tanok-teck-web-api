package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/models"
)

var allowedLocales = map[string]bool{"es": true, "en": true}

func (h *Handler) ListPosts(c *gin.Context) {
	locale := c.DefaultQuery("locale", "es")
	if !allowedLocales[locale] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid locale. Allowed: es, en"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	result, err := h.Services.ListPosts(c.Request.Context(), locale, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetPost(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slug"})
		return
	}

	locale := c.DefaultQuery("locale", "es")
	if !allowedLocales[locale] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid locale. Allowed: es, en"})
		return
	}

	post, err := h.Services.GetPostBySlug(c.Request.Context(), locale, slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	c.JSON(http.StatusOK, models.PostResponse{Post: post})
}

func (h *Handler) GetPostSibling(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slug"})
		return
	}

	locale := c.DefaultQuery("locale", "es")
	if !allowedLocales[locale] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid locale. Allowed: es, en"})
		return
	}

	post, err := h.Services.GetPostBySlug(c.Request.Context(), locale, slug)
	if err != nil || post.TranslationGroupID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sibling not found"})
		return
	}

	sibling, err := h.Services.GetSiblingSlug(c.Request.Context(), post.TranslationGroupID, locale)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sibling not found"})
		return
	}

	c.JSON(http.StatusOK, sibling)
}

func atoiOrDefault(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
