FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/backend .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/backend /backend
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/backend"]
