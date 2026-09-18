# Backend Guidelines

## Structure

This is a Go 1.24 API project. The executable entry point is `cmd/api/main.go`. Organize application code by responsibility:

- `internal/config` for configuration loading.
- `internal/handler` for HTTP transport and request/response handling.
- `internal/service` for business rules.
- `internal/repository` for data access.
- `internal/server` for server setup.
- `pkg` for intentionally reusable exported packages.

Handlers should validate and translate HTTP input, then call services. Keep business rules in services and persistence details in repositories; handlers must not access repositories directly.

## OpenAPI Documentation

Every public HTTP route must have Swaggo annotations that describe its request parameters, authentication, success response, and possible error responses. When a route, its request or response schema, authentication, or API metadata changes, regenerate and commit the OpenAPI contract and ReDoc page:

```bash
make openapi
```

Keep the generated `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`, and `docs/redoc.html` synchronized with the annotations.

## Development and Validation

Run commands from `backend/`:

```bash
go run ./cmd/api   # run the API entry point
go test ./...      # run every package test
go vet ./...       # report common correctness issues
gofmt -w path/to/file.go  # format a Go source file
```

Run `go test ./...` and `go vet ./...` before review. Add tests beside the package they exercise, using files ending in `_test.go` and test names such as `TestCreateCheck`.

## Style and Configuration

Use `gofmt` formatting and idiomatic Go naming: exported identifiers use `PascalCase`, unexported identifiers use `camelCase`, and package names are concise lowercase words. Keep dependencies minimal and record them through `go.mod` and `go.sum`. Never commit credentials or environment-specific secrets; load configuration through the `internal/config` layer as it is introduced.
