package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/models"
	"github.com/tanok/tanok-web-api/internal/services/cron"
)

func (h *Handler) RunDailyGeneration(c *gin.Context) {
	force := c.Query("force") == "true" || c.Query("force") == "1"
	skipIndexNow := c.Query("skipIndexNow") == "true" || c.Query("skipIndexNow") == "1"
	dryRun := c.Query("dryRun") == "true" || c.Query("dryRun") == "1"

	summary := cron.RunDailyGeneration(c.Request.Context(), h.Services, cron.RunDailyOptions{
		Force:        force,
		SkipIndexNow: skipIndexNow,
		DryRun:       dryRun,
	})

	statusCode := http.StatusOK
	if summary.Status == "FAILED" {
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, models.CronResponse{
		Status:  summary.Status,
		PostID:  summary.ESPostID,
		Slot:    summary.Slot,
		Message: summary.Error,
	})
}
