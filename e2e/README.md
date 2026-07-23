# E2E synthetic tests (real browser)

Plain k6 browser test — `test.js` is the whole thing, no separate
scenario/page config layer. Run it directly:

```sh
K6_BROWSER_HEADLESS=true k6 run e2e/test.js
# or against a different environment:
K6_BROWSER_HEADLESS=true BASE_URL=https://... k6 run e2e/test.js
```

`e2e2/` is the same pattern against the second frontend (`frontend2`,
served at `/v2`) — a separate, independent test.js, not shared code.
