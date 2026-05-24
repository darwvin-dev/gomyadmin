package smsgateway

import "github.com/darwvin-dev/gomyadmin/pkg/admin"

type SMSMessage struct{}
type Provider struct{}
type Campaign struct{}
type DeliveryReport struct{}
type FailedMessage struct{}

func Register(app *admin.App) {
	app.Resource(SMSMessage{}).
		Label("SMS Messages").
		Icon("message-square").
		Field("To").String().Searchable().
		Field("Body").Text().Searchable().
		Field("Status").Enum("queued", "sent", "delivered", "failed").Filterable().Badge().
		Action("Send SMS", nil).RequireConfirmation().
		Audit().
		TenantScoped("tenant_id")

	app.Resource(Provider{}).Label("Providers").Icon("radio-tower").Field("Name").String().Searchable().Field("Status").Enum("active", "degraded", "disabled").Filterable().Badge().Audit()
	app.Resource(Campaign{}).Label("Campaigns").Icon("send").Field("Name").String().Searchable().Field("ScheduledAt").DateTime().DateRangeFilter().Action("Launch Campaign", nil).RequireConfirmation().Audit().TenantScoped("tenant_id")
	app.Resource(DeliveryReport{}).Label("Delivery Reports").Icon("chart-no-axes-column").Field("ProviderMessageID").String().Searchable().Field("DeliveredAt").DateTime().Readonly().Audit().TenantScoped("tenant_id")
	app.Resource(FailedMessage{}).Label("Failed Messages").Icon("circle-alert").Field("Reason").Text().Searchable().Action("Retry Message", nil).RequireConfirmation().Audit().TenantScoped("tenant_id")
}
