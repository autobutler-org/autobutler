# Sable's AutoButler Dev Context

> This file is gitignored. It's my personal working memory for this repo.
> PR descriptions are written separately and include rationale for reviewers.

## Makefile — Key Commands

| Command | What it does |
|---------|-------------|
| `make watch/backend` | Start backend with auto-reload (uses `air`) |
| `make build/backend` | Build Go binary to `./build/autobutler` |
| `make build/frontend/web` | Build Flutter web → copies to `internal/server/public/` |
| `make test/unit/backend` | Go tests with coverage |
| `make test/unit/frontend` | Flutter tests |
| `make check/go` | go fmt + go vet |
| `make check/flutter` | dart format + flutter analyze |
| `make generate/backend` | sqlc + swag (auto-runs in watch, never call manually) |
| `make setup` | Install all dev tools (air, sqlc, swag, flutter) |
| `make serve/backend` | Run backend without watch |

**Never call `make generate` directly** — it runs automatically via `make watch`.

## Dev Environment Status (this Pi)

- Go: ✅ 1.26.1
- Flutter: ❌ not installed (can still write Dart, can't run/test)
- air: ❌ not installed (needed for `make watch/backend`)
- sqlc: ❌ not installed (needed for DB generation)
- swag: ❌ not installed (needed for swagger docs)

For backend-only work: `go test ./...` works fine. For full dev loop, need `make setup` run.

## Dev Preferences (from Brandon)

- Use **opus** (claude-opus-4-6) model when doing programming work on this repo
- Assign **brandonapol** to every PR
- Can use small volumes of spoofed/test data when needed for Cirrus development

## What is AutoButler

Private cloud device (Raspberry Pi) — Go backend, Flutter mobile app.
Brandon + James (jamesaorson) are the human devs. James uses Copilot heavily.
AGENTS.md in root is the shared agent conventions file — follow it strictly, don't modify it.

## Stack

- **Backend:** Go 1.26, Gin framework, SQLC for DB, `make watch` for dev (auto-reload)
- **Frontend:** Flutter/Dart (mobile-first), no flutter in PATH on this Pi — can't run/test locally
- **DB:** SQL (sqlc.yaml), migrations in `sql/`
- **Infra:** Makefile-driven, `make generate` is auto-run (never call manually)

## Key Patterns

### Backend
- `internal/server/api/v1/<feature>/` — HTTP handlers only (extract, call service, respond)
- `pkg/util/<feature>util/` — business logic with Params/Result pattern
- `pkg/util/serverutil/` — shared HTTP plumbing: `Response`, `WrapApiRoute`, helpers
- Existing response helpers: `Ok()`, `BadRequest(err)`, `Unauthorized(err)`, `InternalServerError(err)`
- `router.go` `WrapApiRoute` is where errors get converted to HTTP responses — **this is where #564 lands**

### Flutter
- `lib/pages/` screens, `lib/widgets/` reusable, `lib/controllers/` state, `lib/services/` API/network
- No overflow in layouts; use scroll containers explicitly
- Hamburger menu pattern for top-level nav, not back buttons

## My Branch Convention

- Branch: `sable/<short-description>` (e.g. `sable/custom-http-errors`)
- PRs opened from `sable-bot`, description includes rationale from DEVLOG
- Never push to main

## Active Work

| Issue | Title | Branch | Status |
|-------|-------|--------|--------|
| #636  | Sable onboarding & context setup | sable/onboarding | PR open |
| #564  | Custom HTTP errors | sable/custom-http-errors | PR #638 open |
| #426  | Auto-update toggle | sable/auto-update-toggle | PR #639 open |

## DEVLOG

### 2026-03-16

First session. Cloned repo, read AGENTS.md (solid, don't touch it), explored structure.
Opened issue #636 for this onboarding work.

Backend pattern is clean — Params/Result everywhere, handlers are thin. 
Issue #564 (custom HTTP errors) is well-specced by James:
- Need a custom error type that carries an HTTP status code
- `WrapApiRoute` in `serverutil/router.go` should detect this type and use its status code
- Otherwise fallback to current behavior (500)
- Already have `BadRequest`, `Unauthorized` etc. as response helpers — the gap is when
  errors bubble up from service layer and the handler just does `InternalServerError(err)`
  but the error itself knows it should be a 404 or 403

Plan: add `HttpError` type to serverutil, update `WrapApiRoute`, update a couple handlers as examples.

### 2026-03-16 (continued)

PR #638 done — HttpError type, WrapApiRoute updated, tests all pass.

PR #639 done — auto-update toggle. Flutter, so couldn't run locally.
Pattern followed: AppSettings for persistence, SwitchListTile in settings_page,
CirrusService.updateToLatest() calls existing backend endpoint POST /api/v1/version/latest,
_maybeAutoUpdate() in main.dart fires on startup, silent/non-blocking.

Next candidates:
- #424 View SBOM on settings page (Flutter + backend — need a /api/v1/sbom or similar endpoint)
- #605 Cirrus toolbar light/dark mode bug (Flutter UI, need to see screenshots more carefully)
- #403 User color scheme choice (Flutter + AppSettings)

Flutter not available on this Pi — can only do backend work unless I read the Dart files carefully
and write changes without running them. Will note this per PR.
