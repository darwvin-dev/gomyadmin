package server

import (
	"unicode"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

// Record is a generic database row or JSON payload used by AdminStore adapters.
type Record = map[string]any

// record is kept as an internal readability alias.
type record = Record

// ResourceMeta describes one admin resource as exposed to clients and stores.
type ResourceMeta = resourceMeta

// FieldMeta describes one resource field as exposed to clients and stores.
type FieldMeta = fieldMeta

// ActionMeta describes one resource action as exposed to clients and stores.
type ActionMeta = actionMeta

// AuditEvent describes one admin audit entry.
type AuditEvent = auditEvent

type resourceMeta struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Icon        string       `json:"icon,omitempty"`
	Description string       `json:"description,omitempty"`
	Table       string       `json:"table"`
	PrimaryKey  string       `json:"primary_key"`
	TenantKey   string       `json:"tenant_key,omitempty"`
	DefaultSort string       `json:"default_sort"`
	Fields      []fieldMeta  `json:"fields"`
	Actions     []actionMeta `json:"actions"`
}

type fieldMeta struct {
	SQLName    string          `json:"name"`
	fieldName  string          // original CamelCase name, internal only
	Label      string          `json:"label"`
	Type       admin.FieldType `json:"type"`
	Searchable bool            `json:"searchable"`
	Sortable   bool            `json:"sortable"`
	Filterable bool            `json:"filterable"`
	Readonly   bool            `json:"readonly"`
	Hidden     bool            `json:"hidden"`
	Primary    bool            `json:"primary,omitempty"`
	TenantKey  bool            `json:"tenant_key,omitempty"`
	EnumValues []string        `json:"enum_values,omitempty"`
}

type actionMeta struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Icon  string `json:"icon,omitempty"`
}

func appToMeta(app *admin.App) map[string]resourceMeta {
	all := app.Resources()
	result := make(map[string]resourceMeta, len(all))
	for _, r := range all {
		meta := resourceToMeta(r)
		result[r.Table] = meta
	}
	return result
}

func resourceToMeta(r *admin.Resource) resourceMeta {
	fields := make([]fieldMeta, 0, len(r.Fields))
	for _, f := range r.Fields {
		fields = append(fields, fieldToMeta(f))
	}
	actions := make([]actionMeta, 0, len(r.Actions))
	for _, a := range r.Actions {
		actions = append(actions, actionMeta{Name: a.Name, Label: a.Label, Icon: a.Icon})
	}
	return resourceMeta{
		Name:        r.Name,
		Label:       r.LabelText,
		Icon:        r.IconName,
		Description: r.DescriptionText,
		Table:       r.Table,
		PrimaryKey:  camelToSnake(r.PrimaryKey),
		TenantKey:   r.TenantColumn,
		DefaultSort: r.DefaultSort,
		Fields:      fields,
		Actions:     actions,
	}
}

func fieldToMeta(f *admin.Field) fieldMeta {
	return fieldMeta{
		SQLName:    camelToSnake(f.Name),
		fieldName:  f.Name,
		Label:      f.Label,
		Type:       f.Type,
		Searchable: f.SearchableValue,
		Sortable:   f.SortableValue,
		Filterable: f.FilterableValue,
		Readonly:   f.ReadonlyValue,
		Hidden:     f.HiddenValue,
		Primary:    f.PrimaryValue,
		TenantKey:  f.TenantKeyValue,
		EnumValues: f.EnumValues,
	}
}

// camelToSnake converts a CamelCase identifier to snake_case.
// "ID"→"id", "CreatedAt"→"created_at", "TenantID"→"tenant_id", "APIKey"→"api_key".
// Strings that are already snake_case pass through unchanged.
func camelToSnake(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	out := make([]rune, 0, len(runes)+4)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower) {
					out = append(out, '_')
				}
			}
			out = append(out, unicode.ToLower(r))
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
