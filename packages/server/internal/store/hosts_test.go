package store

import (
	"database/sql"
	"errors"
	"testing"
)

func TestCreateAndListHostProfiles(t *testing.T) {
	db := newTestStore(t)

	profiles, err := db.ListHostProfiles()
	if err != nil {
		t.Fatalf("ListHostProfiles() error = %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("len(profiles) = %d, want 0", len(profiles))
	}

	p := &HostProfile{Name: "dev", Description: "dev hosts"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}
	if p.ID == "" {
		t.Fatal("CreateHostProfile() did not assign ID")
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Fatal("CreateHostProfile() did not assign timestamps")
	}

	profiles, err = db.ListHostProfiles()
	if err != nil {
		t.Fatalf("ListHostProfiles() error = %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("len(profiles) = %d, want 1", len(profiles))
	}
	if profiles[0].Name != "dev" {
		t.Fatalf("profiles[0].Name = %q, want %q", profiles[0].Name, "dev")
	}
	if profiles[0].IsActive {
		t.Fatal("profiles[0].IsActive = true, want false")
	}
}

func TestGetHostProfile(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}

	got, err := db.GetHostProfile(p.ID)
	if err != nil {
		t.Fatalf("GetHostProfile() error = %v", err)
	}
	if got.ID != p.ID {
		t.Fatalf("got.ID = %q, want %q", got.ID, p.ID)
	}
	if got.Name != "dev" {
		t.Fatalf("got.Name = %q, want %q", got.Name, "dev")
	}
}

func TestUpdateHostProfile(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}

	p.Name = "staging"
	p.Description = "staging cluster"
	if err := db.UpdateHostProfile(p); err != nil {
		t.Fatalf("UpdateHostProfile() error = %v", err)
	}

	got, err := db.GetHostProfile(p.ID)
	if err != nil {
		t.Fatalf("GetHostProfile() error = %v", err)
	}
	if got.Name != "staging" {
		t.Fatalf("got.Name = %q, want %q", got.Name, "staging")
	}
	if got.Description != "staging cluster" {
		t.Fatalf("got.Description = %q, want %q", got.Description, "staging cluster")
	}
	if !got.UpdatedAt.After(p.CreatedAt) {
		t.Fatalf("got.UpdatedAt (%v) should be after CreatedAt (%v)", got.UpdatedAt, p.CreatedAt)
	}
}

func TestDeleteHostProfile_CascadesEntries(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}
	e := &HostEntry{ProfileID: p.ID, Position: 0, IP: "127.0.0.1", Hostnames: "api.local"}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}

	if err := db.DeleteHostProfile(p.ID); err != nil {
		t.Fatalf("DeleteHostProfile() error = %v", err)
	}

	if _, err := db.GetHostProfile(p.ID); err == nil {
		t.Fatal("GetHostProfile() after delete error = nil, want sql.ErrNoRows")
	}
	entries, err := db.ListHostEntries(p.ID)
	if err != nil {
		t.Fatalf("ListHostEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0 (cascade should have removed them)", len(entries))
	}
}

func TestGetHostProfile_NotFound(t *testing.T) {
	db := newTestStore(t)
	_, err := db.GetHostProfile("missing")
	if err == nil {
		t.Fatal("GetHostProfile() error = nil, want sql.ErrNoRows")
	}
}

func TestCreateHostEntry_AssignsIDAndPosition(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}

	e := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "api.local"}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}
	if e.ID == "" {
		t.Fatal("CreateHostEntry() did not assign ID")
	}
	if e.Position == 0 {
		t.Fatal("CreateHostEntry() did not assign Position (still 0)")
	}
	if e.CreatedAt.IsZero() || e.UpdatedAt.IsZero() {
		t.Fatal("CreateHostEntry() did not assign timestamps")
	}
}

func TestCreateHostEntry_StoresFields(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}

	e := &HostEntry{
		ProfileID: p.ID,
		IP:        "10.0.0.1",
		Hostnames: "a.local b.local",
		Comment:   "cluster gateway",
		Enabled:   true,
	}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}

	gotList, err := db.ListHostEntries(p.ID)
	if err != nil {
		t.Fatalf("ListHostEntries() error = %v", err)
	}
	if len(gotList) != 1 {
		t.Fatalf("len(gotList) = %d, want 1", len(gotList))
	}
	got := &gotList[0]
	if got.IP != "10.0.0.1" {
		t.Fatalf("got.IP = %q, want %q", got.IP, "10.0.0.1")
	}
	if got.Hostnames != "a.local b.local" {
		t.Fatalf("got.Hostnames = %q, want %q", got.Hostnames, "a.local b.local")
	}
	if got.Comment != "cluster gateway" {
		t.Fatalf("got.Comment = %q, want %q", got.Comment, "cluster gateway")
	}
	if !got.Enabled {
		t.Fatal("got.Enabled = false, want true")
	}
}

