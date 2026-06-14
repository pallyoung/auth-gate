package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type HostProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type HostEntry struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Position  int       `json:"position"`
	IP        string    `json:"ip"`
	Hostnames string    `json:"hostnames"`
	Comment   string    `json:"comment"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HostProfileListResponse struct {
	Profiles []HostProfile `json:"profiles"`
	ActiveID string        `json:"active_id"`
}

type HostProfileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type HostEntryRequest struct {
	IP        string   `json:"ip"`
	Comment   string   `json:"comment"`
	Hostnames []string `json:"hostnames"`
	Enabled   bool     `json:"enabled"`
}

type HostEntryReorderRequest struct {
	EntryIDs []string `json:"entry_ids"`
}

func HostProfileResponse(p store.HostProfile) HostProfile {
	return HostProfile{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func HostEntryResponse(e store.HostEntry) HostEntry {
	return HostEntry{
		ID:        e.ID,
		ProfileID: e.ProfileID,
		Position:  e.Position,
		IP:        e.IP,
		Hostnames: e.Hostnames,
		Comment:   e.Comment,
		Enabled:   e.Enabled,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func HostEntryListResponse(entries []store.HostEntry) []HostEntry {
	out := make([]HostEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, HostEntryResponse(e))
	}
	return out
}

func HostProfileListEnvelope(profiles []store.HostProfile, activeID string) HostProfileListResponse {
	out := make([]HostProfile, 0, len(profiles))
	var foundActive string
	for _, p := range profiles {
		out = append(out, HostProfileResponse(p))
		if p.IsActive {
			foundActive = p.ID
		}
	}
	if foundActive != "" {
		activeID = foundActive
	}
	return HostProfileListResponse{Profiles: out, ActiveID: activeID}
}