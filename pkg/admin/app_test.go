package admin

import (
	"reflect"
	"testing"
)

func TestNewDefaultsName(t *testing.T) {
	app := New("")
	if app.Name() != "GoMyAdmin" {
		t.Fatalf("name = %q", app.Name())
	}
}

func TestNewPreservesName(t *testing.T) {
	app := New("My App")
	if app.Name() != "My App" {
		t.Fatalf("name = %q", app.Name())
	}
}

func TestResourceRegistersAndReturns(t *testing.T) {
	app := New("test")
	type User struct{}
	r := app.Resource(User{})
	if r == nil {
		t.Fatal("resource should not be nil")
	}
	if r.Name != "user" {
		t.Fatalf("name = %q", r.Name)
	}
}

func TestResourceIdempotent(t *testing.T) {
	app := New("test")
	type Customer struct{}
	r1 := app.Resource(Customer{})
	r2 := app.Resource(Customer{})
	if r1 != r2 {
		t.Fatal("same resource type should return same pointer")
	}
}

func TestResourceDefaultsTable(t *testing.T) {
	app := New("test")
	type InvoiceItem struct{}
	r := app.Resource(InvoiceItem{})
	if r.Table != "invoice_items" {
		t.Fatalf("table = %q", r.Table)
	}
}

func TestResourceDefaultSort(t *testing.T) {
	app := New("test")
	type Order struct{}
	r := app.Resource(Order{})
	if r.DefaultSort != "-created_at" {
		t.Fatalf("default_sort = %q", r.DefaultSort)
	}
}

func TestResourcesReturnsSortedByName(t *testing.T) {
	app := New("test")
	type Zebra struct{}
	type Alpha struct{}
	type Monkey struct{}
	app.Resource(Zebra{})
	app.Resource(Alpha{})
	app.Resource(Monkey{})
	resources := app.Resources()
	names := make([]string, len(resources))
	for i, r := range resources {
		names[i] = r.Name
	}
	expected := []string{"alpha", "monkey", "zebra"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("resources order = %v", names)
	}
}

func TestGetResource(t *testing.T) {
	app := New("test")
	type Lead struct{}
	app.Resource(Lead{})
	r, ok := app.GetResource("lead")
	if !ok || r == nil {
		t.Fatal("expected to find lead resource")
	}
}

func TestGetResourceMissing(t *testing.T) {
	app := New("test")
	_, ok := app.GetResource("missing")
	if ok {
		t.Fatal("expected resource not found")
	}
}

func TestRegisterPrebuilt(t *testing.T) {
	app := New("test")
	resource := &Resource{Name: "custom", LabelText: "Custom"}
	app.Register(resource)
	r, ok := app.GetResource("custom")
	if !ok || r.LabelText != "Custom" {
		t.Fatal("expected custom resource")
	}
}

func TestRegisterNilIgnored(t *testing.T) {
	app := New("test")
	app.Register(nil)
	if len(app.Resources()) != 0 {
		t.Fatal("nil register should be ignored")
	}
}

// --- naming helpers ---

