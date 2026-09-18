# Uptime App Review Checklist

Apply only the sections selected by changed paths, plus the universal sections. For each item, mark it mentally as pass, fail, not applicable, or not verifiable. Report failures only when the diff provides evidence.

## Universal

### Diff and behavior

- [ ] The review base, commit message, changed paths, and complete diff are known.
- [ ] Changed code is traced to its callers, data sources, error paths, and side effects.
- [ ] Changed behavior matches the requested user-visible result and does not silently broaden scope.
- [ ] Inputs are validated at the trust boundary; malformed, missing, duplicate, oversized, or unauthorized input has a deliberate result.
- [ ] Errors preserve useful context without leaking credentials, tokens, personal data, SQL, filesystem paths, or stack traces.
- [ ] Concurrency, retries, timeouts, cancellation, idempotency, and cleanup are correct where the change can run asynchronously or more than once.

### Security and secrets

- [ ] No API keys, passwords, JWTs, private keys, database URLs with credentials, cookies, or `.env` contents are committed or logged.
- [ ] Authentication is not confused with authorization; resource ownership and role checks happen on the trusted side.
- [ ] User-controlled values are not used unsafely in SQL, shell commands, filesystem paths, redirects, HTML, URLs, or dynamic code.
- [ ] Cookies, CORS, CSRF protections, upload validation, rate limits, and response headers remain appropriate for the changed flow.
- [ ] Tokens stay in the project-approved storage location: refresh token in an HttpOnly cookie and access JWT only in memory.
- [ ] New dependencies are necessary and do not introduce an obvious supply-chain or license concern.

### Tests and coverage

- [ ] Tests cover the changed success path and important rejection/error path.
- [ ] Authorization, validation, boundary, and regression cases are tested where applicable.
- [ ] Tests assert behavior rather than implementation details and are deterministic.
- [ ] Missing tests are reported only when the untested path carries meaningful regression risk.
- [ ] Static-check failures, unavailable services, and date/environment-frozen tests are reported separately from code findings.

### Naming, dead code, and maintainability

- [ ] Names describe domain intent and follow local Go/TypeScript conventions.
- [ ] Exported Go identifiers are documented when required; package names are concise lowercase words; React component files use PascalCase and route directories use kebab-case.
- [ ] No unreachable branches, unused exports, abandoned feature flags, duplicate logic, commented-out implementation, or stale imports were introduced.
- [ ] The change is minimal and does not mix unrelated formatting, dependency, or generated-file churn.
- [ ] Comments explain non-obvious invariants or tradeoffs, not what the syntax already says.

## Backend architecture

- [ ] Handler validates and translates HTTP input, calls a service, and maps service results/errors to HTTP responses.
- [ ] Service owns business rules and does not depend on HTTP request/response types.
- [ ] Repository owns persistence and query details; handlers never call repositories directly.
- [ ] Configuration comes through `internal/config`; secrets are supplied by environment/configuration rather than literals.
- [ ] Database transactions, ownership filters, nullable values, pagination, and error classification are correct.
- [ ] Changes preserve graceful shutdown, request context cancellation, timeouts, and connection-pool behavior.
- [ ] Go code is gofmt-formatted and uses idiomatic error wrapping and naming.

## API contract

- [ ] Every changed public route has complete Swaggo annotations for parameters, auth, success responses, and error responses.
- [ ] Request/response schemas, status codes, validation, and frontend consumers agree.
- [ ] `backend/docs/docs.go`, `swagger.json`, `swagger.yaml`, and `redoc.html` are regenerated when route metadata or schemas change.
- [ ] Authenticated endpoints consistently apply intended middleware and ownership checks.
- [ ] Backward compatibility, migrations, and rollout behavior are considered for persisted or public contract changes.

## Frontend architecture

- [ ] Server and client components are chosen deliberately; browser-only APIs are not evaluated during server rendering.
- [ ] Route-local UI stays near its route; extraction is justified by actual reuse and does not create a misleading shared abstraction.
- [ ] Protected API calls use the established auth provider/fetch path and handle one refresh/retry boundary without loops.
- [ ] Loading, empty, success, validation, unauthorized, and server-error states are represented where the user can encounter them.
- [ ] State ownership is local to the feature unless truly cross-route; effects have correct dependencies and cleanup.
- [ ] `next/image`, useful `alt` text, keyboard access, focus behavior, and responsive layout are preserved for relevant UI.
- [ ] Routing, navigation, metadata, environment variables, and production configuration are updated when needed.

## Accessibility and UX

- [ ] Interactive controls have an accessible name, visible focus, keyboard behavior, and an appropriate semantic element.
- [ ] Form errors are associated with fields and are not communicated by color alone.
- [ ] Destructive or irreversible actions have clear confirmation and pending/disabled behavior.
- [ ] User-visible errors are actionable and do not expose internal implementation details.
- [ ] Optimistic updates and retries cannot show stale or contradictory state.

## Good and bad signals

Good signals include narrow diffs, tests that pin new behavior, explicit validation at boundaries, clear service/repository separation, synchronized API artifacts, safe auth storage, and deliberate loading/error states.

Bad signals include direct handler-to-database access, client-side authorization, tokens in browser storage, secrets in source or logs, unbounded uploads/queries, swallowed errors, duplicated refresh loops, effects without cleanup, route changes without navigation/tests, generated docs left stale, and broad refactors mixed into a feature diff.

## Priority rubric

Prioritize by impact and exploitability, not by how easy a fix looks:

- **P0:** active or likely exploit, data loss/corruption, production startup failure, or release-blocking regression.
- **P1:** severe user/API breakage, auth or authorization defect, unsafe secret handling, or required API contract mismatch.
- **P2:** important correctness or reliability defect, missing high-value regression coverage, architecture violation that creates concrete risk, or significant accessibility failure.
- **P3:** localized non-blocking bug, maintainability issue, dead code, naming problem, or incomplete edge case.
- **P4:** optional polish or refactoring with no demonstrated current risk.
