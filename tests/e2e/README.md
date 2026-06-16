# E2E Tests

Run these after starting a generated app:

```sh
npm --prefix tests/e2e run test:local
```

Set `GOMYADMIN_E2E_BASE_URL` when the frontend is not running on `http://localhost:3000`.

The current suite mocks the admin API and covers login, resource listing, search, filtering, create/update flow, and audit log visibility for the generated CRM demo. Playwright writes screenshots, videos, and traces to `tests/e2e/test-results` when a test fails.

CI can run the same command after starting the generated Next.js app; wiring that app startup into repository CI is a follow-up because the template is not generated as part of the default Go checks yet.
