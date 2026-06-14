package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/api/dto"
	hostservice "github.com/pallyoung/auth-gate/packages/server/internal/service/hosts"
)

func listHostProfiles(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		profiles, err := svc.ListProfiles()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		activeID := c.Query("active_id")
		c.JSON(http.StatusOK, dto.HostProfileListEnvelope(profiles, activeID))
	}
}

func getHostProfile(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := svc.GetProfile(c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.HostProfileResponse(*p))
	}
}

func createHostProfile(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.HostProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}
		p, err := svc.CreateProfile(hostservice.ProfileInput{Name: req.Name, Description: req.Description})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.HostProfileResponse(*p))
	}
}

func updateHostProfile(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.HostProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}
		p, err := svc.UpdateProfile(c.Param("id"), hostservice.ProfileInput{Name: req.Name, Description: req.Description})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.HostProfileResponse(*p))
	}
}

func deleteHostProfile(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.DeleteProfile(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func activateHostProfile(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := svc.ActivateProfile(c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.HostProfileResponse(*p))
	}
}

func listHostEntries(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		entries, err := svc.ListEntries(c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.HostEntryListResponse(entries))
	}
}

func createHostEntry(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.HostEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}
		e, err := svc.CreateEntry(c.Param("id"), hostservice.EntryInput{
			IP:        req.IP,
			Comment:   req.Comment,
			Hostnames: req.Hostnames,
			Enabled:   req.Enabled,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.HostEntryResponse(*e))
	}
}

func updateHostEntry(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.HostEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}
		e, err := svc.UpdateEntry(c.Param("id"), c.Param("eid"), hostservice.EntryInput{
			IP:        req.IP,
			Comment:   req.Comment,
			Hostnames: req.Hostnames,
			Enabled:   req.Enabled,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.HostEntryResponse(*e))
	}
}

func reorderHostEntries(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.HostEntryReorderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}
		if err := svc.ReorderEntries(c.Param("id"), req.EntryIDs); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func deleteHostEntry(svc HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.DeleteEntry(c.Param("id"), c.Param("eid")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}