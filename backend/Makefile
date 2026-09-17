SWAGGO_VERSION := v1.16.6
REDOCLY_VERSION := 1.34.5

.PHONY: openapi

openapi:
	go run github.com/swaggo/swag/cmd/swag@$(SWAGGO_VERSION) init --generalInfo main.go --dir cmd/api,internal/handler,internal/storage,internal/service --output docs --parseInternal
	npx --yes @redocly/cli@$(REDOCLY_VERSION) build-docs docs/swagger.yaml --output docs/redoc.html
