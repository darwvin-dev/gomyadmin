create table tenants (
  id uuid primary key,
  name text not null,
  slug text not null unique,
  created_at timestamptz not null default now()
);

create table users (
  id uuid primary key,
  tenant_id uuid not null references tenants(id),
  email text not null unique,
  name text not null,
  role text not null default 'admin',
  status text not null default 'active',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table customers (
  id uuid primary key,
  tenant_id uuid not null references tenants(id),
  name text not null,
  email text not null,
  status text not null default 'lead',
  plan text not null default 'starter',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table invoices (
  id uuid primary key,
  tenant_id uuid not null references tenants(id),
  customer_id uuid not null references customers(id) on delete cascade,
  number text not null,
  amount numeric(12,2) not null default 0,
  status text not null default 'open',
  due_date date,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index users_tenant_created_idx on users (tenant_id, created_at desc);
create index customers_tenant_created_idx on customers (tenant_id, created_at desc);
create index invoices_tenant_status_idx on invoices (tenant_id, status, created_at desc);
