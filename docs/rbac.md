# RBAC and ABAC

Default roles are `super_admin`, `tenant_admin`, `manager`, `support`, and `viewer`.

Permissions use strings such as `users.view`, `users.create`, `users.actions.block`, `invoices.refund`, `audit.view`, and `settings.manage`.

Frontend checks may hide controls, but backend policy checks are authoritative.
