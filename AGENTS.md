## Purpose

These instructions tell GitHub Copilot how to handle programming in this repository.

## Key rule (always)

- Respect the linting and formatting conventions of the various linting and formatting configurations and tools being used.

## Project type

- This repository is a Flutter mobile app (Dart), with platform folders under `android/` and `ios/` and app code under `lib/`.
- Prefer Dart/Flutter implementations for app logic. Do not introduce web-only patterns or frameworks unless explicitly requested.

## Current app structure (follow this)

- `lib/main.dart`: app entrypoint and root wiring.
- `lib/pages/`: top-level screens.
- `lib/widgets/`: reusable UI components.
- `lib/controllers/`: UI-facing coordination/state orchestration.
- `lib/services/`: API/network and external integration logic.
- `lib/models/`: typed data models.
- `lib/utils/`: small, focused helpers.

## Code organization rules

- Keep business logic out of widgets when possible; widgets should mostly render UI and dispatch actions.
- Put network/data-source concerns in `lib/services/`, not in pages/widgets.
- Put pure mapping/parsing/domain helpers in `lib/utils/` or `lib/models/` as appropriate.
- Keep files focused and avoid large, mixed-responsibility classes.

## Flutter UI/layout principles

- Avoid page-level overflow; design layouts so content fits naturally on mobile screens.
- When content can exceed available height, use explicit scroll containers (e.g., `ListView`, `SingleChildScrollView`, `CustomScrollView`) rather than accidental overflow.
- In `Column`/`Row` layouts, use `Expanded`/`Flexible` correctly so children receive bounded constraints.
- Respect safe areas and platform insets (`SafeArea`, keyboard insets) for production UI.

## State and async behavior

- Keep async operations cancellable or safely guarded against disposed widgets/controllers.
- Represent loading, success, and error states explicitly in UI flows.
- Handle service errors deterministically and surface user-friendly feedback.

## Testing and validation

- Prefer adding or updating focused tests under `test/` for non-trivial logic changes.
- Use `flutter analyze` and relevant tests to validate changes when possible.
- Keep changes minimal, targeted, and consistent with existing patterns in the repository.

## Platform and generated code

- Do not manually edit generated artifacts or build outputs (for example under `build/`, `ios/Flutter/ephemeral/`, or generated plugin registrants) unless explicitly required.
- Scope manual edits primarily to source code in `lib/`, tests in `test/`, and intentional platform configuration files.
