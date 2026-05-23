package admin

import (
	"context"
	"time"
)

// Resource describes an admin-managed database-backed entity.
type Resource struct {
	Name            string    `json:"name"`
	LabelText       string    `json:"label"`
	PluralText      string    `json:"plural_label"`
	IconName        string    `json:"icon"`
	DescriptionText string    `json:"description"`
	Table           string    `json:"table"`
	PrimaryKey      string    `json:"primary_key"`
	DefaultSort     string    `json:"default_sort"`
	DefaultFilters  []Filter  `json:"default_filters,omitempty"`
	DefaultPageSize int       `json:"default_page_size"`
	NavigationGroup string    `json:"navigation_group,omitempty"`
	VisibleWhen     Rule      `json:"-"`
	Fields          []*Field  `json:"fields"`
	Actions         []*Action `json:"actions"`
	PolicyValue     Policy    `json:"-"`
	AuditEnabled    bool      `json:"audit_enabled"`
	TenantColumn    string    `json:"tenant_column,omitempty"`
}

type Filter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type Rule func(Context, Actor) bool

func (r *Resource) Label(label string) *Resource {
	r.LabelText = label
	return r
}

func (r *Resource) PluralLabel(label string) *Resource {
	r.PluralText = label
	return r
}

func (r *Resource) Icon(icon string) *Resource {
	r.IconName = icon
	return r
}

func (r *Resource) Description(description string) *Resource {
	r.DescriptionText = description
	return r
}

func (r *Resource) TableName(table string) *Resource {
	r.Table = table
	return r
}

func (r *Resource) Primary(primaryKey string) *Resource {
	r.PrimaryKey = primaryKey
	return r
}

func (r *Resource) Sort(sort string) *Resource {
	r.DefaultSort = sort
	return r
}

func (r *Resource) PageSize(size int) *Resource {
	if size > 0 {
		r.DefaultPageSize = size
	}
	return r
}

func (r *Resource) Group(group string) *Resource {
	r.NavigationGroup = group
	return r
}

func (r *Resource) Visible(rule Rule) *Resource {
	r.VisibleWhen = rule
	return r
}

func (r *Resource) Field(name string) *Field {
	field := &Field{
		Resource: r,
		Name:     name,
		Label:    name,
		Type:     FieldString,
	}
	r.Fields = append(r.Fields, field)
	return field
}

func (r *Resource) Action(name string, handler any) *Action {
	action := &Action{
		Resource: r,
		Name:     slug(name),
		Label:    name,
		Handler:  handler,
	}
	r.Actions = append(r.Actions, action)
	return action
}

func (r *Resource) Policy(policy Policy) *Resource {
	r.PolicyValue = policy
	return r
}

func (r *Resource) Audit() *Resource {
	r.AuditEnabled = true
	return r
}

func (r *Resource) TenantScoped(column string) *Resource {
	r.TenantColumn = column
	return r
}

func (r *Resource) FieldByName(name string) (*Field, bool) {
	for _, field := range r.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return nil, false
}

func (r *Resource) ActionByName(name string) (*Action, bool) {
	for _, action := range r.Actions {
		if action.Name == name || action.Label == name {
			return action, true
		}
	}
	return nil, false
}

// ActionRequest is passed to custom action handlers.
type ActionRequest[T any] struct {
	Context    Context
	Actor      Actor
	Resource   *Resource
	ResourceID string
	Input      T
	Selected   []string
	Reason     string
}

// ActionResult is a structured response from an action.
type ActionResult struct {
	Message  string         `json:"message"`
	Redirect string         `json:"redirect,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

// ActionHandler is the runtime-compatible action handler signature.
type ActionHandler func(context.Context, ActionRequest[map[string]any]) (ActionResult, error)

type Action struct {
	Resource        *Resource `json:"-"`
	Name            string    `json:"name"`
	Label           string    `json:"label"`
	Description     string    `json:"description,omitempty"`
	Icon            string    `json:"icon,omitempty"`
	Dangerous       bool      `json:"danger"`
	RequiresConfirm bool      `json:"requires_confirmation"`
	RequiresReason  bool      `json:"requires_reason"`
	InputSchema     any       `json:"input_schema,omitempty"`
	Permission      string    `json:"permission,omitempty"`
	Handler         any       `json:"-"`
	Timeout         string    `json:"timeout,omitempty"`
}

func (a *Action) Field(name string) *Field {
	return a.Resource.Field(name)
}

func (a *Action) Action(name string, handler any) *Action {
	return a.Resource.Action(name, handler)
}

func (a *Action) Policy(policy Policy) *Resource {
	return a.Resource.Policy(policy)
}

func (a *Action) Audit() *Resource {
	return a.Resource.Audit()
}

func (a *Action) TenantScoped(column string) *Resource {
	return a.Resource.TenantScoped(column)
}

func (a *Action) Describe(description string) *Action {
	a.Description = description
	return a
}

func (a *Action) WithIcon(icon string) *Action {
	a.Icon = icon
	return a
}

func (a *Action) Danger() *Action {
	a.Dangerous = true
	return a
}

func (a *Action) RequireConfirmation() *Action {
	a.RequiresConfirm = true
	return a
}

func (a *Action) RequireReason() *Action {
	a.RequiresReason = true
	return a
}

func (a *Action) Form(schema any) *Action {
	a.InputSchema = schema
	return a
}

func (a *Action) Can(permission string) *Action {
	a.Permission = permission
	return a
}

func (a *Action) WithTimeout(timeout time.Duration) *Action {
	a.Timeout = timeout.String()
	return a
}
