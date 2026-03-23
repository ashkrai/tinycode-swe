// Package userpb implements the User message defined in proto/user_v2.proto
// using the protobuf wire format directly via google.golang.org/protobuf/encoding/protowire.
//
// In a real project you run:
//   protoc --go_out=. --go_opt=paths=source_relative proto/user_v2.proto
// and this code is generated automatically. This file does exactly what
// protoc-gen-go produces, without requiring protoc to be installed.
package userpb

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// ── Enum types ────────────────────────────────────────────────────────────────

type Role int32

const (
	Role_UNSPECIFIED Role = 0
	Role_GUEST       Role = 1
	Role_MEMBER      Role = 2
	Role_ADMIN       Role = 3
	Role_SUPER_ADMIN Role = 4
)

func (r Role) String() string {
	switch r {
	case Role_GUEST:
		return "ROLE_GUEST"
	case Role_MEMBER:
		return "ROLE_MEMBER"
	case Role_ADMIN:
		return "ROLE_ADMIN"
	case Role_SUPER_ADMIN:
		return "ROLE_SUPER_ADMIN"
	default:
		return "ROLE_UNSPECIFIED"
	}
}
func (r Role) MarshalJSON() ([]byte, error) { return json.Marshal(r.String()) }

type AccountStatus int32

const (
	AccountStatus_UNSPECIFIED AccountStatus = 0
	AccountStatus_PENDING     AccountStatus = 1
	AccountStatus_ACTIVE      AccountStatus = 2
	AccountStatus_SUSPENDED   AccountStatus = 3
	AccountStatus_DELETED     AccountStatus = 4
)

func (s AccountStatus) String() string {
	switch s {
	case AccountStatus_PENDING:
		return "ACCOUNT_STATUS_PENDING"
	case AccountStatus_ACTIVE:
		return "ACCOUNT_STATUS_ACTIVE"
	case AccountStatus_SUSPENDED:
		return "ACCOUNT_STATUS_SUSPENDED"
	case AccountStatus_DELETED:
		return "ACCOUNT_STATUS_DELETED"
	default:
		return "ACCOUNT_STATUS_UNSPECIFIED"
	}
}
func (s AccountStatus) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

type SubscriptionTier int32

const (
	SubscriptionTier_UNSPECIFIED SubscriptionTier = 0
	SubscriptionTier_FREE        SubscriptionTier = 1
	SubscriptionTier_PRO         SubscriptionTier = 2
	SubscriptionTier_ENTERPRISE  SubscriptionTier = 3
)

func (t SubscriptionTier) String() string {
	switch t {
	case SubscriptionTier_FREE:
		return "SUBSCRIPTION_TIER_FREE"
	case SubscriptionTier_PRO:
		return "SUBSCRIPTION_TIER_PRO"
	case SubscriptionTier_ENTERPRISE:
		return "SUBSCRIPTION_TIER_ENTERPRISE"
	default:
		return "SUBSCRIPTION_TIER_UNSPECIFIED"
	}
}
func (t SubscriptionTier) MarshalJSON() ([]byte, error) { return json.Marshal(t.String()) }

// ── Nested message structs ────────────────────────────────────────────────────

// Address — proto field 8 in User.
// Field numbers: street=1, city=2, state=3, country=4, zip=5, formatted=6(v2)
type Address struct {
	Street    string `json:"street,omitempty"`
	City      string `json:"city,omitempty"`
	State     string `json:"state,omitempty"`
	Country   string `json:"country,omitempty"`
	Zip       string `json:"zip,omitempty"`
	Formatted string `json:"formatted,omitempty"` // v2
}

// ContactInfo — proto field 7 in User.
// Field numbers: email=1, phone=2, social_links=3, website=4(v2)
type ContactInfo struct {
	Email       string   `json:"email,omitempty"`
	Phone       string   `json:"phone,omitempty"`
	SocialLinks []string `json:"social_links,omitempty"`
	Website     string   `json:"website,omitempty"` // v2
}

// Preferences — proto field 9 in User.
// Field numbers: language=1, timezone=2, email_newsletter=3, dark_mode=4, theme_color=5(v2)
type Preferences struct {
	Language        string `json:"language,omitempty"`
	Timezone        string `json:"timezone,omitempty"`
	EmailNewsletter bool   `json:"email_newsletter,omitempty"`
	DarkMode        bool   `json:"dark_mode,omitempty"`
	ThemeColor      string `json:"theme_color,omitempty"` // v2
}

