package userpb_test

import (
	"encoding/json"
	"testing"
	"time"

	userpb "github.com/ashkrai/protobuf-user/internal/user"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func sampleUserV2(t *testing.T) *userpb.User {
	t.Helper()
	now := time.Now().Unix()
	return &userpb.User{
		Id:        "usr_test_001",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Role:      userpb.Role_ADMIN,
		Status:    userpb.AccountStatus_ACTIVE,
		Contact: &userpb.ContactInfo{
			Email:       "test@example.com",
			Phone:       "+1-555-0100",
			SocialLinks: []string{"https://github.com/test"},
			Website:     "https://test.dev",
		},
		Address: &userpb.Address{
			Street:    "1 Test St",
			City:      "Testville",
			State:     "TS",
			Country:   "Testland",
			Zip:       "00001",
			Formatted: "1 Test St, Testville TS 00001",
		},
		Preferences: &userpb.Preferences{
			Language:        "en",
			Timezone:        "UTC",
			EmailNewsletter: true,
			DarkMode:        true,
			ThemeColor:      "#ffffff",
		},
		GroupIds:      []string{"grp_a", "grp_b"},
		Metadata:      map[string]string{"key1": "val1", "key2": "val2"},
		CreatedAtUnix: now - 1000,
		UpdatedAtUnix: now,
		SubscriptionTier: userpb.SubscriptionTier_PRO,
		Audit: &userpb.AuditInfo{
			LastLoginIp:   "1.2.3.4",
			LastLoginUnix: now - 60,
			LoginCount:    42,
			MfaEnabled:    true,
		},
		DisplayName: "Test Display",
		BadgeIds:    []string{"badge_a", "badge_b"},
	}
}

// ── round-trip tests ──────────────────────────────────────────────────────────

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	u := sampleUserV2(t)

	bin, err := u.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(bin) == 0 {
		t.Fatal("marshal produced empty bytes")
	}

	got := &userpb.User{}
	if err := got.Unmarshal(bin); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Compare via JSON for readable diff
	origJSON, _ := json.Marshal(u)
	gotJSON, _ := json.Marshal(got)
	if string(origJSON) != string(gotJSON) {
		t.Errorf("round-trip mismatch:\norig: %s\ngot:  %s", origJSON, gotJSON)
	}
}

func TestMarshalUnmarshal_ScalarFields(t *testing.T) {
	tests := []struct {
		name string
		user *userpb.User
	}{
		{
			name: "minimal user",
			user: &userpb.User{Id: "u1", Username: "min"},
		},
		{
			name: "all roles",
			user: &userpb.User{Id: "u2", Role: userpb.Role_SUPER_ADMIN},
		},
		{
			name: "all statuses",
			user: &userpb.User{Id: "u3", Status: userpb.AccountStatus_SUSPENDED},
		},
		{
			name: "all tiers",
			user: &userpb.User{Id: "u4", SubscriptionTier: userpb.SubscriptionTier_ENTERPRISE},
		},
		{
			name: "empty groups",
			user: &userpb.User{Id: "u5", GroupIds: []string{}},
		},
		{
			name: "multiple groups",
			user: &userpb.User{Id: "u6", GroupIds: []string{"g1", "g2", "g3"}},
		},
		{
			name: "multiple badges",
			user: &userpb.User{Id: "u7", BadgeIds: []string{"b1", "b2"}},
		},
		{
			name: "metadata map",
			user: &userpb.User{Id: "u8", Metadata: map[string]string{"a": "1", "b": "2"}},
		},
		{
			name: "large unix timestamps",
			user: &userpb.User{Id: "u9", CreatedAtUnix: 1700000000, UpdatedAtUnix: 1700086400},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bin, err := tc.user.Marshal()
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			got := &userpb.User{}
			if err := got.Unmarshal(bin); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			origJ, _ := json.Marshal(tc.user)
			gotJ, _ := json.Marshal(got)
			if string(origJ) != string(gotJ) {
				t.Errorf("mismatch:\norig: %s\ngot:  %s", origJ, gotJ)
			}
		})
	}
}

func TestMarshal_EmptyUser(t *testing.T) {
	u := &userpb.User{}
	bin, err := u.Marshal()
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	// An empty user should produce 0 bytes (all fields are zero/empty)
	if len(bin) != 0 {
		t.Errorf("expected 0 bytes for empty user, got %d", len(bin))
	}
	got := &userpb.User{}
	if err := got.Unmarshal(bin); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
}

// ── size comparison test ──────────────────────────────────────────────────────

func TestBinarySmallerThanJSON(t *testing.T) {
	u := sampleUserV2(t)

	bin, _ := u.Marshal()
	jsonBytes, _ := json.Marshal(u)

	if len(bin) >= len(jsonBytes) {
		t.Errorf("expected binary (%d) < JSON (%d)", len(bin), len(jsonBytes))
	}
	ratio := float64(len(jsonBytes)) / float64(len(bin))
	t.Logf("Binary: %d bytes, JSON: %d bytes, JSON is %.2f× larger", len(bin), len(jsonBytes), ratio)
}

// ── backward compatibility tests ──────────────────────────────────────────────

