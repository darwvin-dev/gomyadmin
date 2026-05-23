# Architecture

GoMyAdmin is split into runtime packages, generator internals, templates, examples, and docs.

The backend core is `net/http` compatible, with chi as the default adapter. The demo resource store is PostgreSQL-backed through pgx. The frontend template uses Next.js, TypeScript, Tailwind, Radix, TanStack Table, TanStack Query, React Hook Form, Zod, and Lucide icons.

The main architectural choice is explicit generated code instead of opaque runtime magic.