// AuditInfo — proto field 15 in User (v2 only).
// Field numbers: last_login_ip=1, last_login_unix=2, login_count=3, mfa_enabled=4
type AuditInfo struct {
	LastLoginIp   string `json:"last_login_ip,omitempty"`
	LastLoginUnix int64  `json:"last_login_unix,omitempty"`
	LoginCount    int32  `json:"login_count,omitempty"`
	MfaEnabled    bool   `json:"mfa_enabled,omitempty"`
}

// User — root message, v2.
// v1 field numbers 1–13 are frozen. v2 additions use 14–17.
type User struct {
	// v1 fields
	Id            string            `json:"id,omitempty"`
	Username      string            `json:"username,omitempty"`
	FirstName     string            `json:"first_name,omitempty"`
	LastName      string            `json:"last_name,omitempty"`
	Role          Role              `json:"role,omitempty"`
	Status        AccountStatus     `json:"status,omitempty"`
	Contact       *ContactInfo      `json:"contact,omitempty"`
	Address       *Address          `json:"address,omitempty"`
	Preferences   *Preferences      `json:"preferences,omitempty"`
	GroupIds      []string          `json:"group_ids,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAtUnix int64             `json:"created_at_unix,omitempty"`
	UpdatedAtUnix int64             `json:"updated_at_unix,omitempty"`
	// v2 fields (field numbers 14–17)
	SubscriptionTier SubscriptionTier `json:"subscription_tier,omitempty"`
	Audit            *AuditInfo       `json:"audit,omitempty"`
	DisplayName      string           `json:"display_name,omitempty"`
	BadgeIds         []string         `json:"badge_ids,omitempty"`
}

// ── Serialisation helpers ─────────────────────────────────────────────────────
//
// Wire types used:
//   VarintType (0) — int32, int64, bool, enum
//   BytesType  (2) — string, bytes, embedded message, repeated string

func appendStr(b []byte, field protowire.Number, s string) []byte {
	if s == "" {
		return b
	}
	b = protowire.AppendTag(b, field, protowire.BytesType)
	return protowire.AppendString(b, s)
}

func appendInt(b []byte, field protowire.Number, v uint64) []byte {
	if v == 0 {
		return b
	}
	b = protowire.AppendTag(b, field, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

func appendBool(b []byte, field protowire.Number, v bool) []byte {
	if !v {
		return b
	}
	b = protowire.AppendTag(b, field, protowire.VarintType)
	return protowire.AppendVarint(b, 1)
}

func appendMsg(b []byte, field protowire.Number, msg []byte) []byte {
	if len(msg) == 0 {
		return b
	}
	b = protowire.AppendTag(b, field, protowire.BytesType)
	return protowire.AppendBytes(b, msg)
}

// ── Per-message marshalers ────────────────────────────────────────────────────

func marshalAddress(a *Address) []byte {
	if a == nil {
		return nil
	}
	var b []byte
	b = appendStr(b, 1, a.Street)
	b = appendStr(b, 2, a.City)
	b = appendStr(b, 3, a.State)
	b = appendStr(b, 4, a.Country)
	b = appendStr(b, 5, a.Zip)
	b = appendStr(b, 6, a.Formatted)
	return b
}

func marshalContact(c *ContactInfo) []byte {
	if c == nil {
		return nil
	}
	var b []byte
	b = appendStr(b, 1, c.Email)
	b = appendStr(b, 2, c.Phone)
	for _, link := range c.SocialLinks {
		b = appendStr(b, 3, link)
	}
	b = appendStr(b, 4, c.Website)
	return b
}

func marshalPreferences(p *Preferences) []byte {
	if p == nil {
		return nil
	}
	var b []byte
	b = appendStr(b, 1, p.Language)
	b = appendStr(b, 2, p.Timezone)
	b = appendBool(b, 3, p.EmailNewsletter)
	b = appendBool(b, 4, p.DarkMode)
	b = appendStr(b, 5, p.ThemeColor)
	return b
}

func marshalAudit(a *AuditInfo) []byte {
	if a == nil {
		return nil
	}
	var b []byte
	b = appendStr(b, 1, a.LastLoginIp)
	b = appendInt(b, 2, uint64(a.LastLoginUnix))
	b = appendInt(b, 3, uint64(a.LoginCount))
	b = appendBool(b, 4, a.MfaEnabled)
	return b
}

// Marshal serialises User to the protobuf binary wire format.
// Field numbers exactly match proto/user_v2.proto.
func (u *User) Marshal() ([]byte, error) {
	var b []byte

	// v1 fields (1–13)
	b = appendStr(b, 1, u.Id)
	b = appendStr(b, 2, u.Username)
	b = appendStr(b, 3, u.FirstName)
	b = appendStr(b, 4, u.LastName)
	b = appendInt(b, 5, uint64(u.Role))
	b = appendInt(b, 6, uint64(u.Status))
	b = appendMsg(b, 7, marshalContact(u.Contact))
	b = appendMsg(b, 8, marshalAddress(u.Address))
	b = appendMsg(b, 9, marshalPreferences(u.Preferences))
	for _, g := range u.GroupIds {
		b = appendStr(b, 10, g)
	}
	// map<string,string> — each entry encoded as embedded message {key=1, value=2}
	for k, v := range u.Metadata {
		var entry []byte
		entry = appendStr(entry, 1, k)
		entry = appendStr(entry, 2, v)
		b = appendMsg(b, 11, entry)
	}
	b = appendInt(b, 12, uint64(u.CreatedAtUnix))
	b = appendInt(b, 13, uint64(u.UpdatedAtUnix))

	// v2 fields (14–17)
	b = appendInt(b, 14, uint64(u.SubscriptionTier))
	b = appendMsg(b, 15, marshalAudit(u.Audit))
	b = appendStr(b, 16, u.DisplayName)
	for _, badge := range u.BadgeIds {
		b = appendStr(b, 17, badge)
	}

	return b, nil
}

// Unmarshal deserialises a User from binary wire format.
// Unknown field numbers are silently skipped — this is the backward-compat guarantee.
func (u *User) Unmarshal(b []byte) error {
	return u.UnmarshalMaxField(b, 9999)
}

// UnmarshalMaxField deserialises only fields with number <= maxField.
// Used to simulate a v1 reader (maxField=13) ignoring v2 fields (14+).
func (u *User) UnmarshalMaxField(b []byte, maxField protowire.Number) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return fmt.Errorf("invalid tag")
		}
		b = b[n:]

		// If this field is beyond maxField, skip it (simulate old reader)
		if num > maxField {
			n = protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return fmt.Errorf("skip field %d: invalid value", num)
			}
			b = b[n:]
			continue
		}

		switch typ {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return fmt.Errorf("invalid varint field %d", num)
			}
			b = b[n:]
			switch num {
			case 5:
				u.Role = Role(v)
			case 6:
				u.Status = AccountStatus(v)
			case 12:
				u.CreatedAtUnix = int64(v)
			case 13:
				u.UpdatedAtUnix = int64(v)
			case 14:
				u.SubscriptionTier = SubscriptionTier(v)
			}

		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return fmt.Errorf("invalid bytes field %d", num)
			}
			b = b[n:]
			switch num {
			case 1:
				u.Id = string(val)
			case 2:
				u.Username = string(val)
			case 3:
				u.FirstName = string(val)
			case 4:
				u.LastName = string(val)
			case 7:
				if u.Contact == nil {
					u.Contact = &ContactInfo{}
				}
				if err := unmarshalContact(u.Contact, val); err != nil {
					return err
				}
			case 8:
				if u.Address == nil {
					u.Address = &Address{}
				}
				if err := unmarshalAddress(u.Address, val); err != nil {
					return err
				}
			case 9:
				if u.Preferences == nil {
					u.Preferences = &Preferences{}
				}
				if err := unmarshalPreferences(u.Preferences, val); err != nil {
					return err
				}
			case 10:
				u.GroupIds = append(u.GroupIds, string(val))
			case 11:
				k, v, err := unmarshalMapEntry(val)
				if err != nil {
					return err
				}
				if u.Metadata == nil {
					u.Metadata = make(map[string]string)
				}
				u.Metadata[k] = v
			case 15:
				if u.Audit == nil {
					u.Audit = &AuditInfo{}
				}
				if err := unmarshalAudit(u.Audit, val); err != nil {
					return err
				}
			case 16:
				u.DisplayName = string(val)
			case 17:
				u.BadgeIds = append(u.BadgeIds, string(val))
			}

		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return fmt.Errorf("unknown wire type %d for field %d", typ, num)
			}
			b = b[n:]
		}
	}
	return nil
}

// ── Per-message unmarshalers ──────────────────────────────────────────────────

func unmarshalAddress(a *Address, b []byte) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return fmt.Errorf("address: bad tag")
		}
		b = b[n:]
		if typ != protowire.BytesType {
			if n = protowire.ConsumeFieldValue(num, typ, b); n < 0 {
				return fmt.Errorf("address: skip err")
			}
			b = b[n:]
			continue
		}
		val, n := protowire.ConsumeBytes(b)
		if n < 0 {
			return fmt.Errorf("address: bad bytes")
		}
		b = b[n:]
		switch num {
		case 1:
			a.Street = string(val)
		case 2:
			a.City = string(val)
		case 3:
			a.State = string(val)
		case 4:
			a.Country = string(val)
		case 5:
			a.Zip = string(val)
		case 6:
			a.Formatted = string(val)
		}
	}
	return nil
}

func unmarshalContact(c *ContactInfo, b []byte) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return fmt.Errorf("contact: bad tag")
		}
		b = b[n:]
		if typ != protowire.BytesType {
			if n = protowire.ConsumeFieldValue(num, typ, b); n < 0 {
				return fmt.Errorf("contact: skip err")
			}
			b = b[n:]
			continue
		}
		val, n := protowire.ConsumeBytes(b)
		if n < 0 {
			return fmt.Errorf("contact: bad bytes")
		}
		b = b[n:]
		switch num {
		case 1:
			c.Email = string(val)
		case 2:
			c.Phone = string(val)
		case 3:
			c.SocialLinks = append(c.SocialLinks, string(val))
		case 4:
			c.Website = string(val)
		}
	}
	return nil
}

func unmarshalPreferences(p *Preferences, b []byte) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return fmt.Errorf("prefs: bad tag")
		}
		b = b[n:]
		switch typ {
		case protowire.VarintType:
			val, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return fmt.Errorf("prefs: bad varint")
			}
			b = b[n:]
			switch num {
			case 3:
				p.EmailNewsletter = val != 0
			case 4:
				p.DarkMode = val != 0
			}
		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return fmt.Errorf("prefs: bad bytes")
			}
			b = b[n:]
			switch num {
			case 1:
				p.Language = string(val)
			case 2:
				p.Timezone = string(val)
			case 5:
				p.ThemeColor = string(val)
			}
		default:
			if n = protowire.ConsumeFieldValue(num, typ, b); n < 0 {
				return fmt.Errorf("prefs: skip err")
			}
			b = b[n:]
		}
	}
	return nil
}

func unmarshalAudit(a *AuditInfo, b []byte) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return fmt.Errorf("audit: bad tag")
		}
		b = b[n:]
		switch typ {
		case protowire.VarintType:
			val, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return fmt.Errorf("audit: bad varint")
			}
			b = b[n:]
			switch num {
			case 2:
				a.LastLoginUnix = int64(val)
			case 3:
				a.LoginCount = int32(val)
			case 4:
				a.MfaEnabled = val != 0
			}
		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return fmt.Errorf("audit: bad bytes")
			}
			b = b[n:]
			if num == 1 {
				a.LastLoginIp = string(val)
			}
		default:
			if n = protowire.ConsumeFieldValue(num, typ, b); n < 0 {
				return fmt.Errorf("audit: skip err")
			}
			b = b[n:]
		}
	}
	return nil
}

func unmarshalMapEntry(b []byte) (key, value string, err error) {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return "", "", fmt.Errorf("map entry: bad tag")
		}
		b = b[n:]
		if typ != protowire.BytesType {
			if n = protowire.ConsumeFieldValue(num, typ, b); n < 0 {
				return "", "", fmt.Errorf("map entry: skip err")
			}
			b = b[n:]
			continue
		}
		val, n := protowire.ConsumeBytes(b)
		if n < 0 {
			return "", "", fmt.Errorf("map entry: bad bytes")
		}
		b = b[n:]
		switch num {
		case 1:
			key = string(val)
		case 2:
			value = string(val)
		}
	}
	return key, value, nil
}
