package admin

type FieldType string

const (
	FieldString   FieldType = "string"
	FieldText     FieldType = "text"
	FieldEmail    FieldType = "email"
	FieldPassword FieldType = "password"
	FieldNumber   FieldType = "number"
	FieldInteger  FieldType = "integer"
	FieldFloat    FieldType = "float"
	FieldDecimal  FieldType = "decimal"
	FieldMoney    FieldType = "money"
	FieldBoolean  FieldType = "boolean"
	FieldEnum     FieldType = "enum"
	FieldStatus   FieldType = "status"
	FieldDate     FieldType = "date"
	FieldDateTime FieldType = "datetime"
	FieldTime     FieldType = "time"
	FieldUUID     FieldType = "uuid"
	FieldJSON     FieldType = "json"
	FieldJSONB    FieldType = "jsonb"
	FieldMarkdown FieldType = "markdown"
	FieldRichText FieldType = "rich_text"
	FieldImage    FieldType = "image"
	FieldFile     FieldType = "file"
	FieldRelation FieldType = "relation"
	FieldComputed FieldType = "computed"
)

type Field struct {
	Resource            *Resource         `json:"-"`
	Name                string            `json:"name"`
	Label               string            `json:"label"`
	Type                FieldType         `json:"type"`
	RequiredValue       bool              `json:"required"`
	UniqueValue         bool              `json:"unique"`
	NullableValue       bool              `json:"nullable"`
	SearchableValue     bool              `json:"searchable"`
	SortableValue       bool              `json:"sortable"`
	FilterableValue     bool              `json:"filterable"`
	ReadonlyValue       bool              `json:"readonly"`
	HiddenValue         bool              `json:"hidden"`
	HiddenInListValue   bool              `json:"hidden_in_list"`
	HiddenInFormValue   bool              `json:"hidden_in_form"`
	HiddenInDetailValue bool              `json:"hidden_in_detail"`
	PrimaryValue        bool              `json:"primary"`
	TenantKeyValue      bool              `json:"tenant_key"`
	DefaultValueValue   any               `json:"default_value,omitempty"`
	ValidationRules     []string          `json:"validation,omitempty"`
	HelpTextValue       string            `json:"help_text,omitempty"`
	PlaceholderValue    string            `json:"placeholder,omitempty"`
	BadgeColors         map[string]string `json:"badge_colors,omitempty"`
	EnumValues          []string          `json:"enum_values,omitempty"`
	Currency            string            `json:"currency,omitempty"`
	Relation            *Relation         `json:"relation,omitempty"`
	RendererName        string            `json:"renderer,omitempty"`
	FormatterName       string            `json:"formatter,omitempty"`
	ParserName          string            `json:"parser,omitempty"`
}

type Relation struct {
	Resource     string `json:"resource"`
	ForeignKey   string `json:"foreign_key,omitempty"`
	DisplayField string `json:"display_field,omitempty"`
	Kind         string `json:"kind"`
}

func (f *Field) Field(name string) *Field {
	return f.Resource.Field(name)
}

func (f *Field) Action(name string, handler any) *Action {
	return f.Resource.Action(name, handler)
}

func (f *Field) Policy(policy Policy) *Resource {
	return f.Resource.Policy(policy)
}

func (f *Field) Audit() *Resource {
	return f.Resource.Audit()
}

func (f *Field) TenantScoped(column string) *Resource {
	return f.Resource.TenantScoped(column)
}

