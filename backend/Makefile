SWAGGO_VERSION := v1.16.6
REDOCLY_VERSION := 1.34.5

.PHONY: openapi redoc

openapi:
	go run github.com/swaggo/swag/cmd/swag@$(SWAGGO_VERSION) init --generalInfo main.go --dir cmd/api,internal/handler,internal/storage,internal/service,internal/model --output docs --parseInternal
	$(MAKE) redoc

redoc:
	npx --yes @redocly/cli@$(REDOCLY_VERSION) build-docs docs/swagger.yaml --output docs/redoc.html
	node -e 'const fs = require("node:fs"); const p = "docs/redoc.html"; fs.writeFileSync(p, fs.readFileSync(p, "utf8").replace(/[ \t]+$$/gm, ""));'
