# E2E Tests

Run these after starting a generated app:

```sh
cd tests/e2e
npx playwright test
```

Set `GOMYADMIN_E2E_BASE_URL` when the frontend is not running on `http://localhost:3000`.

End-to-end browser tests should cover login, resource listing, filtering, sorting, custom actions, audit visibility, and file upload once Playwright is added to the frontend template.
