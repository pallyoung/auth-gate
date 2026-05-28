package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestUserResponse_JSONIncludesDisabledState(t *testing.T) {
	payload, err := json.Marshal(UserResponse(store.User{
		ID:        "user-1",
		Username:  "disabled-user",
		Role:      store.RoleMember,
		Enabled:   false,
		CreatedAt: time.Date(2026, time.May, 27, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.May, 27, 12, 30, 0, 0, time.UTC),
	}))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	enabled, ok := decoded["enabled"]
	if !ok {
		t.Fatalf("response JSON missing enabled field: %s", string(payload))
	}
	if enabled != false {
		t.Fatalf("response enabled = %v, want false", enabled)
	}
}
