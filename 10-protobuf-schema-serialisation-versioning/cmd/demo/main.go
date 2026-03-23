package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protowire"

	userpb "github.com/ashkrai/protobuf-user/internal/user"
)

func main() {
	fmt.Println(banner("11 · Protocol Buffers Demo"))

	// ── 1. Build a rich v2 User ───────────────────────────────────────────────
	now := time.Now().Unix()
	user := &userpb.User{
		Id:        "usr_01HX9K2PQRS4T5U6V7W8X9Y0Z",
		Username:  "ashkrai",
		FirstName: "Ashkrai",
		LastName:  "Dev",
		Role:      userpb.Role_ADMIN,
		Status:    userpb.AccountStatus_ACTIVE,
		Contact: &userpb.ContactInfo{
			Email: "ashkrai@example.com",
			Phone: "+91-9876543210",
			SocialLinks: []string{
				"https://github.com/ashkrai",
				"https://twitter.com/ashkrai",
			},
			Website: "https://ashkrai.dev",
		},
		Address: &userpb.Address{
			Street:    "42 Bazaar Road",
			City:      "Varanasi",
			State:     "Uttar Pradesh",
			Country:   "India",
			Zip:       "221001",
			Formatted: "42 Bazaar Road, Varanasi, UP 221001, India",
		},
		Preferences: &userpb.Preferences{
			Language:        "en",
			Timezone:        "Asia/Kolkata",
			EmailNewsletter: true,
			DarkMode:        true,
			ThemeColor:      "#1a73e8",
		},
		GroupIds: []string{"grp_engineers", "grp_backend", "grp_leads"},
		Metadata: map[string]string{
			"signup_source":   "organic",
			"referral_code":   "PROTO2024",
			"onboarding_done": "true",
		},
		CreatedAtUnix:    now - 86400*30,
		UpdatedAtUnix:    now,
		SubscriptionTier: userpb.SubscriptionTier_PRO,
		Audit: &userpb.AuditInfo{
			LastLoginIp:   "103.21.244.0",
			LastLoginUnix: now - 3600,
			LoginCount:    247,
			MfaEnabled:    true,
		},
		DisplayName: "Ashkrai Dev",
		BadgeIds:    []string{"badge_early_adopter", "badge_power_user", "badge_contributor"},
	}

	// ── 2. Serialise to binary ────────────────────────────────────────────────
	section("Step 1 — Serialise to Binary (Protobuf Wire Format)")

	binary, err := user.Marshal()
	must(err, "marshal binary")

	fmt.Printf("  Binary size: %d bytes\n", len(binary))
	preview := binary
	if len(preview) > 64 {
		preview = preview[:64]
	}
	fmt.Printf("  First 64 bytes (hex): %s\n\n", hex.EncodeToString(preview))

	// ── 3. Serialise to JSON ──────────────────────────────────────────────────
	section("Step 2 — Serialise to JSON")

	jsonBytes, err := json.Marshal(user)
	must(err, "marshal JSON")

	jsonPretty, _ := json.MarshalIndent(user, "  ", "  ")
	fmt.Printf("  JSON size: %d bytes\n\n", len(jsonBytes))
	fmt.Printf("  JSON:\n  %s\n\n", string(jsonPretty))

	// ── 4. Size comparison ────────────────────────────────────────────────────
	section("Step 3 — Binary vs JSON Size Comparison")

	ratio := float64(len(jsonBytes)) / float64(len(binary))
	saving := 100.0 - (float64(len(binary))/float64(len(jsonBytes)))*100.0

	fmt.Printf("  ┌──────────────────────────────────────────────┐\n")
	fmt.Printf("  │  Format    │  Size     │  Ratio              │\n")
	fmt.Printf("  ├──────────────────────────────────────────────┤\n")
	fmt.Printf("  │  Protobuf  │  %4d B   │  1.00× (baseline)   │\n", len(binary))
	fmt.Printf("  │  JSON      │  %4d B   │  %.2f× larger        │\n", len(jsonBytes), ratio)
	fmt.Printf("  ├──────────────────────────────────────────────┤\n")
	fmt.Printf("  │  Protobuf is %.0f%% smaller than JSON           │\n", saving)
	fmt.Printf("  └──────────────────────────────────────────────┘\n\n")

	fmt.Println("  Why protobuf is smaller:")
	fmt.Println("  • Field names not stored — only compact field numbers (1, 2, 3…)")
	fmt.Println("  • Integers use varint: small numbers take fewer bytes")
	fmt.Println("  • No quotes, braces, colons, commas, or whitespace overhead")
	fmt.Println("  • Booleans encoded as varint 0/1, not 4–5 chars 'true'/'false'")
	fmt.Println("  • Zero/empty values omitted entirely (proto3 default)\n")

	// ── 5. Deserialise from binary ────────────────────────────────────────────
	section("Step 4 — Deserialise from Binary")

	decoded := &userpb.User{}
	must(decoded.Unmarshal(binary), "unmarshal binary")

	fmt.Printf("  id:             %s\n", decoded.Id)
	fmt.Printf("  username:       %s\n", decoded.Username)
	fmt.Printf("  role:           %s\n", decoded.Role)
	fmt.Printf("  status:         %s\n", decoded.Status)
	fmt.Printf("  email:          %s\n", decoded.Contact.Email)
	fmt.Printf("  city:           %s\n", decoded.Address.City)
	fmt.Printf("  dark_mode:      %v\n", decoded.Preferences.DarkMode)
	fmt.Printf("  groups:         %v\n", decoded.GroupIds)
	fmt.Printf("  tier:           %s\n", decoded.SubscriptionTier)
	fmt.Printf("  login_count:    %d\n", decoded.Audit.LoginCount)
	fmt.Printf("  mfa_enabled:    %v\n", decoded.Audit.MfaEnabled)
	fmt.Printf("  display_name:   %s\n", decoded.DisplayName)
	fmt.Printf("  badges:         %v\n\n", decoded.BadgeIds)

	origJSON, _ := json.Marshal(user)
	decodedJSON, _ := json.Marshal(decoded)
	if string(origJSON) == string(decodedJSON) {
		fmt.Println("  ✔  Round-trip verified: binary → decode → JSON matches original\n")
	} else {
		fmt.Println("  ✘  Round-trip MISMATCH — check encoder/decoder")
		os.Exit(1)
	}

	// ── 6. Backward compat: v1 reader reads v2 binary ────────────────────────
	section("Step 5 — Backward Compatibility (v1 reader reads v2 binary)")

	fmt.Println("  Scenario: service A (v1) receives a binary message from service B (v2).")
	fmt.Println("  v2 adds fields 14–17. v1 reader encounters unknown tags → skips them.")
	fmt.Println()

	v1Decoded := &userpb.User{}
	must(unmarshalV1Only(v1Decoded, binary), "unmarshal v1 only")

	fmt.Printf("  v1 decoded id:              %s\n", v1Decoded.Id)
	fmt.Printf("  v1 decoded username:        %s\n", v1Decoded.Username)
	fmt.Printf("  v1 decoded role:            %s\n", v1Decoded.Role)
	fmt.Printf("  v1 decoded email:           %s\n", v1Decoded.Contact.Email)
	fmt.Printf("  v1 decoded groups:          %v\n", v1Decoded.GroupIds)
	fmt.Printf("  v1 decoded SubscriptionTier: %-22s ← zero (v2 field skipped)\n", v1Decoded.SubscriptionTier)
	fmt.Printf("  v1 decoded Audit:            %-22v ← nil  (v2 field skipped)\n", v1Decoded.Audit)
	fmt.Printf("  v1 decoded DisplayName:      %-22q ← empty (v2 field skipped)\n\n", v1Decoded.DisplayName)

	fmt.Println("  ✔  v1 reader successfully decoded v2 binary (unknown fields ignored)\n")

	// ── 7. Forward compat: v2 reader reads v1 binary ─────────────────────────
	section("Step 6 — Forward Compatibility (v2 reader reads v1 binary)")

	fmt.Println("  Scenario: service B (v2) receives an old binary from service A (v1).")
	fmt.Println("  v2-only fields are absent → proto3 zero values apply.")
	fmt.Println()

	v1User := &userpb.User{
		Id:            "usr_legacy_001",
		Username:      "legacy_user",
		FirstName:     "Legacy",
		LastName:      "User",
		Role:          userpb.Role_MEMBER,
		Status:        userpb.AccountStatus_ACTIVE,
		Contact:       &userpb.ContactInfo{Email: "legacy@example.com"},
		CreatedAtUnix: now - 86400*365,
		UpdatedAtUnix: now,
		GroupIds:      []string{"grp_members"},
		// No v2 fields set
	}
	v1Bin, _ := v1User.Marshal()

	v2Read := &userpb.User{}
	must(v2Read.Unmarshal(v1Bin), "unmarshal v1 binary with v2 reader")

	fmt.Printf("  v2 decoded id:              %s\n", v2Read.Id)
	fmt.Printf("  v2 decoded username:        %s\n", v2Read.Username)
	fmt.Printf("  v2 decoded SubscriptionTier: %-22s ← zero (absent in v1 msg)\n", v2Read.SubscriptionTier)
	fmt.Printf("  v2 decoded Audit:            %-22v ← nil  (absent in v1 msg)\n", v2Read.Audit)
	fmt.Printf("  v2 decoded DisplayName:      %-22q ← empty (absent in v1 msg)\n\n", v2Read.DisplayName)

	fmt.Println("  ✔  v2 reader handled v1 binary gracefully (missing fields = zero values)\n")

	// ── 8. Wire inspection ────────────────────────────────────────────────────
	section("Step 7 — Wire Format Inspection")

	fmt.Println("  Decoding raw wire tags from the binary:")
	fmt.Println("  (field_number, wire_type, byte_length/value)")
	fmt.Println()
	inspectWire(binary)

	// ── 9. Write output files ─────────────────────────────────────────────────
	section("Step 8 — Writing Output Files")

	must(os.WriteFile("user_v2.bin", binary, 0644), "write bin")
	must(os.WriteFile("user_v2.json", jsonPretty, 0644), "write json")
	fmt.Println("  ✔  Written: user_v2.bin  — protobuf binary")
	fmt.Println("  ✔  Written: user_v2.json — JSON")
	fmt.Printf("\n  Binary: %d bytes   JSON: %d bytes   JSON is %.1f× larger\n",
		len(binary), len(jsonBytes), ratio)

	fmt.Println(banner("Done"))
}

