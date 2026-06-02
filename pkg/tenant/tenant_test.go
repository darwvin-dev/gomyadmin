package tenant

import (
	"context"
	"testing"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

func TestActorResolverRequiresTenant(t *testing.T) {
	_, err := ActorResolver{}.Resolve(context.Background(), admin.Actor{})
	if err != ErrTenantRequired {
		t.Fatalf("expected ErrTenantRequired, got %v", err)
	}
}

func TestActorResolverReturnsTenant(t *testing.T) {
	tenant, err := ActorResolver{}.Resolve(context.Background(), admin.Actor{TenantID: "tenant_1"})
	if err != nil {
		t.Fatal(err)
	}
	if tenant.ID != "tenant_1" {
		t.Fatalf("tenant = %q", tenant.ID)
	}
}

func TestStaticResolverReturnsTenant(t *testing.T) {
	r := StaticResolver{Tenant: Tenant{ID: "t1", Name: "Acme", Slug: "acme"}}
	tenant, err := r.Resolve(context.Background(), admin.Actor{})
	if err != nil {
		t.Fatal(err)
	}
	if tenant.ID != "t1" {
		t.Fatalf("tenant ID = %q", tenant.ID)
	}
	if tenant.Name != "Acme" {
		t.Fatalf("tenant Name = %q", tenant.Name)
	}
	if tenant.Slug != "acme" {
		t.Fatalf("tenant Slug = %q", tenant.Slug)
	}
}

func TestStaticResolverEmptyIDReturnsError(t *testing.T) {
	r := StaticResolver{Tenant: Tenant{}}
	_, err := r.Resolve(context.Background(), admin.Actor{})
	if err != ErrTenantRequired {
		t.Fatalf("expected ErrTenantRequired, got %v", err)
	}
}

func TestActorResolverReturnsAllFields(t *testing.T) {
	actor := admin.Actor{TenantID: "org-42"}
	tenant, err := ActorResolver{}.Resolve(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	if tenant.ID != "org-42" {
		t.Fatalf("ID = %q", tenant.ID)
	}
}
