# E2E synthetic tests (real browser)

Declarative, tool-agnostic test structure on top of k6's browser module
(k6's own recommended pattern is Page Object Model — this is that, plus a
generic interpreter so scenarios are data, not code):

- `pages/*.json` — one file per page: a friendly-name → CSS selector map,
  plus the page's path. Update here when the DOM changes; scenarios don't
  need to.
- `scenarios/*.json` — a named list of steps (`goto`, `fill`, `click`,
  `waitFor`, `assertContains`) referencing selectors by friendly name.
  Editable by anyone without knowing k6 or Playwright's API.
- `lib/runner.js` — the generic interpreter. Add new step actions here,
  not per-scenario.
- `test.js` — k6 entrypoint; loads `$SCENARIO` (env var, defaults to
  `create-and-view-note.json`) and runs it against `$BASE_URL`.

Run it:

```sh
K6_BROWSER_HEADLESS=true k6 run e2e/test.js
# or against a different scenario/environment:
K6_BROWSER_HEADLESS=true SCENARIO=./scenarios/other.json BASE_URL=https://... k6 run e2e/test.js
```

Adding a new scenario means adding a new `scenarios/*.json` file — no new
JS. Adding a new page means adding a new `pages/*.json` selector map.
