# Multi-Tenancy

The first-class strategy is `tenant_id` column scoping:

```go
app.Resource(Invoice{}).TenantScoped("tenant_id")
```

The tenant resolver derives the active tenant from the authenticated actor. Future strategies include schema-per-tenant and database-per-tenant.
