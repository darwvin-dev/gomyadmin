create table if not exists tenants (
  id text primary key,
  name text not null,
  slug text not null unique,
  created_at timestamptz not null default now()
);

create table if not exists users (
  id text primary key,
  tenant_id text not null references tenants(id),
  email text not null unique,
  name text not null,
  role text not null,
  status text not null default 'active',
  password_hash text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists customers (
  id text primary key,
  tenant_id text not null references tenants(id),
  name text not null,
  email text not null,
  status text not null default 'lead',
  plan text not null default 'starter',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists invoices (
  id text primary key,
  tenant_id text not null references tenants(id),
  customer_id text not null references customers(id) on delete cascade,
  number text not null,
  amount numeric(12,2) not null default 0,
  status text not null default 'open',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists invoice_items (
  id text primary key,
  tenant_id text not null references tenants(id),
  invoice_id text not null references invoices(id) on delete cascade,
  description text not null,
  quantity integer not null default 1,
  unit_price numeric(12,2) not null default 0,
  created_at timestamptz not null default now()
);

create table if not exists payments (
  id text primary key,
  tenant_id text not null references tenants(id),
  invoice_id text not null references invoices(id) on delete cascade,
  amount numeric(12,2) not null default 0,
  provider text not null default 'stripe',
  status text not null default 'pending',
  created_at timestamptz not null default now()
);

create table if not exists tickets (
  id text primary key,
  tenant_id text not null references tenants(id),
  customer_id text not null references customers(id) on delete cascade,
  subject text not null,
  priority text not null default 'normal',
  status text not null default 'open',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists audit_logs (
  id text primary key,
  actor_id text,
  actor_email text,
  tenant_id text,
  action text not null,
  resource text not null,
  resource_id text,
  old_values jsonb,
  new_values jsonb,
  metadata jsonb,
  ip_address text,
  user_agent text,
  request_id text,
  created_at timestamptz not null default now()
);

create table if not exists files (
  id text primary key,
  tenant_id text not null references tenants(id),
  key text not null unique,
  name text not null,
  content_type text not null,
  size bigint not null default 0,
  visibility text not null default 'private',
  created_at timestamptz not null default now()
);

create index if not exists users_tenant_created_idx on users (tenant_id, created_at desc);
create index if not exists customers_tenant_created_idx on customers (tenant_id, created_at desc);
create index if not exists invoices_tenant_created_idx on invoices (tenant_id, created_at desc);
create index if not exists invoices_status_idx on invoices (tenant_id, status);
create index if not exists invoice_items_invoice_idx on invoice_items (invoice_id);
create index if not exists payments_invoice_idx on payments (invoice_id);
create index if not exists tickets_status_idx on tickets (tenant_id, status);
create index if not exists audit_logs_tenant_created_idx on audit_logs (tenant_id, created_at desc);
create index if not exists audit_logs_resource_idx on audit_logs (resource, resource_id);
create index if not exists files_tenant_created_idx on files (tenant_id, created_at desc);
