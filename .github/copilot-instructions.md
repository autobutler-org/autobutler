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

Files
-----
- Canonical stylesheet (source): `internal/server/public/styles/site.css`
- When linking from templates use: `/styles/site.css`

Example (Go template head)
--------------------------
{{`<head>`}}
	{{`<meta charset="utf-8">`}}
	{{`<meta name="viewport" content="width=device-width,initial-scale=1">`}}
	{{`<link rel="stylesheet" href="/styles/site.css">`}}
{{`</head>`}}