// unmarshalV1Only decodes only field numbers 1–13, simulating a v1 reader
// that predates the v2 additions (fields 14–17).
func unmarshalV1Only(u *userpb.User, b []byte) error {
	return u.UnmarshalMaxField(b, 13)
}

// inspectWire walks the wire and prints each tag.
func inspectWire(b []byte) {
	fieldNames := map[protowire.Number]string{
		1: "id", 2: "username", 3: "first_name", 4: "last_name",
		5: "role", 6: "status", 7: "contact", 8: "address",
		9: "preferences", 10: "group_ids", 11: "metadata",
		12: "created_at_unix", 13: "updated_at_unix",
		14: "subscription_tier", 15: "audit",
		16: "display_name", 17: "badge_ids",
	}
	wireTypeNames := map[protowire.Type]string{
		protowire.VarintType: "varint",
		protowire.Fixed64Type: "fixed64",
		protowire.BytesType:  "len",
		protowire.Fixed32Type: "fixed32",
	}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		name := fieldNames[num]
		if name == "" {
			name = "unknown"
		}
		typName := wireTypeNames[typ]

		switch typ {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				break
			}
			b = b[n:]
			fmt.Printf("  field %-2d %-18s  %-7s  value=%d\n", num, "("+name+")", typName, v)
		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(b)
			if n < 0 {
				break
			}
			b = b[n:]
			preview := string(val)
			if len(preview) > 32 {
				preview = preview[:32] + "…"
			}
			// Only print printable strings
			isPrint := true
			for _, c := range val {
				if c < 0x20 && c != 0x0a {
					isPrint = false
					break
				}
			}
			if isPrint {
				fmt.Printf("  field %-2d %-18s  %-7s  len=%-4d %q\n", num, "("+name+")", typName, len(val), preview)
			} else {
				fmt.Printf("  field %-2d %-18s  %-7s  len=%-4d [embedded message]\n", num, "("+name+")", typName, len(val))
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				break
			}
			b = b[n:]
		}
	}
	fmt.Println()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func banner(s string) string {
	line := strings.Repeat("═", len(s)+4)
	return fmt.Sprintf("\n╔%s╗\n║  %s  ║\n╚%s╝\n", line, s, line)
}

func section(s string) {
	pad := 55 - len(s)
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("\n── %s %s\n\n", s, strings.Repeat("─", pad))
}

func must(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", msg, err)
		os.Exit(1)
	}
}
