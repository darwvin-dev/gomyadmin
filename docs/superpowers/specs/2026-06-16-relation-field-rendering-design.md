# Relation Field Rendering (frontend) — Design

Issue: #3 "Add relation field rendering in the frontend template"

## Problem

The Go backend already models relation fields and serializes them in
`GET /admin/api/resources` as:

```json
{ "type": "relation", "relation": { "resource": "customers", "foreign_key": "customer_id", "display_field": "name", "kind": "belongs_to" } }
```

The Next.js admin template does not handle `type: "relation"`:
`FieldMeta` has no `relation` property, and neither the list grid cell nor the
form field input branch on `"relation"`. A relation field therefore renders as
its raw foreign-key id (e.g. `cus_001`) in tables and as a plain text input in
forms.

## Scope

In scope (this change):

- `belongs_to` relations only.
- Readable label instead of the raw id in the resource list grid.
- A `<select>` selector in the create/edit form, populated from the target
  resource's records.
- Client-side label resolution (no Go backend change).
- Demo API emits relation metadata so the live demo exercises this.
- Playwright coverage and a roadmap tick.

Out of scope: `has_many` rendering, async/typeahead selectors, a dedicated
detail/show page, and any Go backend change.

## Design

### 1. Types — `lib/api.ts`

Extend `FieldMeta` with an optional relation descriptor matching the backend
JSON:

```ts
export type RelationMeta = {
  resource: string
  foreign_key?: string
  display_field?: string
  kind: string
}
// FieldMeta gains: relation?: RelationMeta
```

### 2. Client-side resolution — shared hook

A small module (e.g. `components/admin/relation-field.tsx`) exports:

- `useRelationOptions(relation: RelationMeta)` — fetches the target resource
  list via `api.list(relation.resource, new URLSearchParams())` under
  `queryKey: ["relation-options", relation.resource]`. React Query dedupes by
  key, so many cells/inputs pointing at the same resource trigger one fetch.
  Returns `{ options: { id: string; label: string }[], labelFor(id): string, isLoading }`.
- `relationLabel(record, relation)` — pure helper picking the display label:
  `record[display_field]` if `display_field` is set and present; otherwise the
  first present field among `name`, `title`, `label`, `email`; otherwise the
  raw id string. Exported for direct unit reasoning and reuse.

Records are keyed by `id` (the convention already used across the template).

### 3. List grid — `data-grid.tsx` `Cell`

Add a branch: when `field.type === "relation" && field.relation`, render
`<RelationLabel relation={field.relation} value={value} />`. That component
uses `useRelationOptions` and shows `labelFor(String(value))`. While loading or
if unresolved, it shows the raw id string so the cell is never empty.

### 4. Form — `resource-form.tsx` `FieldInput`

Add a branch before the generic input: when `field.type === "relation" &&
field.relation`, render `<RelationSelect>`. It is a native `<select>`
(consistent with the existing enum select) with:

- a leading empty `"Select"` option,
- one `<option value={id}>{label}</option>` per resolved target record,
- `defaultValue = String(value ?? "")`,
- `onChange` calling the field's `onChange` with the selected id string.

`editableFields` already excludes hidden/readonly/id/created_at, so relation FKs
remain editable.

### Label fallback

If the target list is still loading, or the id is not found, or no display
field resolves, fall back to the raw id. This keeps the UI functional even when
`display_field` is absent or the relation target is empty.

### 5. Demo API — `deploy/cloudflare/worker/src/demo-api.mjs`

Extend the `field()` helper to pass through an optional `relation` object, and
declare the existing relation fields with metadata:

- `invoices.customer_id` → `relation: { resource: "customers", display_field: "name", foreign_key: "customer_id", kind: "belongs_to" }`
- `tickets.customer_id` → same.

Records already store `customer_id` as a customer id (`cus_001`...), so the
frontend can resolve labels against the `customers` list. Redeploy the demo API
worker (through the local proxy) so the live demo reflects this.

### 6. Tests — `tests/e2e/admin.spec.ts`

Add a mocked resource (or extend the existing one) that includes a `relation`
field plus a target resource, and assert:

- the list grid shows the human-readable label (target record's display field),
  not the raw id;
- the edit form renders a `<select>` whose options include the target labels,
  and selecting one submits the target id.

Mocks follow the existing `mockAdminAPI` pattern (route `**/admin/api/**`).

### 7. Roadmap

Check off "Relation field rendering in the frontend" in the README roadmap.

## Verification

- `yarn typecheck` and `yarn build` pass in `templates/frontend-next-shadcn`.
- Playwright relation specs pass.
- On the live demo, Invoices/Tickets show the customer **name** in the
  `Customer` column and a customer `<select>` in the edit form.
