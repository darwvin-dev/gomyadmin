# Deployment

Production deployments should run the Go backend and Next.js frontend as separate services behind HTTPS.

Checklist:

- set secure cookie flags
- use a strong session secret
- restrict CORS origins
- configure PostgreSQL pooling and TLS
- configure S3/R2 storage
- run migrations before deploy
- run `gomyadmin doctor`
- enable audit retention
- add tenant isolation tests in CI
