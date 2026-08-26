# `AGENTS.md`

## Golang Backend

### Key rule (always)

- Respect the linting and formatting conventions of the various linting and formatting configurations and tools being used.

### Minimizing Database Usage

- Add or modify DB tables only as a last resort.
- Prefer native file edits or other non-database mechanisms whenever possible.
- Only use the database for data that genuinely cannot be managed reliably on disk or in files.

### Backend development assumptions

- Assume the developer is running the backend via `make watch` and that it will auto-reload on code changes.
- Never run the `make generate` target. Just assume the code is generated automatically as a part of `make watch`.
- Never attempt to start, stop, or restart the backend server yourself.
- Focus on code changes only; the running server will pick them up automatically.

### Use Makefile Targets

- Agents should use existing Makefile targets to `run`, `test`, and `lint` the codebase rather than crafting their own shell
  commands.
- Do **not** run ad hoc commands for these standard flows — use `make test`, `make lint`, etc.
- If an action needs to be templatized for general usage, add a new Makefile target for it rather than running raw commands.

### API endpoint architecture

- Always separate business logic from HTTP handling in API endpoints.
- API endpoints should only:
  1. Extract data from the request and request URL
  2. Call a service/library function to perform the actual operation
  3. Construct an API response from the result
- Business logic functions must live in `pkg/util/` packages, NOT in API handler files.
- Use the Params/Result pattern for service functions (similar to gRPC):
  - Define a `*Params` struct containing all input parameters
  - Define a `*Result` struct containing all outputs (not including errors)
  - Example: `DeleteFiles(params DeleteFilesParams) (DeleteFilesResult, error)`
- This pattern ensures:
  - Business logic is testable in isolation without HTTP context
  - Service functions are reusable across different parts of the codebase (API, CLI, background jobs)
  - Clear separation of concerns between HTTP layer and domain logic

## Flutter Development

### Purpose

These instructions tell GitHub Copilot how to handle programming in this repository.

### Key rule (always)

- Respect the linting and formatting conventions of the various linting and formatting configurations and tools being used.

### Project type

- This repository is a Flutter mobile app (Dart), with platform folders under `android/` and `ios/` and app code under `lib/`.
- Prefer Dart/Flutter implementations for app logic. Do not introduce web-only patterns or frameworks unless explicitly
  requested.

### Packages Directory

- `packages/` contains fully independent Dart or Flutter libraries maintained alongside Quark.
- These packages are kept generic and are **not** monolithic to the Quark codebase — they are intended for both internal
  use and eventual public publication.
- Other Flutter developers can adopt these packages independently; keep them decoupled from app-specific logic to maximize
  reusability.

### Current app structure (follow this)

- `lib/main.dart`: app entrypoint and root wiring.
- `lib/pages/`: top-level screens.
- `lib/widgets/`: reusable UI components.
- `lib/controllers/`: UI-facing coordination/state orchestration.
- `lib/services/`: API/network and external integration logic.
- `lib/models/`: typed data models.
- `lib/utils/`: small, focused helpers.

### Code organization rules

- Keep business logic out of widgets when possible; widgets should mostly render UI and dispatch actions.
- Put network/data-source concerns in `lib/services/`, not in pages/widgets.
- Put pure mapping/parsing/domain helpers in `lib/utils/` or `lib/models/` as appropriate.
- Keep files focused and avoid large, mixed-responsibility classes.
- For obvious global tuning/configuration values (for example thresholds,
  debounce durations, and UI behavior constants), prefer a dedicated static
  const config class under `lib/utils/` and reference it from feature code
  instead of scattering hardcoded literals.

### Flutter UI/layout principles

- Avoid page-level overflow; design layouts so content fits naturally on mobile screens.
- When content can exceed available height, use explicit scroll containers
  (e.g., `ListView`, `SingleChildScrollView`, `CustomScrollView`) rather than accidental overflow.
- In `Column`/`Row` layouts, use `Expanded`/`Flexible` correctly so children receive bounded constraints.
- Respect safe areas and platform insets (`SafeArea`, keyboard insets) for production UI.
- Keep top-level page navigation consistent: pages should use a hamburger menu in the app bar/drawer pattern by default
  (for example, Files/Photos/Settings), not a back button, unless a page is explicitly a drill-down/detail flow.

### Custom Widget Guidelines (spreadsheet editor)

- Prefer small, focused sub-widgets: extract visibly independent parts (cells, rows, editors) into private widgets to keep
  files readable and testable.
- Keep business logic out of widgets: use `ChangeNotifier`/controller objects (for example `SheetController`) or services
  for model updates and side effects.
- Minimize rebuilds: for large, scrollable datasets prefer lazy vertical builders (`ListView.builder`) and limit rebuild
  scope with `ValueListenableBuilder` or per-row `ValueNotifier`s rather than rebuilding the whole grid.
- Single editing controller: share a single `TextEditingController` for inline editing to avoid lifecycle complexity and
  state duplication; manage which cell is active in the view state.
- Use stable keys: attach stable `Key`s (for example `ValueKey('r${r}c$c')`) to cells/rows so Flutter preserves state during
  list changes.
- Column sizing: use `Expanded`/`Flexible` with per-column `flex` (configurable via the controller) to keep column widths
  aligned across rows and support runtime updates.
