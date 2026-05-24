package basic

import "github.com/darwvin-dev/gomyadmin/pkg/admin"

type User struct{}
type Role struct{}
type Post struct{}

func Register(app *admin.App) {
	app.Resource(User{}).
		Label("Users").
		Icon("users").
		Field("ID").UUID().Primary().Readonly().
		Field("Email").Email().Required().Searchable().Sortable().
		Field("Role").Enum("admin", "editor", "viewer").Filterable().
		Action("Reset Password", nil).RequireConfirmation().
		Audit()

	app.Resource(Role{}).
		Label("Roles").
		Icon("shield").
		Field("Name").String().Required().Searchable().Sortable().
		Field("Permissions").JSONB().
		Audit()

	app.Resource(Post{}).
		Label("Posts").
		Icon("file-text").
		Field("Title").String().Required().Searchable().Sortable().
		Field("Status").Enum("draft", "published", "archived").Filterable().Badge().
		Field("CreatedAt").DateTime().Readonly().DateRangeFilter().
		Action("Archive Selected", nil).RequireConfirmation().
		Audit()
}
