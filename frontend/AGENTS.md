<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

# Frontend Guidelines

## Structure

This is a Next.js 16 App Router application. Routes, layouts, and page-specific UI live in `src/app/`; place global styles in `src/app/globals.css`. Store static assets in `public/`. Keep components close to the route that uses them; extract a component only when it is shared.

## Development and Validation

Run commands from `frontend/`:

```bash
npm install       # install dependencies from package-lock.json
npm run dev       # start the development server
npm run lint      # run ESLint and Next.js checks
npm run build     # create a production build
npm run start     # serve the production build
```

Run `npm run lint` for every frontend change and `npm run build` when changing routing, configuration, or production behavior. No frontend test runner is configured yet; introduce tests with a documented package script when adding a testing framework.

## Style

Use TypeScript and React function components. Follow the existing two-space indentation, double quotes, and semicolon style. Name exported component files in `PascalCase`; use `kebab-case` for route directories. Prefer Tailwind utility classes. Add CSS to `globals.css` only when it must apply across routes.

Use `next/image` for local and remote images when applicable, with useful `alt` text. Do not edit generated output such as `.next/`.
