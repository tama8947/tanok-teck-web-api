package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins string) gin.HandlerFunc {
	origins := strings.Split(allowedOrigins, ",")
	originSet := make(map[string]bool, len(origins))
	for _, o := range origins {
		originSet[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		for _, o := range origins {
			o = strings.TrimSpace(o)
			if o == origin {
				allowed = true
				break
			}
			if strings.HasPrefix(o, "https://*.") && strings.HasSuffix(origin, strings.TrimPrefix(o, "https://*")) {
				allowed = true
				break
			}
			if strings.HasPrefix(o, "http://*.") && strings.HasSuffix(origin, strings.TrimPrefix(o, "http://*")) {
				allowed = true
				break
			}
		}
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Visitor-Id, X-Forwarded-For, Cookie")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
