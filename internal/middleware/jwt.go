package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/services"
)

func JWTAuth(svc *services.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Set("userId", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	cookie, err := c.Cookie("auth_token")
	if err == nil && cookie != "" {
		return cookie
	}

	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}
