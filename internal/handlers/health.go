package handlers

import "github.com/gin-gonic/gin"

func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
