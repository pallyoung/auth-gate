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
