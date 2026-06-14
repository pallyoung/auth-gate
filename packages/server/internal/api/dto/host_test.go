package dto

import (
	"testing"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestHostProfileResponseRoundTrip(t *testing.T) {
	created := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC)
	profile := store.HostProfile{
		ID:          "profile-1",
		Name:        "dev",
		Description: "developer hosts",
		IsActive:    true,
		CreatedAt:   created,
		UpdatedAt:   updated,
	}

	resp := HostProfileResponse(profile)
	if resp.ID != profile.ID {
		t.Errorf("ID = %q, want %q", resp.ID, profile.ID)
	}
	if resp.Name != profile.Name {
		t.Errorf("Name = %q, want %q", resp.Name, profile.Name)
	}
	if resp.Description != profile.Description {
		t.Errorf("Description = %q, want %q", resp.Description, profile.Description)
	}
	if !resp.IsActive {
		t.Errorf("IsActive = false, want true")
	}
	if !resp.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, created)
	}
	if !resp.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt = %v, want %v", resp.UpdatedAt, updated)
	}
}

func TestHostEntryResponseRoundTrip(t *testing.T) {
	created := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC)
	entry := store.HostEntry{
		ID:        "entry-1",
		ProfileID: "profile-1",
		Position:  0,
		IP:        "127.0.0.1",
		Hostnames: "a.local b.local",
		Comment:   "local dev",
		Enabled:   true,
		CreatedAt: created,
		UpdatedAt: updated,
	}

	resp := HostEntryResponse(entry)
	if resp.ID != entry.ID {
		t.Errorf("ID = %q, want %q", resp.ID, entry.ID)
	}
	if resp.Hostnames != entry.Hostnames {
		t.Errorf("Hostnames = %q, want %q", resp.Hostnames, entry.Hostnames)
	}
	if !resp.Enabled {
		t.Errorf("Enabled = false, want true")
	}
}

func TestHostEntryListResponsePreservesOrder(t *testing.T) {
	entries := []store.HostEntry{
		{ID: "e1", ProfileID: "p1", Position: 0, IP: "10.0.0.1", Hostnames: "a", Enabled: true},
		{ID: "e2", ProfileID: "p1", Position: 1, IP: "10.0.0.2", Hostnames: "b", Enabled: true},
	}

	list := HostEntryListResponse(entries)
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	if list[0].ID != "e1" || list[1].ID != "e2" {
		t.Errorf("order = [%s, %s], want [e1, e2]", list[0].ID, list[1].ID)
	}
}

func TestHostProfileListEnvelopePicksActiveID(t *testing.T) {
	profiles := []store.HostProfile{
		{ID: "p1", Name: "dev"},
		{ID: "p2", Name: "prod", IsActive: true},
		{ID: "p3", Name: "staging"},
	}

	envelope := HostProfileListEnvelope(profiles, "")
	if len(envelope.Profiles) != 3 {
		t.Fatalf("len(Profiles) = %d, want 3", len(envelope.Profiles))
	}
	if envelope.ActiveID != "p2" {
		t.Errorf("ActiveID = %q, want %q", envelope.ActiveID, "p2")
	}
}

func TestHostProfileListEnvelopePrefersStoredActiveOverCaller(t *testing.T) {
	profiles := []store.HostProfile{
		{ID: "p1", Name: "dev", IsActive: true},
		{ID: "p2", Name: "prod"},
	}

	envelope := HostProfileListEnvelope(profiles, "p2")
	if envelope.ActiveID != "p1" {
		t.Errorf("ActiveID = %q, want %q (stored active profile should win over caller-supplied id)", envelope.ActiveID, "p1")
	}
}

func TestHostProfileListEnvelopeEmptyWhenNoActive(t *testing.T) {
	profiles := []store.HostProfile{
		{ID: "p1", Name: "dev"},
		{ID: "p2", Name: "prod"},
	}

	envelope := HostProfileListEnvelope(profiles, "")
	if envelope.ActiveID != "" {
		t.Errorf("ActiveID = %q, want empty", envelope.ActiveID)
	}
	if len(envelope.Profiles) != 2 {
		t.Errorf("len(Profiles) = %d, want 2", len(envelope.Profiles))
	}
}