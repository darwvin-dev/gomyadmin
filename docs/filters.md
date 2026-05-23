# Filters

Server-side filtering is parsed from allowed metadata only. Sorting and filters are whitelisted by field name and compiled into parameterized SQL by the PostgreSQL adapter.

Example:

```text
/admin/api/users?page=1&per_page=25&sort=-created_at&filter[role]=admin&filter[created_at][gte]=2026-01-01
```

Supported operators include exact, contains, starts with, ends with, enum, boolean, date ranges, number ranges, relation filters, and safe JSONB contains filters.
