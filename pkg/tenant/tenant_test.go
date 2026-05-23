package tenant

import (
	"context"
	"testing"

	"github.com/darwvin/gomyadmin/pkg/admin"
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
