# Resource Definition

Resources describe database-backed admin surfaces. They include metadata, fields, actions, policies, audit behavior, and tenant scope.

```go
app.Resource(User{}).
    Label("Users").
    TableName("users").
    Field("Email").Email().Required().Searchable().Sortable().
    Field("Status").Enum("active", "blocked").Filterable().Badge().
    Action("Block User", BlockUser).Danger().RequireConfirmation().
    Audit().
    TenantScoped("tenant_id")
```

Generated resource files are intended to be edited and committed.