func TestCreateHostEntry_PositionMonotonicallyIncreases(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}

	e1 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "first.local"}
	if err := db.CreateHostEntry(e1); err != nil {
		t.Fatalf("CreateHostEntry(e1) error = %v", err)
	}
	e2 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "second.local"}
	if err := db.CreateHostEntry(e2); err != nil {
		t.Fatalf("CreateHostEntry(e2) error = %v", err)
	}
	e3 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "third.local"}
	if err := db.CreateHostEntry(e3); err != nil {
		t.Fatalf("CreateHostEntry(e3) error = %v", err)
	}

	if e1.Position == 0 || e2.Position == 0 || e3.Position == 0 {
		t.Fatalf("positions not assigned: e1=%d e2=%d e3=%d", e1.Position, e2.Position, e3.Position)
	}
	if !(e1.Position < e2.Position && e2.Position < e3.Position) {
		t.Fatalf("positions not monotonically increasing: e1=%d e2=%d e3=%d", e1.Position, e2.Position, e3.Position)
	}
}

func TestListHostEntries_OrdersByPosition(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}

	e1 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "first.local"}
	if err := db.CreateHostEntry(e1); err != nil {
		t.Fatalf("CreateHostEntry(e1) error = %v", err)
	}
	e2 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "second.local"}
	if err := db.CreateHostEntry(e2); err != nil {
		t.Fatalf("CreateHostEntry(e2) error = %v", err)
	}

	entries, err := db.ListHostEntries(p.ID)
	if err != nil {
		t.Fatalf("ListHostEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].ID != e1.ID || entries[1].ID != e2.ID {
		t.Fatalf("order = [%s, %s], want [%s, %s]", entries[0].ID, entries[1].ID, e1.ID, e2.ID)
	}
}

func TestListHostEntries_UnknownProfileReturnsEmpty(t *testing.T) {
	db := newTestStore(t)
	entries, err := db.ListHostEntries("missing-profile")
	if err != nil {
		t.Fatalf("ListHostEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestGetHostEntry(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}
	e := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "api.local"}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}

	got, err := db.GetHostEntry(e.ID)
	if err != nil {
		t.Fatalf("GetHostEntry() error = %v", err)
	}
	if got.ID != e.ID {
		t.Fatalf("got.ID = %q, want %q", got.ID, e.ID)
	}
	if got.IP != "127.0.0.1" {
		t.Fatalf("got.IP = %q, want %q", got.IP, "127.0.0.1")
	}
	if got.Hostnames != "api.local" {
		t.Fatalf("got.Hostnames = %q, want %q", got.Hostnames, "api.local")
	}
}

func TestGetHostEntry_NotFound(t *testing.T) {
	db := newTestStore(t)
	_, err := db.GetHostEntry("missing")
	if err == nil {
		t.Fatal("GetHostEntry() error = nil, want sql.ErrNoRows")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetHostEntry() error = %v, want sql.ErrNoRows", err)
	}
}

func TestUpdateHostEntry(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}
	e := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "api.local"}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}

	e.IP = "10.0.0.1"
	e.Hostnames = "api.local db.local"
	e.Comment = "gateway"
	e.Enabled = false
	if err := db.UpdateHostEntry(e); err != nil {
		t.Fatalf("UpdateHostEntry() error = %v", err)
	}

	got, err := db.GetHostEntry(e.ID)
	if err != nil {
		t.Fatalf("GetHostEntry() error = %v", err)
	}
	if got.IP != "10.0.0.1" {
		t.Fatalf("got.IP = %q, want %q", got.IP, "10.0.0.1")
	}
	if got.Hostnames != "api.local db.local" {
		t.Fatalf("got.Hostnames = %q, want %q", got.Hostnames, "api.local db.local")
	}
	if got.Comment != "gateway" {
		t.Fatalf("got.Comment = %q, want %q", got.Comment, "gateway")
	}
	if got.Enabled {
		t.Fatal("got.Enabled = true, want false")
	}
	if !got.UpdatedAt.After(e.CreatedAt) {
		t.Fatalf("got.UpdatedAt (%v) should be after CreatedAt (%v)", got.UpdatedAt, e.CreatedAt)
	}
}

