# PostgreSQL CRM Example

This example is a small real schema for trying the introspection flow.

```sh
createdb gomyadmin_crm
psql gomyadmin_crm < examples/postgres-crm/schema.sql
psql gomyadmin_crm < examples/postgres-crm/seed.sql

export DATABASE_URL=postgres://localhost/gomyadmin_crm?sslmode=disable
gomyadmin introspect --database-url "$DATABASE_URL" > schema.json
gomyadmin generate from-schema schema.json --package adminapp
```

The schema includes tenants, users, customers, and invoices with indexes that match common admin queries.
