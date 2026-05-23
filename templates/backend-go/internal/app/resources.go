package app

func demoResources() map[string]ResourceMeta {
	return map[string]ResourceMeta{
		"users": {
			Name: "users", Label: "Users", Icon: "users", Description: "CRM operators and support teammates", Table: "users", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "email", Label: "Email", Type: "email", Searchable: true, Sortable: true},
				{Name: "name", Label: "Name", Type: "string", Searchable: true, Sortable: true},
				{Name: "role", Label: "Role", Type: "enum", Filterable: true, EnumValues: []string{"super_admin", "tenant_admin", "manager", "support", "viewer"}},
				{Name: "status", Label: "Status", Type: "status", Filterable: true, EnumValues: []string{"active", "blocked", "pending"}},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Filterable: true, Readonly: true},
				{Name: "updated_at", Label: "Updated", Type: "datetime", Sortable: true, Readonly: true},
			},
			Actions: []ActionMeta{
				{Name: "block-user", Label: "Block User", Icon: "ban", Danger: true, RequiresConfirmation: true, RequiresReason: true},
				{Name: "reset-password", Label: "Reset Password", Icon: "key-round", RequiresConfirmation: true},
			},
		},
		"customers": {
			Name: "customers", Label: "Customers", Icon: "building-2", Description: "Companies and buyers managed by the CRM", Table: "customers", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "name", Label: "Name", Type: "string", Searchable: true, Sortable: true},
				{Name: "email", Label: "Email", Type: "email", Searchable: true, Sortable: true},
				{Name: "status", Label: "Status", Type: "status", Filterable: true, EnumValues: []string{"active", "lead", "churned"}},
				{Name: "plan", Label: "Plan", Type: "enum", Filterable: true, EnumValues: []string{"starter", "growth", "enterprise"}},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Filterable: true, Readonly: true},
				{Name: "updated_at", Label: "Updated", Type: "datetime", Sortable: true, Readonly: true},
			},
		},
		"invoices": {
			Name: "invoices", Label: "Invoices", Icon: "receipt", Description: "Billing records, payment state, and refund actions", Table: "invoices", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "customer_id", Label: "Customer", Type: "relation", Filterable: true},
				{Name: "number", Label: "Number", Type: "string", Searchable: true, Sortable: true},
				{Name: "amount", Label: "Amount", Type: "money", Sortable: true},
				{Name: "status", Label: "Status", Type: "status", Filterable: true, EnumValues: []string{"draft", "open", "paid", "failed", "refunded"}},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Filterable: true, Readonly: true},
				{Name: "updated_at", Label: "Updated", Type: "datetime", Sortable: true, Readonly: true},
			},
			Actions: []ActionMeta{
				{Name: "mark-as-paid", Label: "Mark as Paid", Icon: "check-circle", RequiresConfirmation: true},
				{Name: "refund-invoice", Label: "Refund Invoice", Icon: "undo-2", Danger: true, RequiresConfirmation: true, RequiresReason: true},
			},
		},
		"invoice_items": {
			Name: "invoice_items", Label: "Invoice Items", Icon: "list", Description: "Line items attached to invoices", Table: "invoice_items", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "invoice_id", Label: "Invoice", Type: "relation", Filterable: true},
				{Name: "description", Label: "Description", Type: "string", Searchable: true, Sortable: true},
				{Name: "quantity", Label: "Quantity", Type: "integer", Sortable: true},
				{Name: "unit_price", Label: "Unit Price", Type: "money", Sortable: true},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Readonly: true},
			},
		},
		"payments": {
			Name: "payments", Label: "Payments", Icon: "credit-card", Description: "Payment attempts and settlement records", Table: "payments", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "invoice_id", Label: "Invoice", Type: "relation", Filterable: true},
				{Name: "amount", Label: "Amount", Type: "money", Sortable: true},
				{Name: "provider", Label: "Provider", Type: "enum", Filterable: true, EnumValues: []string{"stripe", "manual", "bank"}},
				{Name: "status", Label: "Status", Type: "status", Filterable: true, EnumValues: []string{"succeeded", "failed", "pending"}},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Filterable: true, Readonly: true},
			},
		},
		"tickets": {
			Name: "tickets", Label: "Tickets", Icon: "life-buoy", Description: "Customer support queue and escalation workflow", Table: "tickets", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "customer_id", Label: "Customer", Type: "relation", Filterable: true},
				{Name: "subject", Label: "Subject", Type: "string", Searchable: true, Sortable: true},
				{Name: "priority", Label: "Priority", Type: "enum", Filterable: true, EnumValues: []string{"low", "normal", "high", "urgent"}},
				{Name: "status", Label: "Status", Type: "status", Filterable: true, EnumValues: []string{"open", "waiting", "solved"}},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Filterable: true, Readonly: true},
				{Name: "updated_at", Label: "Updated", Type: "datetime", Sortable: true, Readonly: true},
			},
			Actions: []ActionMeta{{Name: "close-ticket", Label: "Close Ticket", Icon: "check", RequiresConfirmation: true}},
		},
		"audit_logs": {
			Name: "audit_logs", Label: "Audit Logs", Icon: "database", Description: "Immutable operational activity", Table: "audit_logs", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "actor_email", Label: "Actor", Type: "email", Searchable: true, Sortable: true, Readonly: true},
				{Name: "action", Label: "Action", Type: "string", Searchable: true, Filterable: true, Readonly: true},
				{Name: "resource", Label: "Resource", Type: "string", Searchable: true, Filterable: true, Readonly: true},
				{Name: "resource_id", Label: "Resource ID", Type: "string", Searchable: true, Readonly: true},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Filterable: true, Readonly: true},
			},
		},
		"files": {
			Name: "files", Label: "Files", Icon: "file", Description: "Uploaded files and object metadata", Table: "files", PrimaryKey: "id", TenantKey: "tenant_id",
			Fields: []FieldMeta{
				{Name: "id", Label: "ID", Type: "uuid", Sortable: true, Readonly: true},
				{Name: "name", Label: "Name", Type: "string", Searchable: true, Sortable: true},
				{Name: "content_type", Label: "MIME", Type: "string", Filterable: true},
				{Name: "size", Label: "Size", Type: "integer", Sortable: true},
				{Name: "visibility", Label: "Visibility", Type: "enum", Filterable: true, EnumValues: []string{"private", "public"}},
				{Name: "tenant_id", Label: "Tenant", Type: "uuid", Hidden: true},
				{Name: "created_at", Label: "Created", Type: "datetime", Sortable: true, Readonly: true},
			},
		},
	}
}
