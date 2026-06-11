# Integration Tests

Default integration smoke tests run with:

```sh
go test ./tests/integration
```

Docker Compose smoke tests are opt-in because they build and run containers and may need to pull base images. The smoke test currently verifies the generated PostgreSQL + backend path only.

```sh
GOMYADMIN_RUN_DOCKER_COMPOSE_TEST=1 go test -tags=integration ./tests/integration
```

Integration coverage should run against PostgreSQL using the CI service container. The current unit suite covers the critical query, permission, tenant, audit, auth, and storage primitives; database-backed CRUD tests belong here as the persistent store matures.