func TestBackwardCompat_V1ReaderIgnoresV2Fields(t *testing.T) {
	// Build a v2 user with v2-only fields set
	u := sampleUserV2(t)
	bin, _ := u.Marshal()

	// Simulate a v1 reader (only reads fields 1–13)
	v1 := &userpb.User{}
	if err := v1.UnmarshalMaxField(bin, 13); err != nil {
		t.Fatalf("v1 unmarshal: %v", err)
	}

	// v1 fields should be populated
	if v1.Id != u.Id {
		t.Errorf("id: want %q, got %q", u.Id, v1.Id)
	}
	if v1.Username != u.Username {
		t.Errorf("username: want %q, got %q", u.Username, v1.Username)
	}
	if v1.Role != u.Role {
		t.Errorf("role: want %v, got %v", u.Role, v1.Role)
	}
	if v1.Contact == nil || v1.Contact.Email != u.Contact.Email {
		t.Errorf("contact email: want %q, got %v", u.Contact.Email, v1.Contact)
	}

	// v2 fields should be zero/nil
	if v1.SubscriptionTier != userpb.SubscriptionTier_UNSPECIFIED {
		t.Errorf("subscription_tier should be zero, got %v", v1.SubscriptionTier)
	}
	if v1.Audit != nil {
		t.Errorf("audit should be nil, got %v", v1.Audit)
	}
	if v1.DisplayName != "" {
		t.Errorf("display_name should be empty, got %q", v1.DisplayName)
	}
	if len(v1.BadgeIds) != 0 {
		t.Errorf("badge_ids should be empty, got %v", v1.BadgeIds)
	}
	t.Log("✔ v1 reader correctly ignored v2 fields")
}

func TestForwardCompat_V2ReaderHandlesV1Binary(t *testing.T) {
	// Build a v1-style user (no v2 fields)
	now := time.Now().Unix()
	v1User := &userpb.User{
		Id:            "usr_v1_legacy",
		Username:      "legacy",
		Role:          userpb.Role_MEMBER,
		Status:        userpb.AccountStatus_ACTIVE,
		Contact:       &userpb.ContactInfo{Email: "legacy@example.com"},
		CreatedAtUnix: now - 86400,
		UpdatedAtUnix: now,
	}
	v1Bin, _ := v1User.Marshal()

	// v2 reader reads v1 binary — should not error
	v2 := &userpb.User{}
	if err := v2.Unmarshal(v1Bin); err != nil {
		t.Fatalf("v2 reader failed on v1 binary: %v", err)
	}

	// v1 fields present
	if v2.Id != v1User.Id {
		t.Errorf("id: want %q, got %q", v1User.Id, v2.Id)
	}
	if v2.Contact == nil || v2.Contact.Email != v1User.Contact.Email {
		t.Errorf("contact email mismatch")
	}

	// v2-only fields should be zero
	if v2.SubscriptionTier != userpb.SubscriptionTier_UNSPECIFIED {
		t.Errorf("tier should be zero, got %v", v2.SubscriptionTier)
	}
	if v2.Audit != nil {
		t.Errorf("audit should be nil, got %v", v2.Audit)
	}
	t.Log("✔ v2 reader handled v1 binary gracefully")
}

// ── enum tests ────────────────────────────────────────────────────────────────

func TestEnumStrings(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Role_UNSPECIFIED", userpb.Role_UNSPECIFIED.String(), "ROLE_UNSPECIFIED"},
		{"Role_GUEST", userpb.Role_GUEST.String(), "ROLE_GUEST"},
		{"Role_ADMIN", userpb.Role_ADMIN.String(), "ROLE_ADMIN"},
		{"Role_SUPER_ADMIN", userpb.Role_SUPER_ADMIN.String(), "ROLE_SUPER_ADMIN"},
		{"AccountStatus_ACTIVE", userpb.AccountStatus_ACTIVE.String(), "ACCOUNT_STATUS_ACTIVE"},
		{"AccountStatus_SUSPENDED", userpb.AccountStatus_SUSPENDED.String(), "ACCOUNT_STATUS_SUSPENDED"},
		{"SubscriptionTier_PRO", userpb.SubscriptionTier_PRO.String(), "SUBSCRIPTION_TIER_PRO"},
		{"SubscriptionTier_ENTERPRISE", userpb.SubscriptionTier_ENTERPRISE.String(), "SUBSCRIPTION_TIER_ENTERPRISE"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %q, got %q", tc.want, tc.got)
			}
		})
	}
}

func TestEnumJSONMarshal(t *testing.T) {
	u := &userpb.User{
		Role:             userpb.Role_SUPER_ADMIN,
		Status:           userpb.AccountStatus_ACTIVE,
		SubscriptionTier: userpb.SubscriptionTier_ENTERPRISE,
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{`"ROLE_SUPER_ADMIN"`, `"ACCOUNT_STATUS_ACTIVE"`, `"SUBSCRIPTION_TIER_ENTERPRISE"`} {
		if !contains(s, want) {
			t.Errorf("JSON %q not found in %s", want, s)
		}
	}
}

func TestNestedMessages_NilSafe(t *testing.T) {
	u := &userpb.User{Id: "u1"} // no nested messages set
	bin, err := u.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := &userpb.User{}
	if err := got.Unmarshal(bin); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Contact != nil {
		t.Errorf("contact should be nil")
	}
	if got.Address != nil {
		t.Errorf("address should be nil")
	}
	if got.Preferences != nil {
		t.Errorf("preferences should be nil")
	}
	if got.Audit != nil {
		t.Errorf("audit should be nil")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}
