package saas

import "github.com/darwvin/gomyadmin/pkg/admin"

type Organization struct{}
type User struct{}
type Subscription struct{}
type Invoice struct{}
type APIKey struct{}

func Register(app *admin.App) {
	app.Resource(Organization{}).
		Label("Organizations").
		Icon("building-2").
		Field("Name").String().Required().Searchable().Sortable().
		Field("Plan").Enum("starter", "growth", "enterprise").Filterable().
		Field("Status").Enum("active", "past_due", "blocked").Filterable().Badge().
		Audit()

	app.Resource(User{}).Label("Users").Icon("users").Field("Email").Email().Searchable().Sortable().Field("TenantID").TenantKey().Hidden().Audit().TenantScoped("tenant_id")
	app.Resource(Subscription{}).Label("Subscriptions").Icon("refresh-cw").Field("Status").Enum("trialing", "active", "past_due", "canceled").Filterable().Badge().Audit().TenantScoped("tenant_id")
	app.Resource(Invoice{}).Label("Invoices").Icon("receipt").Field("Amount").Money("USD").Sortable().Field("Status").Enum("paid", "open", "failed", "refunded").Filterable().Badge().Action("Refund Invoice", nil).Danger().RequireReason().Audit().TenantScoped("tenant_id")
	app.Resource(APIKey{}).Label("API Keys").Icon("key-round").Field("Name").String().Searchable().Field("LastUsedAt").DateTime().Readonly().Action("Rotate Key", nil).RequireConfirmation().Audit().TenantScoped("tenant_id")
}
