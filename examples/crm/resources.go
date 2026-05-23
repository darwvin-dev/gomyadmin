package crm

import "github.com/darwvin/gomyadmin/pkg/admin"

type Lead struct{}
type Customer struct{}
type Deal struct{}
type Task struct{}
type Note struct{}

func Register(app *admin.App) {
	app.Resource(Lead{}).
		Label("Leads").
		Icon("target").
		Field("Email").Email().Searchable().Sortable().
		Field("Source").Enum("website", "referral", "campaign").Filterable().
		Field("Status").Enum("new", "qualified", "lost").Filterable().Badge().
		Action("Assign Lead", nil).RequireConfirmation().
		Audit().
		TenantScoped("tenant_id")

	app.Resource(Customer{}).Label("Customers").Icon("building").Field("Name").String().Searchable().Sortable().Audit().TenantScoped("tenant_id")
	app.Resource(Deal{}).Label("Deals").Icon("badge-dollar-sign").Field("Amount").Money("USD").Sortable().Field("Stage").Enum("open", "won", "lost").Filterable().Badge().Audit().TenantScoped("tenant_id")
	app.Resource(Task{}).Label("Tasks").Icon("check-square").Field("Title").String().Searchable().Field("DueAt").DateTime().DateRangeFilter().Audit().TenantScoped("tenant_id")
	app.Resource(Note{}).Label("Notes").Icon("sticky-note").Field("Body").Markdown().Searchable().Audit().TenantScoped("tenant_id")
}
