# `AGENTS.md`

## Golang Backend

### Key rule (always)

- Respect the linting and formatting conventions of the various linting and formatting configurations and tools being used.

### Backend development assumptions

- Assume the developer is running the backend via `make watch` and that it will auto-reload on code changes.
- Never run the `make generate` target. Just assume the code is generated automatically as a part of `make watch`.
- Never attempt to start, stop, or restart the backend server yourself.
- Focus on code changes only; the running server will pick them up automatically.

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
  (for example, Cirrus/Photos/Settings), not a back button, unless a page is explicitly a drill-down/detail flow.

### State and async behavior

- Keep async operations cancellable or safely guarded against disposed widgets/controllers.
- Represent loading, success, and error states explicitly in UI flows.
- Handle service errors deterministically and surface user-friendly feedback.

### Testing and validation

- Prefer adding or updating focused tests under `test/` for non-trivial logic changes.
- Use `flutter analyze` and relevant tests to validate changes when possible.
- Keep changes minimal, targeted, and consistent with existing patterns in the repository.

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
