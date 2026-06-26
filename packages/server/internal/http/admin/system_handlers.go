package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	systemservice "github.com/pallyoung/auth-gate/packages/server/internal/service/system"
)

func getSystemStats(svc *systemservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := svc.Stats()
		if err != nil {
			writeError(c, http.StatusInternalServerError, "stats_failed", "failed to collect system stats")
			return
		}
		c.JSON(http.StatusOK, stats)
	}
}
