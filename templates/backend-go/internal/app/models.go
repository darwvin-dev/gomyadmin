package app

import (
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

type Record map[string]any

type ResourceMeta struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Icon        string       `json:"icon"`
	Description string       `json:"description"`
	Table       string       `json:"table"`
	PrimaryKey  string       `json:"primary_key"`
	TenantKey   string       `json:"tenant_key,omitempty"`
	Fields      []FieldMeta  `json:"fields"`
	Actions     []ActionMeta `json:"actions"`
}

type FieldMeta struct {
	Name       string   `json:"name"`
	Label      string   `json:"label"`
	Type       string   `json:"type"`
	Searchable bool     `json:"searchable"`
	Sortable   bool     `json:"sortable"`
	Filterable bool     `json:"filterable"`
	Readonly   bool     `json:"readonly"`
	Hidden     bool     `json:"hidden"`
	EnumValues []string `json:"enum_values,omitempty"`
}

type ActionMeta struct {
	Name                 string `json:"name"`
	Label                string `json:"label"`
	Icon                 string `json:"icon"`
	Danger               bool   `json:"danger"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
	RequiresReason       bool   `json:"requires_reason"`
}

type AuditEvent struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`
	ActorEmail string         `json:"actor_email"`
	TenantID   string         `json:"tenant_id"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	OldValues  map[string]any `json:"old_values,omitempty"`
	NewValues  map[string]any `json:"new_values,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	IPAddress  string         `json:"ip_address"`
	UserAgent  string         `json:"user_agent"`
	RequestID  string         `json:"request_id"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Actor = admin.Actor
