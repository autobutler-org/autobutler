Purpose
-------
These instructions tell GitHub Copilot how to handle programming in this repository.

Project Overview
----------------
AutoButler is a plug-and-play private cloud system built as a full-stack web application. It's designed to run on dedicated hardware in users' homes, giving them complete sovereignty over their data.

**Technology Stack:**
- **Backend:** Go 1.24+ with Gin web framework, SQLite database, golang-migrate for migrations
- **Frontend:** Vanilla JavaScript, Tailwind CSS, Flowbite Icons
- **Templating:** templ (type-safe HTML templating for Go)
- **Testing:** Playwright for E2E tests, Go standard testing for unit tests
- **Code Generation:** sqlc for database code generation, templ for template compilation
- **Observability:** OpenTelemetry for monitoring and tracing

Architecture and Project Structure
-----------------------------------
```
autobutler/
├── cmd/autobutler/          # CLI entry points (serve, install, version)
├── internal/
│   ├── server/              # HTTP server, routes, middleware
│   │   ├── api/v1/          # REST API endpoints
│   │   └── public/          # Static assets (CSS, JS, images)
│   ├── db/                  # Database layer with sqlc-generated code
│   ├── install/             # Installation and service management
│   └── update/              # Update management
├── pkg/
│   ├── botel/               # Botel (bot + hotel) infrastructure
│   ├── calendar/            # Calendar domain logic
│   ├── ui/                  # HTML UI components (templ templates)
│   └── util/                # Shared utilities (business logic lives here)
├── sql/queries/             # SQL queries for code generation
└── tests/e2e/               # End-to-end Playwright tests
```

Build, Test, and Lint Commands
-------------------------------
**Development workflow:**
- `make watch` - Run backend with hot-reloading (auto-generates code on changes)
- `make generate` - Generate templ templates and sqlc database code (NOT needed with `make watch`)
- `make build` - Build the backend binary
- `make serve` - Run backend in production mode

**Testing:**
- `make test` - Run all tests (unit + E2E)
- `make test/unit` - Run Go unit tests with coverage
- `npm run test/e2e` - Run Playwright E2E tests
- `npm run test/e2e/report` - View E2E test reports

**Linting and formatting:**
- `make lint` - Lint all code (Go, SQL, templ, JS, TS, CSS, YAML)
- `make format` - Format all code
- `make fix` - Fix common code issues

**Dependencies:**
- `make setup` - Install all development tools (gotools, sqlc, templ)
- `make upgrade` - Upgrade Go and JavaScript dependencies
- `go mod tidy` - Tidy Go module dependencies

Code Standards and Naming Conventions
--------------------------------------
**Go code:**
- Follow standard Go conventions (gofmt, go vet)
- Use meaningful variable and function names
- Keep functions focused and testable
- Separate business logic from HTTP handlers (see API endpoint architecture section)
- Use structs for function parameters when there are multiple related inputs

**JavaScript/TypeScript:**
- Use ESLint and Prettier for code style
- Prefer vanilla JavaScript over frameworks
- Keep code modular and maintainable
- Follow existing patterns in `internal/server/public/scripts/`

**Database:**
- SQL queries live in `sql/queries/` and are generated into Go code using sqlc
- Never write raw SQL in Go code - use sqlc-generated functions
- Use migrations for schema changes (`sql/migrations/`)

**Templates:**
- Use templ for type-safe HTML generation
- Template files end with `.templ` and are compiled to Go code
- Never write string concatenation for HTML - use templ components

Dependency Management
---------------------
**Go dependencies:**
- Add with `go get <package>`
- Run `go mod tidy` after adding dependencies
- Prefer standard library when possible
- Check for security vulnerabilities before adding new dependencies

**JavaScript dependencies:**
- Add with `npm install <package>`
- Keep dependencies minimal (this is a vanilla JS project)
- Run `npm run check-updates` to check for updates

