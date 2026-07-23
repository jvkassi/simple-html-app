# API synthetic tests

`openapi.yaml` is the source of truth for the public API contract (only
externally-reachable routes — internal probe endpoints like `/healthz`
and `/readyz` are deliberately excluded, since they're not exposed through
the Ingress).

Regenerate the k6 script after any spec change:

```sh
npx --yes @openapitools/openapi-generator-cli generate \
  -i api/openapi.yaml -g k6 -o api/generated
```

Run it:

```sh
k6 run api/generated/script.js
```

`api/generated/` is a build artifact (regenerate, don't hand-edit) — not
committed. Same pattern works with a Postman collection instead of
OpenAPI via [`postman-to-k6`](https://github.com/grafana/postman-to-k6)
if a team already maintains one of those instead.