func (f *Field) String() *Field      { f.Type = FieldString; return f }
func (f *Field) Text() *Field        { f.Type = FieldText; return f }
func (f *Field) Email() *Field       { f.Type = FieldEmail; return f }
func (f *Field) Password() *Field    { f.Type = FieldPassword; return f }
func (f *Field) Number() *Field      { f.Type = FieldNumber; return f }
func (f *Field) Integer() *Field     { f.Type = FieldInteger; return f }
func (f *Field) Float() *Field       { f.Type = FieldFloat; return f }
func (f *Field) Decimal() *Field     { f.Type = FieldDecimal; return f }
func (f *Field) Boolean() *Field     { f.Type = FieldBoolean; return f }
func (f *Field) Date() *Field        { f.Type = FieldDate; return f }
func (f *Field) DateTime() *Field    { f.Type = FieldDateTime; return f }
func (f *Field) Time() *Field        { f.Type = FieldTime; return f }
func (f *Field) UUID() *Field        { f.Type = FieldUUID; return f }
func (f *Field) JSON() *Field        { f.Type = FieldJSON; return f }
func (f *Field) JSONB() *Field       { f.Type = FieldJSONB; return f }
func (f *Field) Markdown() *Field    { f.Type = FieldMarkdown; return f }
func (f *Field) RichText() *Field    { f.Type = FieldRichText; return f }
func (f *Field) ImageUpload() *Field { f.Type = FieldImage; return f }
func (f *Field) FileUpload() *Field  { f.Type = FieldFile; return f }
func (f *Field) Computed() *Field    { f.Type = FieldComputed; return f }

func (f *Field) Money(currency string) *Field {
	f.Type = FieldMoney
	f.Currency = currency
	return f
}

func (f *Field) Enum(values ...string) *Field {
	f.Type = FieldEnum
	f.EnumValues = values
	return f
}

func (f *Field) Badge() *Field {
	if f.Type == FieldEnum {
		f.Type = FieldStatus
	}
	return f
}

func (f *Field) BadgeColor(value, color string) *Field {
	if f.BadgeColors == nil {
		f.BadgeColors = map[string]string{}
	}
	f.BadgeColors[value] = color
	return f
}

func (f *Field) BelongsTo(resource any) *Field {
	f.Type = FieldRelation
	f.Relation = &Relation{Resource: resourceName(resource), Kind: "belongs_to"}
	return f
}

func (f *Field) HasMany(resource any) *Field {
	f.Type = FieldRelation
	f.Relation = &Relation{Resource: resourceName(resource), Kind: "has_many"}
	return f
}

func (f *Field) RelationTo(resource any) *Field {
	return f.BelongsTo(resource)
}

func (f *Field) Display(field string) *Field {
	if f.Relation == nil {
		f.Relation = &Relation{}
	}
	f.Relation.DisplayField = field
	return f
}

func (f *Field) ForeignKey(field string) *Field {
	if f.Relation == nil {
		f.Relation = &Relation{}
	}
	f.Relation.ForeignKey = field
	return f
}

func (f *Field) Required() *Field        { f.RequiredValue = true; return f }
func (f *Field) Unique() *Field          { f.UniqueValue = true; return f }
func (f *Field) Nullable() *Field        { f.NullableValue = true; return f }
func (f *Field) Searchable() *Field      { f.SearchableValue = true; return f }
func (f *Field) Sortable() *Field        { f.SortableValue = true; return f }
func (f *Field) Filterable() *Field      { f.FilterableValue = true; return f }
func (f *Field) Readonly() *Field        { f.ReadonlyValue = true; return f }
func (f *Field) Hidden() *Field          { f.HiddenValue = true; return f }
func (f *Field) HiddenInList() *Field    { f.HiddenInListValue = true; return f }
func (f *Field) HiddenInForm() *Field    { f.HiddenInFormValue = true; return f }
func (f *Field) HiddenInDetail() *Field  { f.HiddenInDetailValue = true; return f }
func (f *Field) Primary() *Field         { f.PrimaryValue = true; f.Resource.PrimaryKey = f.Name; return f }
func (f *Field) TenantKey() *Field       { f.TenantKeyValue = true; return f }
func (f *Field) DateRangeFilter() *Field { f.FilterableValue = true; return f }

func (f *Field) DefaultValue(value any) *Field {
	f.DefaultValueValue = value
	return f
}

func (f *Field) Validation(rules ...string) *Field {
	f.ValidationRules = append(f.ValidationRules, rules...)
	return f
}

func (f *Field) Help(text string) *Field {
	f.HelpTextValue = text
	return f
}

func (f *Field) Placeholder(text string) *Field {
	f.PlaceholderValue = text
	return f
}

func (f *Field) Renderer(name string) *Field {
	f.RendererName = name
	return f
}

func (f *Field) Formatter(name string) *Field {
	f.FormatterName = name
	return f
}

func (f *Field) Parser(name string) *Field {
	f.ParserName = name
	return f
}