func TestTitleize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user", "User"},
		{"invoiceItem", "Invoice Item"},
		{"APIKey", "API Key"},
		{"", ""},
		{"sms_message", "Sms Message"},
	}
	for _, c := range cases {
		if got := titleize(c.in); got != c.want {
			t.Errorf("titleize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPluralize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"User", "Users"},
		{"Category", "Categories"},
		{"Tax", "Taxes"},
		{"Address", "Addresses"},
		{"Invoice", "Invoices"},
	}
	for _, c := range cases {
		if got := pluralize(c.in); got != c.want {
			t.Errorf("pluralize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSnakePlural(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user", "users"},
		{"invoiceItem", "invoice_items"},
		{"APIKey", "api_keys"},
	}
	for _, c := range cases {
		if got := snakePlural(c.in); got != c.want {
			t.Errorf("snakePlural(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- resource builder ---

func TestResourceBuilderChain(t *testing.T) {
	app := New("test")
	type Product struct{}
	r := app.Resource(Product{}).
		Label("Products").
		PluralLabel("All Products").
		Icon("box").
		Description("Manage products").
		TableName("products").
		Sort("-name").
		PageSize(50).
		Group("catalog").
		Audit().
		TenantScoped("org_id")

	if r.LabelText != "Products" {
		t.Errorf("label = %q", r.LabelText)
	}
	if r.PluralText != "All Products" {
		t.Errorf("plural = %q", r.PluralText)
	}
	if r.IconName != "box" {
		t.Errorf("icon = %q", r.IconName)
	}
	if r.Table != "products" {
		t.Errorf("table = %q", r.Table)
	}
	if r.DefaultSort != "-name" {
		t.Errorf("sort = %q", r.DefaultSort)
	}
	if r.DefaultPageSize != 50 {
		t.Errorf("page size = %d", r.DefaultPageSize)
	}
	if r.NavigationGroup != "catalog" {
		t.Errorf("group = %q", r.NavigationGroup)
	}
	if !r.AuditEnabled {
		t.Error("audit should be enabled")
	}
	if r.TenantColumn != "org_id" {
		t.Errorf("tenant column = %q", r.TenantColumn)
	}
}

func TestResourcePageSizeIgnoresZero(t *testing.T) {
	app := New("test")
	type X struct{}
	r := app.Resource(X{}).PageSize(0)
	if r.DefaultPageSize != 25 {
		t.Fatalf("page size = %d", r.DefaultPageSize)
	}
}

// --- field builder ---

func TestFieldTypes(t *testing.T) {
	app := New("test")
	type Form struct{}
	r := app.Resource(Form{})

	r.Field("Email").Email()
	r.Field("Age").Integer()
	r.Field("Price").Money("USD")
	r.Field("Status").Enum("active", "inactive").Badge()
	r.Field("Body").Markdown()
	r.Field("Photo").ImageUpload()
	r.Field("Doc").FileUpload()
	r.Field("Meta").JSONB()

	byName := func(name string) *Field {
		f, ok := r.FieldByName(name)
		if !ok {
			t.Fatalf("field %q not found", name)
		}
		return f
	}

	if byName("Email").Type != FieldEmail {
		t.Error("email type")
	}
	if byName("Age").Type != FieldInteger {
		t.Error("integer type")
	}
	if byName("Price").Type != FieldMoney || byName("Price").Currency != "USD" {
		t.Error("money type or currency")
	}
	if byName("Status").Type != FieldStatus {
		t.Error("status (badge enum) type")
	}
	if len(byName("Status").EnumValues) != 2 {
		t.Error("enum values count")
	}
	if byName("Body").Type != FieldMarkdown {
		t.Error("markdown type")
	}
	if byName("Photo").Type != FieldImage {
		t.Error("image type")
	}
	if byName("Doc").Type != FieldFile {
		t.Error("file type")
	}
	if byName("Meta").Type != FieldJSONB {
		t.Error("jsonb type")
	}
}

func TestFieldFlags(t *testing.T) {
	app := New("test")
	type Item struct{}
	r := app.Resource(Item{})
	r.Field("Name").
		Required().
		Unique().
		Nullable().
		Searchable().
		Sortable().
		Filterable().
		Readonly().
		Hidden().
		HiddenInList().
		HiddenInForm().
		HiddenInDetail()

	f, _ := r.FieldByName("Name")
	if !f.RequiredValue {
		t.Error("required")
	}
	if !f.UniqueValue {
		t.Error("unique")
	}
	if !f.NullableValue {
		t.Error("nullable")
	}
	if !f.SearchableValue {
		t.Error("searchable")
	}
	if !f.SortableValue {
		t.Error("sortable")
	}
	if !f.FilterableValue {
		t.Error("filterable")
	}
	if !f.ReadonlyValue {
		t.Error("readonly")
	}
	if !f.HiddenValue {
		t.Error("hidden")
	}
	if !f.HiddenInListValue {
		t.Error("hidden_in_list")
	}
	if !f.HiddenInFormValue {
		t.Error("hidden_in_form")
	}
	if !f.HiddenInDetailValue {
		t.Error("hidden_in_detail")
	}
}

func TestFieldPrimarySetsPrimaryKey(t *testing.T) {
	app := New("test")
	type Order struct{}
	r := app.Resource(Order{})
	r.Field("UUID").UUID().Primary()
	if r.PrimaryKey != "UUID" {
		t.Fatalf("primary key = %q", r.PrimaryKey)
	}
	f, _ := r.FieldByName("UUID")
	if !f.PrimaryValue {
		t.Error("primary flag not set")
	}
}

func TestFieldRelations(t *testing.T) {
	app := New("test")
	type Company struct{}
	type Employee struct{}
	r := app.Resource(Employee{})
	r.Field("CompanyID").BelongsTo(Company{}).ForeignKey("company_id").Display("name")

	f, _ := r.FieldByName("CompanyID")
	if f.Type != FieldRelation {
		t.Error("relation type")
	}
	if f.Relation == nil || f.Relation.Kind != "belongs_to" {
		t.Error("belongs_to kind")
	}
	if f.Relation.ForeignKey != "company_id" {
		t.Errorf("foreign key = %q", f.Relation.ForeignKey)
	}
	if f.Relation.DisplayField != "name" {
		t.Errorf("display field = %q", f.Relation.DisplayField)
	}
}

func TestFieldBadgeColors(t *testing.T) {
	app := New("test")
	type Ticket struct{}
	r := app.Resource(Ticket{})
	r.Field("Priority").Enum("low", "high", "critical").
		BadgeColor("low", "green").
		BadgeColor("high", "yellow").
		BadgeColor("critical", "red")

	f, _ := r.FieldByName("Priority")
	if f.BadgeColors["critical"] != "red" {
		t.Errorf("badge color = %q", f.BadgeColors["critical"])
	}
}

func TestFieldDefaultValueAndHelp(t *testing.T) {
	app := New("test")
	type Config struct{}
	r := app.Resource(Config{})
	r.Field("Timeout").Integer().DefaultValue(30).Help("in seconds").Placeholder("e.g. 30")

	f, _ := r.FieldByName("Timeout")
	if f.DefaultValueValue != 30 {
		t.Errorf("default = %v", f.DefaultValueValue)
	}
	if f.HelpTextValue != "in seconds" {
		t.Errorf("help = %q", f.HelpTextValue)
	}
	if f.PlaceholderValue != "e.g. 30" {
		t.Errorf("placeholder = %q", f.PlaceholderValue)
	}
}

func TestFieldValidation(t *testing.T) {
	app := New("test")
	type Profile struct{}
	r := app.Resource(Profile{})
	r.Field("Bio").Text().Validation("max:500", "min:10")

	f, _ := r.FieldByName("Bio")
	if len(f.ValidationRules) != 2 {
		t.Fatalf("validation rules = %v", f.ValidationRules)
	}
}

func TestFieldRendererFormatterParser(t *testing.T) {
	app := New("test")
	type Row struct{}
	r := app.Resource(Row{})
	r.Field("Amount").Decimal().Renderer("currency").Formatter("usd").Parser("numeric")

	f, _ := r.FieldByName("Amount")
	if f.RendererName != "currency" || f.FormatterName != "usd" || f.ParserName != "numeric" {
		t.Errorf("renderer=%q formatter=%q parser=%q", f.RendererName, f.FormatterName, f.ParserName)
	}
}

// --- actions ---

func TestActionBuilder(t *testing.T) {
	app := New("test")
	type Invoice struct{}
	r := app.Resource(Invoice{})
	r.Action("Refund", nil).
		Describe("Process a full refund").
		WithIcon("rotate-ccw").
		Danger().
		RequireConfirmation().
		RequireReason().
		Can("invoices.refund")

	a, ok := r.ActionByName("refund")
	if !ok {
		t.Fatal("action not found")
	}
	if a.Description != "Process a full refund" {
		t.Errorf("description = %q", a.Description)
	}
	if !a.Dangerous {
		t.Error("dangerous")
	}
	if !a.RequiresConfirm {
		t.Error("requires confirmation")
	}
	if !a.RequiresReason {
		t.Error("requires reason")
	}
	if a.Permission != "invoices.refund" {
		t.Errorf("permission = %q", a.Permission)
	}
}

func TestActionByNameLabel(t *testing.T) {
	app := New("test")
	type Order struct{}
	r := app.Resource(Order{})
	r.Action("Cancel Order", nil)

	_, ok := r.ActionByName("Cancel Order")
	if !ok {
		t.Fatal("should find by label")
	}
	_, ok = r.ActionByName("cancel-order")
	if !ok {
		t.Fatal("should find by slug name")
	}
}

// --- context ---

func TestActorPermissionExact(t *testing.T) {
	actor := Actor{Permissions: []string{"users.delete"}}
	if !actor.Can("users.delete") {
		t.Error("exact permission should match")
	}
	if actor.Can("orders.delete") {
		t.Error("different resource should not match")
	}
}

func TestActorPermissionWildcard(t *testing.T) {
	actor := Actor{Permissions: []string{"*"}}
	if !actor.Can("anything.ever") {
		t.Error("* should match everything")
	}
}

func TestActorPermissionResourceWildcard(t *testing.T) {
	actor := Actor{Permissions: []string{"users.*"}}
	if !actor.Can("users.view") || !actor.Can("users.delete") {
		t.Error("users.* should match all users operations")
	}
	if actor.Can("orders.view") {
		t.Error("users.* should not match orders")
	}
}

func TestActorPermissionOperationWildcard(t *testing.T) {
	actor := Actor{Permissions: []string{"*.view"}}
	if !actor.Can("users.view") || !actor.Can("orders.view") {
		t.Error("*.view should match any resource view")
	}
	if actor.Can("users.delete") {
		t.Error("*.view should not match delete")
	}
}

func TestActorHasRole(t *testing.T) {
	actor := Actor{Roles: []string{"admin", "support"}}
	if !actor.HasRole("admin") {
		t.Error("should have admin role")
	}
	if actor.HasRole("viewer") {
		t.Error("should not have viewer role")
	}
}
