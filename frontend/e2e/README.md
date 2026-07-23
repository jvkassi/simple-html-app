# E2E synthetic test (real browser)

Plain k6 browser test — `test.js` is the whole thing, no separate
scenario/page config layer. Run it directly:

```sh
K6_BROWSER_HEADLESS=true k6 run frontend/e2e/test.js
# or via the npm script from the repo root:
npm run test:e2e
# or against a different environment:
K6_BROWSER_HEADLESS=true BASE_URL=https://... k6 run frontend/e2e/test.js
```

`frontend2/e2e/` is the same pattern against the second frontend
(served at `/v2`) — a separate, independent test.js, not shared code.
