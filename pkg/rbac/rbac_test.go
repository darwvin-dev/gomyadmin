package rbac

import (
	"testing"

	"github.com/darwvin/gomyadmin/pkg/admin"
)

func TestAuthorizerWildcardPermissions(t *testing.T) {
	authorizer := NewDefaultAuthorizer()
	actor := admin.Actor{Roles: []string{RoleViewer}}
	if !authorizer.Can(actor, "users.view") {
		t.Fatal("viewer should be able to view users")
	}
	if authorizer.Can(actor, "users.delete") {
		t.Fatal("viewer should not delete users")
	}
}
