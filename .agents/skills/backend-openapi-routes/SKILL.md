---
name: backend-openapi-routes
description: "Document changed Go backend HTTP routes with Swaggo annotations and regenerate the committed OpenAPI and ReDoc artifacts. Use when a backend route, handler registration, request/response schema, authentication rule, or API metadata is added, edited, or removed."
metadata:
  short-description: "Keep backend routes synchronized with OpenAPI"
---

# Backend OpenAPI Routes

Use this skill for backend API changes in `backend/`. Its goal is to keep the executable route set, Swaggo annotations, and committed API documentation synchronized.

## When to apply

Activate when the change adds, edits, moves, or removes any public HTTP route, or changes a route's request/response schema, authentication, status codes, or API metadata. Do not apply it to changes that only affect private service or repository behavior without changing the HTTP contract.

## Workflow

1. Inspect the route registrations and the complete diff before editing documentation. Search `backend/` for `HandleFunc`, `mux.Handle`, `@Router`, and related handler methods. Treat the registered routes as authoritative: every public route must have one matching documented operation, and removed routes must disappear from the generated contract.
2. Keep Swaggo annotations next to the handler they describe. Each operation needs a useful `@Summary`, HTTP method/path via `@Router`, request parameters or body schema, authentication/security declarations when applicable, and success plus relevant error responses. Reuse existing response and schema types instead of inventing documentation-only types.
3. Preserve the handler → service → repository boundaries. Do not move business logic into documentation work, and do not change behavior merely to make Swaggo parse a route.
4. Run [`scripts/sync_openapi.sh`](scripts/sync_openapi.sh) from any directory. It changes to `backend/`, runs the canonical `make openapi` target, and therefore generates the Swaggo contract first and builds ReDoc immediately afterward. Do not hand-edit generated files.
5. Verify the result with the script's checks: inspect the generated route list and security definitions, parse `docs/swagger.json`, ensure `docs/redoc.html` is non-empty, and run `git diff --check`. Run the backend checks required by `backend/AGENTS.md` for the affected code. If dependency downloads or another environment issue blocks generation or tests, report that as blocked rather than claiming the contract is synchronized.

## Local project invariants

- Swaggo generation must scan `cmd/api,internal/handler,internal/storage,internal/service` with `--parseInternal`; running it only from `cmd/api` can omit routes and schemas.
- The generated artifacts are committed project files. A route change is incomplete until annotations and all four generated files agree.
- Keep public-route documentation policy in `backend/AGENTS.md`; use that guide as the source of truth if the implementation evolves.
- Preserve unrelated working-tree changes and do not create commits or pull requests unless the user separately asks for them.