- Performance guards: consider `RepaintBoundary`, `AutomaticKeepAliveClientMixin` / `KeepAlive` for focused editors or heavy
  cells to avoid unnecessary repaints or losing focus, and avoid expensive work in `build()`.
- Horizontal scale: for extremely wide sheets consider horizontal virtualization (lazy cell builders) or a custom two-dimensional
  viewport rather than creating all cells eagerly.
- File organization: when splitting large widgets, place each public class in its own file, update imports/exports, and
  keep private helper widgets near the public API that uses them.
- Constructor patterns: public widgets should accept a named `Key? key` (use `super.key`), prefer `super` parameter forwarding,
  and avoid adding unused `key` parameters to private/internal widgets.
- Validation: after refactor run the analyzer (`flutter analyze` / `make check`), update or add focused unit/widget tests,
  and update example/demo imports and documentation.

### State and async behavior

- Keep async operations cancellable or safely guarded against disposed widgets/controllers.
- Represent loading, success, and error states explicitly in UI flows.
- Handle service errors deterministically and surface user-friendly feedback.

### Refresh pattern (always follow this)

- **All pages with a refresh action must use `AutoRefreshMixin`** (`lib/utils/auto_refresh_mixin.dart`).
  - Add `with WidgetsBindingObserver, AutoRefreshMixin` to the `State` class.
  - Implement `Future<void> refresh()` with the data-fetching logic.
  - Do NOT override `initState` for initial data loads — the mixin calls `refresh()` on startup automatically.
  - Use `manualRefresh()` for button/pull-to-refresh wiring.
- **All refresh buttons must use `RefreshIconButton`** (`lib/widgets/refresh_icon_button.dart`).
  - Pass `isRefreshing: isRefreshing` (from the mixin) and `onPressed: manualRefresh`.
  - Do NOT use raw `IconButton(icon: Icon(Icons.refresh))` for refresh actions.
- **Loading state must distinguish initial load from subsequent refreshes:**
  - Show a full-screen spinner only when `isInitialLoad == true` (no data yet).
  - While refreshing with existing data, keep current content visible — do not replace it with a spinner.
  - For `FutureBuilder`-based pages, pass `initialData: _cachedData` to preserve stale content during refresh.

### Navigation and routing (always follow this)

- The app uses `go_router` with `PathUrlStrategy` — clean URLs (`/files`, `/photos`, no `#`).
- All routes are declared in `lib/router.dart`. Route path strings are constants on `AppRoutes`.
- **When adding a new top-level page, you must:**
  1. Add a `static const` path to `AppRoutes` in `lib/router.dart`
  2. Add a `GoRoute` entry to the `router` in `lib/router.dart`
  3. Use `context.go(AppRoutes.yourRoute)` for navigation (not `Navigator.pushReplacement`)
  4. Use `context.push(AppRoutes.yourRoute)` for drill-down/detail flows that should be back-stackable
- Do NOT use `Navigator.pushReplacement` or `Navigator.of(context).push` for top-level page changes — use `context.go`.
- `Navigator.push` / `Navigator.pop` is still acceptable for modal dialogs and overlays (image/video viewers, confirmation
  dialogs).
- If a new page requires auth gating, add the path to the `publicRoutes` set in `_authRedirect` in `lib/router.dart` if
  it should be accessible without login, or do nothing if it should be protected.

### Testing and validation

- Prefer adding or updating focused tests under `test/` for non-trivial logic changes.
- Use `flutter analyze` and relevant tests to validate changes when possible.
- Keep changes minimal, targeted, and consistent with existing patterns in the repository.

### Pull request and commit conventions (always follow this)

- **PR titles must use conventional commits format:** `type: description`
  - `feat:` — new feature
  - `fix:` — bug fix
  - `chore:` — maintenance, tooling, config
  - `refactor:` — code change with no behavior change
  - `docs:` — documentation only
  - `test:` — adding or fixing tests
  - `perf:` — performance improvement
- The description should be lowercase, imperative mood: `fix: add null check` not `Fix: Added null check`
- Include the issue number in the PR body (`Closes #N`), not the title
- Branch names should reflect the issue: `fix/123-short-description`, `feat/456-short-description`

### Platform and generated code

- Do not manually edit generated artifacts or build outputs (for example under `build/`, `ios/Flutter/ephemeral/`, or
  generated plugin registrants) unless explicitly required.
- Scope manual edits primarily to source code in `lib/`, tests in `test/`, and intentional platform configuration files.

## Relynce

This project uses Relynce for reliability risk analysis. The following skills are available:

### Risk Detection

- `/rely:detect-risks` — Scan code for reliability risks and submit findings
- `/rely:risk-guidance` — Get detailed guidance for a specific risk

### Risk Remediation

- `/rely:remediate-risks` — Auto-implement fixes for detected risks

### Quick Reference

- Run `rely risk list` to see current risks
- Run `rely risk show <code>` for risk details with mapped controls
- Run `rely control show <code>` for control implementation guidance

### Risk Detection

- `/rely:detect-risks` — Scan code for reliability risks and submit findings
- `/rely:risk-guidance` — Get detailed guidance for a specific risk

### Risk Remediation

- `/rely:remediate-risks` — Auto-implement fixes for detected risks

### Quick Reference

- Run `rely risk list` to see current risks
- Run `rely risk show <code>` for risk details with mapped controls
- Run `rely control show <code>` for control implementation guidance
