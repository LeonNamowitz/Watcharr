# Repository Guidelines

## Architecture

Watcharr is a Go API with a SvelteKit frontend:

- `server/feature/` contains feature handlers. Shared persistence, domain models,
  media integrations, jobs, and utilities live in their corresponding `server/`
  packages.
- `src/routes/` contains SvelteKit pages and layouts; reusable UI and API helpers
  live in `src/lib/`, with global styles in `src/styles/`.
- `static/` contains web assets and `doc/` contains the Docusaurus site. Do not
  edit generated `build/` or `.svelte-kit/` output.

SvelteKit route groups such as `(app)`, `(plain)`, and `(public)` organize layouts
but do not add URL segments. Before adding a route, check all groups for the same
resolved URL to avoid duplicate-route conflicts.

## Development and Validation

Run commands from the repository root unless a command changes directory:

- Install frontend dependencies: `npm install`
- Start the frontend: `make dev_host` or `npm run dev`
- Start the backend in development mode: `make server`
- Check Svelte and TypeScript: `npm run check`
- Check frontend formatting and lint: `npm run lint`
- Build the production frontend: `npm run build`
- Test the backend: `cd server && go test ./...`
- Check Go formatting: `cd server && gofmt -l .` (clean output is empty)

Use validation proportional to the change. The frontend PR workflow runs
`npm run lint`; also run `npm run check` for frontend behavior/type changes and
`npm run build` when routing, bundling, or production output may be affected. The
backend PR workflow runs both `go test ./...` and the Go formatting check.

## Code Conventions

- Follow `.prettierrc`: tabs, 80-column lines, semicolons, double quotes, and
  trailing commas. Prefer formatting touched frontend files; `npm run format`
  rewrites the whole repository.
- Use PascalCase for Svelte components and preserve SvelteKit filenames such as
  `+page.svelte` and `+layout.svelte`.
- Follow idiomatic Go naming and run `gofmt` on every changed Go file.
- Keep backend tests beside their packages as `*_test.go`. The frontend has no
  dedicated unit-test command, so use the checks above plus focused manual UI
  verification when interaction behavior changes.

## Compatibility and Data Boundaries

Prefer small, non-invasive changes that remain compatible with the upstream
project. Reuse established handlers, API helpers, and UI flows before introducing
parallel abstractions. Preserve existing API payloads and logged-in behavior unless
the task explicitly changes them.

Avoid database schema and migration changes when another implementation is
reasonable. If a schema change is genuinely required, document the compatibility
impact and coordinate it before implementing it.

Public sharing must remain anonymous, owner-scoped, and read-only. Public handlers
must validate the requested owner and privacy settings before returning data or
making external lookups; never substitute the signed-in viewer's data for the
shared owner's data or expose mutations through public routes.

## Contributions

Keep commits narrowly scoped. As requested by `CONTRIBUTING.md`, discuss changes
larger than a few lines in an issue before opening a contribution. A pull request
should describe user-visible behavior and validation, include screenshots for
visual changes, and disclose AI assistance using `.github/PULL_REQUEST_TEMPLATE.md`.
Do not include credentials, runtime data from `server/data/`, or generated output.
