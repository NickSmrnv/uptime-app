---
name: frontend-visual-verification
description: Visually verify uptime-app web frontend changes with the project Playwright MCP before completion. Use whenever a change can affect rendered layout, styling, content flow, responsive behavior, navigation, or browser interaction; skip backend-only and provably non-visual changes.
metadata:
  short-description: Verify frontend changes with Playwright MCP
---

# Frontend Visual Verification

Treat Playwright MCP verification as a required completion gate for visual or interactive changes under `frontend/`. Static checks, unit tests, builds, code inspection, and screenshots produced without exercising the changed page do not replace this gate.

## Approval gate

Immediately before opening or controlling a browser through Playwright MCP, explain the routes and states that will be checked and ask the user for explicit confirmation. Do not invoke Playwright MCP until that confirmation is received.

Approval to edit frontend code does not imply approval to start the browser check. If the user declines, does not answer, or Playwright MCP is unavailable, leave visual verification marked as pending or blocked and do not describe the frontend change as fully verified.

## Verification workflow

After approval:

1. Inspect the frontend diff and identify every affected route, visual state, interaction, and relevant responsive breakpoint.
2. Start or reuse the appropriate local frontend server. Do not alter production data or external accounts merely to make a state reachable.
3. Use the project-configured Playwright MCP browser tools to open each affected route. Verify at least one representative desktop viewport and one mobile viewport when the layout is responsive.
4. Exercise changed interactions and important states, including loading, empty, error, open/closed, and navigation states when they are affected and locally reachable.
5. Inspect visible output together with browser console errors, failed page requests, missing assets, clipping, overflow, overlap, focus behavior, and unexpected layout shifts.
6. Capture screenshot evidence for the changed states. When a regression is found and the requested task authorizes a fix, correct it and repeat the affected checks.
7. Report the exact routes, viewports, states, and observed results. Separate passed checks from blocked or unreachable states; never infer visual success from source code alone.

If authentication, test data, configuration, or a running service blocks a required state, report the precise blocker and ask for only the missing input or authorization needed to continue.