**Code generation dependencies:**
- sqlc: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0`
- templ: `go install github.com/a-h/templ/cmd/templ@latest`

Key rule (always)
-----------------
- Respect the linting and formatting conventions of the various linting and formatting configurations and tools being used.
- Always use the repository's canonical stylesheet located at `internal/server/public/styles/site.css` for site-specific styling.
- When adding or changing styles for the application UI, prefer adding new rules to `internal/server/public/styles/site.css` instead of creating new top-level CSS files or embedding inline styles.

When generating HTML/templates/components
----------------------------------------
- Ensure the page or template links to the served stylesheet. Use the public path `/styles/site.css` (this file is stored in the repo at `internal/server/public/styles/site.css`). Example HTML head snippet:

	<link rel="stylesheet" href="/styles/site.css">

- Prefer applying CSS classes that already exist in `site.css`. If a needed utility/class is missing, add it to `site.css` with a clear name and comment, and then use it in the markup.

Avoid
-----
- Do not add inline styles (style="...") for persistent UI design. Small one-off debug styles are allowed in development but should be moved into `site.css` before merging.
- Do not introduce new global stylesheets at the project root. Keep site-wide styles centralized in `internal/server/public/styles/site.css`.
- **NEVER add CSS transforms (transform: scale(), translateX(), etc.) to interactive elements unless explicitly requested.** Transforms on :active, :hover, or :focus states cause positioning bugs where clicks fail to register because elements move away from the cursor during interaction. This has caused numerous bugs.

Style additions best practices
-----------------------------
- Keep new selectors specific and prefixed if needed to avoid collisions (e.g., .ab- or .site-).
- Add a short comment above any new section in `site.css` describing its purpose and where it's used.
- When changing existing styles, search for usages in `pkg/ui/` and other templates to avoid regressions.

Notes for Tailwind or other utilities
-----------------------------------
- This repository includes `tailwind.config.js`. When generating utility classes that are Tailwind-based, prefer using the existing Tailwind setup for utility-style needs, but still centralize site overrides and component-level custom CSS in `internal/server/public/styles/site.css`.

If you cannot follow the rule
---------------------------
- If a generated change absolutely requires a separate stylesheet (e.g., for a large, self-contained third-party bundle), add a short rationale in the PR description and keep it scoped to the feature directory. Prefer linking to and documenting that stylesheet in `internal/server/public/README.md`.

Backend development assumptions
-------------------------------
- Assume the developer is running the backend via `make watch` and that it will auto-reload on code changes.
- Never run the `make generate` target. Just assume the code is generated automatically as a part of `make watch`.
- Never attempt to start, stop, or restart the backend server yourself.
- Focus on code changes only; the running server will pick them up automatically.

End-to-end testing requirements
-------------------------------
- Write end-to-end tests for any new UI features you implement. E2E tests should be added to the `tests/e2e` directory using Playwright.
- When fixing UI bugs, always add an end-to-end test that validates the fix. This ensures the bug can be caught if it reappears in the future.
- Follow the existing test patterns in `tests/e2e/*.spec.ts` for consistency.
- End-to-end tests help maintain quality and prevent regressions in the UI.

Mock UI Elements
-------------------------------
- When generating mock UI elements for testing or demonstration purposes, ensure they are clearly marked as mock components.
- To mark a UI element as mock, use the `mock_badge` component.
- If most of a page is a mock, use the `mock_banner` component.

API endpoint architecture
-------------------------------
- Always separate business logic from HTTP handling in API endpoints.
- API endpoints should only:
  1. Extract data from the request and request URL
  2. Call a service/library function to perform the actual operation
  3. Construct an API response from the result
- Business logic functions must live in `pkg/util/` packages (e.g., `pkg/util/fileutil/service.go`), NOT in API handler files.
- Use the Params/Result pattern for service functions (similar to gRPC):
  - Define a `*Params` struct containing all input parameters
  - Define a `*Result` struct containing all outputs (not including errors)
  - Example: `DeleteFiles(params DeleteFilesParams) (DeleteFilesResult, error)`
- This pattern ensures:
  - Business logic is testable in isolation without HTTP context
  - Service functions are reusable across different parts of the codebase (API, CLI, background jobs)
  - Clear separation of concerns between HTTP layer and domain logic
- See `pkg/util/fileutil/service.go` and `internal/server/api/v1/files/` for reference implementations.

Git and Version Control
-----------------------
- Use clear, concise commit messages (under 80 characters)
- Keep commits focused and atomic
- We use linear commit history (generally single commit pull requests)
- Sign commits for authenticity
- Create focused pull requests with one change per PR

Common Development Patterns
----------------------------
**When adding a new feature:**
1. Add SQL queries to `sql/queries/` (if database access needed)
2. Create business logic in `pkg/util/<domain>/service.go`
3. Add API endpoint in `internal/server/api/v1/<domain>/`
4. Create UI components using templ in `pkg/ui/`
5. Add E2E tests in `tests/e2e/`

**When fixing a bug:**
1. Write a failing test that reproduces the bug
2. Fix the bug with minimal code changes
3. Verify the test passes
4. Add E2E test if it's a UI bug

**When updating dependencies:**
1. Check for security vulnerabilities first
2. Update one dependency at a time
3. Test thoroughly after each update
4. Document any breaking changes in PR description
