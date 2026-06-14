package store

import "testing"

func TestCreateAndListHostProfiles(t *testing.T) {
	db := newTestSQLite(t)

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
	db := newTestSQLite(t)
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
	db := newTestSQLite(t)
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

func TestGetHostProfile_NotFound(t *testing.T) {
	db := newTestSQLite(t)
	_, err := db.GetHostProfile("missing")
	if err == nil {
		t.Fatal("GetHostProfile() error = nil, want sql.ErrNoRows")
	}
}
