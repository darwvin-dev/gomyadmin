# Security Policy

GoMyAdmin is designed for production backoffice systems, so security reports are handled as first-priority maintenance.

## Supported Versions

Security fixes target the latest minor release. Until the project reaches v1, only the `main` branch is supported.

## Reporting a Vulnerability

Please do not open a public issue for a suspected vulnerability. Email the maintainer with:

- affected version or commit
- reproduction steps
- expected impact
- suggested fix, if known

You should receive an acknowledgement within 72 hours.

## Security Model

GoMyAdmin assumes all authorization decisions are enforced on the Go backend. Frontend permission checks are treated only as UX hints.

Production deployments should enable:

- HTTPS-only secure cookies
- strong `GOMYADMIN_SESSION_SECRET`
- restricted CORS origins
- login rate limiting
- PostgreSQL TLS where available
- private object storage for sensitive files
- tenant isolation tests in CI
- audit log retention and export controls

## Defaults

The demo uses `admin@example.com` and `password` for local development only. Generated applications must rotate credentials, set secure cookies, and configure production storage before public exposure.
