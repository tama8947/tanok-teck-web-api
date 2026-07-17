package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/models"
)

const (
	authCookieName   = "auth_token"
	authCookieMaxAge = 604800
)

func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	token, user, err := h.Services.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(authCookieName, token, authCookieMaxAge, "/", h.Services.Config.CookieDomain, true, true)
	c.JSON(http.StatusOK, models.LoginResponse{Success: true, User: user})
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(authCookieName, "", -1, "/", h.Services.Config.CookieDomain, true, true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) Me(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, err := h.Services.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{User: user})
}
