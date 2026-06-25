package admin

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/config"
)

type listenEntryResponse struct {
	Addr string `json:"addr"`
	TLS  bool   `json:"tls"`
}

type configResponse struct {
	Listen []listenEntryResponse `json:"listen"`
}

type listenEntryRequest struct {
	Addr string `json:"addr"`
	TLS  bool   `json:"tls"`
}

type configUpdateRequest struct {
	Listen []listenEntryRequest `json:"listen"`
}

func getConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		listen := make([]listenEntryResponse, 0, len(cfg.Server.Listen))
		for _, e := range cfg.Server.Listen {
			listen = append(listen, listenEntryResponse{Addr: e.Addr, TLS: e.TLS})
		}
		// If no entries configured, return defaults
		if len(listen) == 0 {
			listen = []listenEntryResponse{{Addr: ":80", TLS: false}}
		}
		c.JSON(http.StatusOK, configResponse{Listen: listen})
	}
}

func updateConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req configUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		if len(req.Listen) == 0 {
			writeError(c, http.StatusBadRequest, "invalid_request", "at least one listen address is required")
			return
		}

		entries := make([]config.ListenEntry, 0, len(req.Listen))
		for _, e := range req.Listen {
			if e.Addr == "" {
				continue
			}
			entries = append(entries, config.ListenEntry{Addr: e.Addr, TLS: e.TLS})
		}

		if len(entries) == 0 {
			writeError(c, http.StatusBadRequest, "invalid_request", "at least one valid listen address is required")
			return
		}

		cfg.Server.Listen = entries

		if err := cfg.Save(); err != nil {
			writeError(c, http.StatusInternalServerError, "save_failed", "failed to save config: "+err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "config saved, restart required to apply changes"})

		// Trigger graceful restart after response is sent
		go func() {
			p, _ := os.FindProcess(os.Getpid())
			if p != nil {
				p.Signal(os.Interrupt)
			}
		}()
	}
}
