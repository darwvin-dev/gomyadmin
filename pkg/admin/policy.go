package admin

// Policy allows resource-level and row-level permission decisions.
type Policy interface {
	CanView(ctx Context, actor Actor, row any) bool
	CanCreate(ctx Context, actor Actor) bool
	CanUpdate(ctx Context, actor Actor, row any) bool
	CanDelete(ctx Context, actor Actor, row any) bool
}

type AllowAllPolicy struct{}

func (AllowAllPolicy) CanView(Context, Actor, any) bool   { return true }
func (AllowAllPolicy) CanCreate(Context, Actor) bool      { return true }
func (AllowAllPolicy) CanUpdate(Context, Actor, any) bool { return true }
func (AllowAllPolicy) CanDelete(Context, Actor, any) bool { return true }

type DenyAllPolicy struct{}

func (DenyAllPolicy) CanView(Context, Actor, any) bool   { return false }
func (DenyAllPolicy) CanCreate(Context, Actor) bool      { return false }
func (DenyAllPolicy) CanUpdate(Context, Actor, any) bool { return false }
func (DenyAllPolicy) CanDelete(Context, Actor, any) bool { return false }
