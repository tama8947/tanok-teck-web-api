package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CronSecret(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if provided == "" {
			provided = c.Query("secret")
		}

		if provided == "" || provided != secret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Next()
	}
}
