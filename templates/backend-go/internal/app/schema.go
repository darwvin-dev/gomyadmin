package app

const schemaSQL = `
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
`

const seedSQL = `
insert into tenants (id, name, slug) values
  ('tenant_1', 'Northstar CRM', 'northstar'),
  ('tenant_2', 'Acme Telecom', 'acme')
on conflict (id) do nothing;

insert into users (id, tenant_id, email, name, role, status, password_hash, created_at, updated_at) values
  ('usr_1', 'tenant_1', 'admin@example.com', 'Darwin Admin', 'super_admin', 'active', $1, now() - interval '5 days', now()),
  ('usr_2', 'tenant_1', 'support@example.com', 'Nima Support', 'support', 'active', $1, now() - interval '4 days', now()),
  ('usr_3', 'tenant_2', 'manager@acme.test', 'Acme Manager', 'manager', 'pending', $1, now() - interval '3 days', now())
on conflict (id) do nothing;

insert into customers (id, tenant_id, name, email, status, plan, created_at, updated_at) values
  ('cus_1', 'tenant_1', 'Atlas Retail', 'ops@atlas.test', 'active', 'enterprise', now() - interval '20 days', now()),
  ('cus_2', 'tenant_1', 'Signal Clinic', 'billing@signal.test', 'lead', 'growth', now() - interval '10 days', now()),
  ('cus_3', 'tenant_1', 'Nexavo Labs', 'team@nexavo.test', 'active', 'starter', now() - interval '3 days', now()),
  ('cus_4', 'tenant_2', 'Acme Mobile', 'admin@acme.test', 'active', 'enterprise', now() - interval '30 days', now())
on conflict (id) do nothing;

insert into invoices (id, tenant_id, customer_id, number, amount, status, created_at, updated_at) values
  ('inv_1', 'tenant_1', 'cus_1', 'INV-1001', 2400, 'paid', now() - interval '8 days', now()),
  ('inv_2', 'tenant_1', 'cus_1', 'INV-1002', 1200, 'open', now() - interval '2 days', now()),
  ('inv_3', 'tenant_1', 'cus_2', 'INV-1003', 890, 'failed', now() - interval '1 day', now()),
  ('inv_4', 'tenant_2', 'cus_4', 'INV-2001', 5100, 'paid', now() - interval '6 days', now())
on conflict (id) do nothing;

insert into invoice_items (id, tenant_id, invoice_id, description, quantity, unit_price, created_at) values
  ('item_1', 'tenant_1', 'inv_1', 'CRM Enterprise seats', 12, 200, now() - interval '8 days'),
  ('item_2', 'tenant_1', 'inv_2', 'Usage overage', 1, 1200, now() - interval '2 days'),
  ('item_3', 'tenant_1', 'inv_3', 'Support package', 1, 890, now() - interval '1 day'),
  ('item_4', 'tenant_2', 'inv_4', 'Telecom billing module', 1, 5100, now() - interval '6 days')
on conflict (id) do nothing;

insert into payments (id, tenant_id, invoice_id, amount, provider, status, created_at) values
  ('pay_1', 'tenant_1', 'inv_1', 2400, 'stripe', 'succeeded', now() - interval '7 days'),
  ('pay_2', 'tenant_1', 'inv_3', 890, 'stripe', 'failed', now() - interval '20 hours'),
  ('pay_3', 'tenant_2', 'inv_4', 5100, 'bank', 'succeeded', now() - interval '5 days')
on conflict (id) do nothing;

insert into tickets (id, tenant_id, customer_id, subject, priority, status, created_at, updated_at) values
  ('tic_1', 'tenant_1', 'cus_1', 'Invoice PDF is missing VAT number', 'high', 'open', now() - interval '6 hours', now()),
  ('tic_2', 'tenant_1', 'cus_2', 'Webhook retries are delayed', 'normal', 'waiting', now() - interval '1 day', now()),
  ('tic_3', 'tenant_2', 'cus_4', 'Need monthly CDR export', 'urgent', 'open', now() - interval '2 days', now())
on conflict (id) do nothing;

insert into audit_logs (id, actor_id, actor_email, tenant_id, action, resource, resource_id, old_values, new_values, metadata, ip_address, user_agent, request_id, created_at) values
  ('audit_1', 'usr_1', 'admin@example.com', 'tenant_1', 'login', 'auth', 'usr_1', null, null, '{}', '127.0.0.1', 'seed', 'seed-1', now() - interval '2 hours'),
  ('audit_2', 'usr_1', 'admin@example.com', 'tenant_1', 'update', 'invoices', 'inv_2', '{"status":"draft"}', '{"status":"open"}', '{}', '127.0.0.1', 'seed', 'seed-2', now() - interval '90 minutes'),
  ('audit_3', 'usr_2', 'support@example.com', 'tenant_1', 'custom_action:mark-as-paid', 'invoices', 'inv_1', null, '{"status":"paid"}', '{}', '127.0.0.1', 'seed', 'seed-3', now() - interval '45 minutes')
on conflict (id) do nothing;
`
