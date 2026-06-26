package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const settingLogRetentionDays = "log_retention_days"

func getLogRetention(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, err := db.GetSetting(settingLogRetentionDays)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "read_failed", "failed to read setting")
			return
		}
		days, _ := strconv.Atoi(val)
		c.JSON(http.StatusOK, gin.H{"days": days})
	}
}

func updateLogRetention(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Days int `json:"days"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if req.Days < 0 {
			writeError(c, http.StatusBadRequest, "invalid_request", "days must be >= 0")
			return
		}
		if err := db.SetSetting(settingLogRetentionDays, strconv.Itoa(req.Days)); err != nil {
			writeError(c, http.StatusInternalServerError, "save_failed", "failed to save setting")
			return
		}
		c.JSON(http.StatusOK, gin.H{"days": req.Days})
	}
}
