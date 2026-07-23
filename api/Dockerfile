FROM node:20-alpine AS generator
RUN apk add --no-cache openjdk17-jre
WORKDIR /gen
COPY api/openapi.yaml openapitools.json ./
RUN npx --yes @openapitools/openapi-generator-cli generate -i openapi.yaml -g k6 -o generated

FROM grafana/k6:2.1.0
COPY --from=generator /gen/generated/script.js /scripts/script.js
ENTRYPOINT ["k6", "run", "/scripts/script.js"]
