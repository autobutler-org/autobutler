Purpose
-------
These instructions tell GitHub Copilot how to handle styling and CSS when generating or modifying UI, templates, HTML, and static assets for this repository.

Key rule (always)
-----------------
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

Style additions best practices
-----------------------------
- Keep new selectors specific and prefixed if needed to avoid collisions (e.g., .ab- or .site-).
- Add a short comment above any new section in `site.css` describing its purpose and where it's used.
- When changing existing styles, search for usages in `internal/server/ui/` and other templates to avoid regressions.

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
