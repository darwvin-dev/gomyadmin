# Existing chi App

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
)

type User struct{}

func main() {
	app := admin.New("Acme Admin")
	app.Resource(User{}).
		TableName("users").
		Field("ID").String().Primary().Readonly().
		Field("Email").Email().Searchable().Sortable().
		Field("Status").Enum("active", "blocked").Filterable()

	adminServer, err := server.New(context.Background(), server.Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		App:         app,
		Authenticate: func(ctx context.Context, email, password string) (admin.Actor, bool, error) {
			return admin.Actor{ID: "admin", Email: email, Roles: []string{"super_admin"}, Permissions: []string{"*"}}, true, nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer adminServer.Close()

	r := chi.NewRouter()
	r.Mount("/admin", http.StripPrefix("/admin", adminServer.Handler()))
	log.Fatal(http.ListenAndServe(":8080", r))
}
```
