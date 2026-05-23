# Troubleshooting

Run:

```sh
gomyadmin doctor
```

Common issues:

- Docker is not running or WSL integration is disabled
- `DATABASE_URL` is missing
- Node or Go versions are older than the template expects
- frontend and backend are running on mismatched URLs
- browser cookies are blocked for localhost
