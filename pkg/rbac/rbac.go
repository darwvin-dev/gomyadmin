package rbac

import "github.com/darwvin-dev/gomyadmin/pkg/admin"

const (
	RoleSuperAdmin  = "super_admin"
	RoleTenantAdmin = "tenant_admin"
	RoleManager     = "manager"
	RoleSupport     = "support"
	RoleViewer      = "viewer"
)

type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type Authorizer struct {
	roles map[string]Role
}

func NewDefaultAuthorizer() *Authorizer {
	authorizer := &Authorizer{roles: map[string]Role{}}
	for _, role := range DefaultRoles() {
		authorizer.roles[role.Name] = role
	}
	return authorizer
}

func DefaultRoles() []Role {
	return []Role{
		{Name: RoleSuperAdmin, Description: "Unrestricted platform administrator", Permissions: []string{"*"}},
		{Name: RoleTenantAdmin, Description: "Manage tenant resources", Permissions: []string{"users.*", "invoices.*", "files.*", "audit.view"}},
		{Name: RoleManager, Description: "Manage operational workflows", Permissions: []string{"users.view", "users.update", "invoices.view", "invoices.update", "audit.view"}},
		{Name: RoleSupport, Description: "Read and support customer records", Permissions: []string{"users.view", "invoices.view", "audit.view"}},
		{Name: RoleViewer, Description: "Read-only access", Permissions: []string{"*.view"}},
	}
}

func (a *Authorizer) Can(actor admin.Actor, permission string) bool {
	if actor.Can(permission) {
		return true
	}
	for _, roleName := range actor.Roles {
		role, ok := a.roles[roleName]
		if !ok {
			continue
		}
		for _, granted := range role.Permissions {
			if adminMatchPermission(granted, permission) {
				return true
			}
		}
	}
	return false
}

func adminMatchPermission(granted, requested string) bool {
	if granted == requested || granted == "*" {
		return true
	}
	if len(granted) > 2 && granted[len(granted)-2:] == ".*" {
		return len(requested) >= len(granted)-1 && requested[:len(granted)-1] == granted[:len(granted)-1]
	}
	if len(granted) > 2 && granted[:2] == "*." {
		suffix := granted[1:]
		return len(requested) >= len(suffix) && requested[len(requested)-len(suffix):] == suffix
	}
	return false
}
