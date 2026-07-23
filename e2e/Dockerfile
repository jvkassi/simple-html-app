FROM grafana/k6:2.1.0-with-browser
COPY e2e/ /e2e/
WORKDIR /e2e
ENTRYPOINT ["k6", "run", "test.js"]
