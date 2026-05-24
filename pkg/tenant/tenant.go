package tenant

import (
	"context"
	"errors"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

var ErrTenantRequired = errors.New("tenant is required")

type Tenant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Resolver interface {
	Resolve(ctx context.Context, actor admin.Actor) (Tenant, error)
}

type ActorResolver struct{}

func (ActorResolver) Resolve(_ context.Context, actor admin.Actor) (Tenant, error) {
	if actor.TenantID == "" {
		return Tenant{}, ErrTenantRequired
	}
	return Tenant{ID: actor.TenantID}, nil
}

type StaticResolver struct {
	Tenant Tenant
}

func (r StaticResolver) Resolve(context.Context, admin.Actor) (Tenant, error) {
	if r.Tenant.ID == "" {
		return Tenant{}, ErrTenantRequired
	}
	return r.Tenant, nil
}
