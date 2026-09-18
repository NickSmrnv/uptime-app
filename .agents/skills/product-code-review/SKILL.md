---
name: product-code-review
description: Review uptime-app diffs across Go backend and Next.js frontend, run scoped static checks, and report evidence-backed findings ordered from P0 to P4.
metadata:
  short-description: Prioritized Go and Next.js code review
---

# Product Code Review

Use this skill when reviewing a commit, branch, pull request, or working-tree diff in `uptime-app`. The skill is review-only: do not modify product code, generated artifacts, or configuration unless the user separately asks for fixes.

## Review contract

Review the smallest relevant diff and its surrounding code. Every finding must include:

- priority (`P0` to `P4`), with `P0` most urgent;
- exact file and line (or the narrowest available diff location);
- concrete failure, risk, or violated project rule;
- why it matters and a focused correction direction.

Do not report style preferences without a user-visible, correctness, security, maintainability, or architecture consequence. Distinguish confirmed findings from questions and from check failures caused by the environment.

## Workflow

1. Establish the review target:
   - inspect `git status --short --branch`;
   - identify the commit/branch and comparison base (`--base <ref>` when supplied; otherwise use the PR base, upstream branch, or immediately preceding commit);
   - collect the commit subject/body and changed paths with `git log` and `git diff --stat`;
   - inspect the complete diff, then read surrounding files needed to verify behavior.
2. Run scoped checks before making review judgments:
   ```text
   python3 .codex/skills/product-code-review/scripts/run_static_checks.py --base <review-base>
   ```
   Use `--staged` or `--working-tree` when the review target is not a commit. The helper detects `backend/` and `frontend/` paths and runs only applicable checks. Record each command and result; a failed check is not automatically a code finding.
3. Read [review-checklist.md](references/review-checklist.md). Always apply `Universal`, `Security`, `Testing`, `Naming and dead code`, and `Priority and evidence`. Apply backend sections for `backend/` paths and frontend sections for `frontend/` paths. For a cross-application change, inspect the API/data boundary in both directions.
4. Walk the checklist against the diff. Trace changed functions, handlers, API consumers, environment variables, and tests instead of assuming a hunk is self-contained.
5. Report findings in descending priority, followed by checks, positive observations, and coverage limitations. If there are no findings, say so explicitly.

## Repository architecture to enforce

This is one repository with two applications:

- `backend/`: Go 1.24 API. `cmd/api` is the executable entry point. `internal/config` loads configuration; `internal/handler` owns HTTP transport, validation, and translation; `internal/service` owns business rules; `internal/repository` owns persistence; `internal/server` owns composition and server setup; `pkg` is for deliberately reusable exported packages.
- `frontend/`: Next.js 16 App Router. Routes and route-local UI live in `src/app/`; shared components are extracted only when genuinely shared; global styles belong in `src/app/globals.css`; static assets belong in `public/`.

Backend dependency direction is handler -> service -> repository. Handlers must not access repositories directly, services must not contain HTTP concerns, and repositories must not implement business policy. Public route changes require Swaggo annotations and synchronized `backend/docs` artifacts via `make openapi`.

Frontend code must respect server/client boundaries, use the established `AuthProvider.apiFetch` path for protected requests, keep access JWTs in memory, and rely on the refresh cookie for restoration. Do not move credentials into `localStorage` or `sessionStorage`. Check that route changes are reachable through existing navigation and that API error/loading states are handled.

## Output format

Use this compact structure:

```markdown
## Review findings

### P1 — Short imperative finding
- Location: `path/to/file.ts:42`
- Evidence: what the diff does and the relevant caller/data flow.
- Impact: concrete security, correctness, reliability, or maintainability consequence.
- Recommendation: smallest safe correction.

## Checks
- PASS/FAIL/SKIPPED — command — relevant output or reason.

## Positive observations
- Specific practices that reduce risk.

## Coverage and assumptions
- Files/paths reviewed and anything not verifiable locally.
```

Use `P0` for exploitable security issues, data loss/corruption, broken production startup, or a release-blocking regression. Use `P1` for severe user or API breakage, auth/authorization defects, or missing required contract changes. Use `P2` for important correctness, test, architecture, or maintainability gaps. Use `P3` for localized non-blocking issues. Use `P4` for optional improvements. Never inflate priority to compensate for uncertainty.