func TestUpdateHostEntry_NotFound(t *testing.T) {
	db := newTestStore(t)
	e := &HostEntry{ID: "missing", IP: "127.0.0.1", Hostnames: "api.local"}
	err := db.UpdateHostEntry(e)
	if err == nil {
		t.Fatal("UpdateHostEntry() error = nil, want sql.ErrNoRows")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("UpdateHostEntry() error = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteHostEntry(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}
	e := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "api.local"}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}

	if err := db.DeleteHostEntry(e.ID); err != nil {
		t.Fatalf("DeleteHostEntry() error = %v", err)
	}

	if _, err := db.GetHostEntry(e.ID); err == nil {
		t.Fatal("GetHostEntry() after delete error = nil, want sql.ErrNoRows")
	}
	entries, err := db.ListHostEntries(p.ID)
	if err != nil {
		t.Fatalf("ListHostEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestDeleteHostEntry_NotFoundIsNoOp(t *testing.T) {
	db := newTestStore(t)
	if err := db.DeleteHostEntry("missing"); err != nil {
		t.Fatalf("DeleteHostEntry() error = %v, want nil (idempotent)", err)
	}
}

func TestReorderHostEntries(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}
	e1 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "a.local"}
	e2 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "b.local"}
	e3 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "c.local"}
	for _, e := range []*HostEntry{e1, e2, e3} {
		if err := db.CreateHostEntry(e); err != nil {
			t.Fatalf("CreateHostEntry(%s) error = %v", e.ID, err)
		}
	}

	if err := db.ReorderHostEntries(p.ID, []string{e3.ID, e1.ID, e2.ID}); err != nil {
		t.Fatalf("ReorderHostEntries() error = %v", err)
	}

	entries, err := db.ListHostEntries(p.ID)
	if err != nil {
		t.Fatalf("ListHostEntries() error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	wantOrder := []string{e3.ID, e1.ID, e2.ID}
	for i, want := range wantOrder {
		if entries[i].ID != want {
			t.Fatalf("entries[%d].ID = %q, want %q", i, entries[i].ID, want)
		}
		if entries[i].Position != i {
			t.Fatalf("entries[%d].Position = %d, want %d", i, entries[i].Position, i)
		}
	}
}

func TestListEnabledHostEntries(t *testing.T) {
	db := newTestStore(t)
	p := &HostProfile{Name: "dev"}
	if err := db.CreateHostProfile(p); err != nil {
		t.Fatalf("CreateHostProfile() error = %v", err)
	}
	e1 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "a.local", Enabled: true}
	e2 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "b.local", Enabled: false}
	e3 := &HostEntry{ProfileID: p.ID, IP: "127.0.0.1", Hostnames: "c.local", Enabled: true}
	for _, e := range []*HostEntry{e1, e2, e3} {
		if err := db.CreateHostEntry(e); err != nil {
			t.Fatalf("CreateHostEntry(%s) error = %v", e.ID, err)
		}
	}

	enabled, err := db.ListEnabledHostEntries(p.ID)
	if err != nil {
		t.Fatalf("ListEnabledHostEntries() error = %v", err)
	}
	if len(enabled) != 2 {
		t.Fatalf("len(enabled) = %d, want 2", len(enabled))
	}
	if enabled[0].ID != e1.ID || enabled[1].ID != e3.ID {
		t.Fatalf("enabled order = [%s, %s], want [%s, %s]", enabled[0].ID, enabled[1].ID, e1.ID, e3.ID)
	}
	for _, e := range enabled {
		if !e.Enabled {
			t.Fatalf("entry %s.Enabled = false, want true", e.ID)
		}
	}
}

func TestListEnabledHostEntries_EmptyReturnsEmptySlice(t *testing.T) {
	db := newTestStore(t)
	enabled, err := db.ListEnabledHostEntries("missing-profile")
	if err != nil {
		t.Fatalf("ListEnabledHostEntries() error = %v", err)
	}
	if enabled == nil {
		t.Fatal("ListEnabledHostEntries() returned nil, want empty slice")
	}
	if len(enabled) != 0 {
		t.Fatalf("len(enabled) = %d, want 0", len(enabled))
	}
}
