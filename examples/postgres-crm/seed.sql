insert into tenants (id, name, slug) values
  ('00000000-0000-0000-0000-000000000001', 'Northstar CRM', 'northstar');

insert into users (id, tenant_id, email, name, role, status) values
  ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'admin@example.com', 'Darwin Admin', 'admin', 'active'),
  ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001', 'support@example.com', 'Support Lead', 'support', 'active');

insert into customers (id, tenant_id, name, email, status, plan) values
  ('00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', 'Acme Telecom', 'ops@acme.test', 'active', 'scale'),
  ('00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000001', 'Nova Logistics', 'admin@nova.test', 'lead', 'starter');

insert into invoices (id, tenant_id, customer_id, number, amount, status, due_date) values
  ('00000000-0000-0000-0000-000000000301', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000201', 'INV-1001', 4800.00, 'open', current_date + 14),
  ('00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000202', 'INV-1002', 900.00, 'paid', current_date - 7);
