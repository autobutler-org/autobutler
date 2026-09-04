# `AGENTS.md`

## Golang Backend

### Key rule (always)

- Respect the linting and formatting conventions of the various linting and formatting configurations and tools being used.

### Streaming and memory (always)

The service runs in low-memory environments and file sizes are user-controlled
and unbounded. A 4 GiB archive must cost megabytes of heap, not gigabytes.

- Thread `io.Reader` / `io.Writer` through. Never materialize a file, an upload,
  a download, or an archive entry as a `[]byte` or a `string`.
- On any path that can carry file content, treat these as defects: `io.ReadAll`,
  `os.ReadFile`, `ioutil.ReadAll`, `c.GetRawData()`, and a `bytes.Buffer` that
  accumulates a whole body before it is written anywhere.
- Move bytes with `io.Copy` / `io.CopyBuffer`. Serve them with
  `http.ServeContent` or `c.DataFromReader` rather than buffering and then
  writing.
- Need random access (zip readers, image decoding, HTTP range requests)? Take
  the `io.ReaderAt` / `io.ReadSeeker` the source already offers and size it from
  `Stat`. Do not buffer a stream to make it seekable. `vfs.VFS.Open` returns an
  `*os.File` for the local and storage-service namespaces, so both satisfy this.
- Ownership travels with the reader: if a returned reader streams out of a file,
  the returned `Close` has to close that file.
- Bound what genuinely cannot be streamed. Wrap it in `io.LimitReader` and
  check whether the limit was reached — never trust a size a caller declared.
- Reading something whole is fine when its size is bounded by us, not by a
  user: config files, JSON request bodies, a generated thumbnail. The rule is
  about anything a user can make arbitrarily large.

See #1705: `io.ReadAll` on a 4 GiB zip asked for ~12 GiB of heap and returned a
500. Streaming the same archive costs 17 MiB.

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

### API package layout

Every handler package under `internal/server/api/v0/` has the same shape, so a route can be found from
its URL without grepping.

- **Directory name matches the URL path segment.** `/api/v0/version/*` lives in `version/`, `/api/v0/albums/*`
  in `albums/`. The segment wins over grammar — no singular/plural normalization, because renaming a
  directory to read better would mean renaming the route.
- **`<pkg>.go` is the interface file**, named after its own directory (`albums/albums.go`), and it is strictly
  public: exported types and functions only, `NewRouter()` among them. Nothing private goes in it — not the
  `router` struct, not a request DTO, not a helper. A reader who opens `albums/albums.go` should see the
  package's whole exported surface and nothing they are not allowed to call.
- **`types.go` holds every private type**, the `router` struct included. A method belongs to its type, so
  `Routes()` — a method on the private `router` — moves into `types.go` with it. Exported types may live here
  too when `<pkg>.go` would otherwise sprawl.
- **`helpers.go` holds every other private function**, whatever it is shared with. Two helpers files split by
  topic (`upload_session_helpers.go` next to `helpers.go`) is the layout this convention replaced.
- **One handler per file, named `verb_noun.go`** after what the handler does: `list_albums.go`,
  `delete_device.go`, `rename_device.go`. Never a file named after the resource alone (`albums.go` holding all
  five CRUD routes) — that is the layout this convention replaced. **The handler itself is the one private
  function exempt from `helpers.go`**: it stays in its own file along with the
  `var xxxRoute = serverutil.ApiRoute(...)` that registers it, because splitting a route from its handler is
  what makes a route hard to find. Anything else private in that file — a DTO, a shared subroutine — moves out
  to `types.go` or `helpers.go`.
- A private const or var used across files goes to `types.go` or `helpers.go`, whichever fits; one used by a
  single handler stays in that handler's file.
- Neither `types.go` nor `helpers.go` may hold a route.

### Go linting

`make check/lint/go` is the whole Go check, and CI runs the same target — nothing is advisory.

- **golangci-lint** against `.golangci.yml`: `govet`, `errcheck`, `ineffassign`, `staticcheck` and `unused`,
  plus `revive`. Install it with `make setup/golangci-lint`. staticcheck runs only as a golangci-lint linter;
  there is no second standalone copy to disagree with it.
- **`scripts/check-go-structure.bash`** enforces the layout rules above, which no general-purpose linter can
  see: every package under `pkg/` and every router package under `internal/server/api/` has its `<pkg>.go`
  interface file and declares nothing private in it, no `v<N>_` filename prefix disagrees with the version
  directory it sits in, and no handler package imports the low-level packages (`os/exec`, `syscall`,
  `golang.org/x/sys/unix`, database drivers) that belong in `pkg/util/` or `internal/db/`.
- **The interface file is public in `pkg/` too, not just under `internal/server/api/`.** A package under
  `pkg/` puts its exported types and functions in `<pkg>.go` and keeps every private one out: private types
  go to that package's `types.go`, private functions to its `helpers.go`. Unlike a handler package, `pkg/`
  does not consolidate — a private helper already sitting beside the topical code it serves (`storageutil/
  partition_linux.go`, `photoutil/rotate.go`) stays there. The rule is only that `<pkg>.go` stays public, so
  reading it tells you the whole API and nothing you cannot call. Private consts and vars are not checked,
  and a tuning value next to the exported thing it tunes is fine where it is.
- `.golangci.yml` is a ratchet: every rule enabled there is at zero violations, and the rules still switched
  off name the sweep they are waiting on. Turning one on means fixing the code in that same PR — do not add a
  `//nolint` to get a build green, and do not disable a rule to avoid a fix.

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

### Error text (always follow this)

- **Every user-facing error string comes from `Errors` in `lib/utils/error_text.dart`.** Never write error copy inline,
  and never put a thrown object into text a user reads — `'Save failed: $e'` renders exception class names, OS errno
  values and full request URIs into the UI (#1622).
- In a page or widget, call `Errors.message(error, '<action>')`. The action is a bare verb phrase that reads after
  "Couldn't": `'save the file'`, `'load your photos'`, `'delete the album'` — lowercase, no trailing period.
- In a service, throw a type `Errors` can read:
  - `ApiException(statusCode, contextForLogs)` — the Quark answered with a non-success status.
  - `throwApiError(statusCode, body?['error'], contextForLogs)` — same, but the Quark may have sent its own message.
  - `MessageException('...')` — only for copy written for a user to read, like "Invalid username or password."
  - plain `Exception('...')` — diagnostics only. It never reaches the UI; the user sees the generic sentence instead.
- **To extend, add a `static` method or a status branch to `Errors`** — never a new string at the call site. Statuses the
  backend does not return do not need a branch; the fallback is a true sentence, and a guess is not.
- This is also the groundwork for localization: these strings are the app's whole user-facing error vocabulary, so
  translating means swapping the bodies in one file, not hunting interpolations across twenty pages.
- `test/utils/error_text_test.dart` enforces the rule — it fails the build on any `Text('...$e')` left in `lib/`.

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

### Widget package rules (`packages/quark_widgets`, always follow this)

Every reusable visual component lives in `packages/quark_widgets`. Pages, controllers, and services stay in the app.
The package is a separate pub package with no dependency on the app, so it cannot import `lib/services`,
`lib/router.dart`, `AppSettings`, `http`, or platform channels. Keep it that way: never add those to the package pubspec.

- **Data in, callbacks out.** A widget's constructor takes immutable values (`final` fields, `const` constructor where
  possible) and handlers (`VoidCallback`, `ValueChanged<T>`). It never fetches, never reads global state, never
  navigates. The parent decides what happens on an action.
- **Domain state lives in the caller.** Selection, expansion, the chosen album, favorites, sort order, loading and error
  flags are all passed in and changed by calling back out (`selectedIds` in, `onSelect` out). `StatefulWidget` is
  allowed only for state Flutter forces on you: `AnimationController`, `FocusNode`, `TextEditingController`,
  `ScrollController`, hover and pressed visuals. Even then, expose the outcome through a callback if a caller could care.
- **Loading, empty, and error are explicit inputs.** A widget that lists things takes `isLoading` and `error` (or a
  sealed state) and renders each case. It never decides when to load.
- **Anything that needs the network is a builder.** `thumbnailBuilder(BuildContext, PhotoItem)` and friends. The app
  supplies `CachedNetworkImage`; the gallery supplies a placeholder.
- **The package owns its value types.** Small immutable classes in `lib/src/models/` (`AlbumItem`, `PhotoItem`, ...).
  Controllers in the app map `lib/models` into them. The package never imports app models.
- **Keys for Flutter Probe.** Every widget takes `super.key`. Every tappable part gets a deterministic `ValueKey` built
  from a documented prefix and the item id, for example `ValueKey('album_tile_$id')`. The class doc lists its prefixes
  so a `.probe` script can write `tap #album_tile_vacation`.
- **Layout survives narrow viewports.** No `Expanded` or `Flexible` inside slivers or other unbounded parents (#1599).
  Every test file has a narrow (360x640) and a wide (1280x800) case.
- **Each widget ships as a set.** One file in `lib/src/<group>/` exported from the barrel; `///` docs on the class
  (purpose, key prefixes, one usage snippet), every constructor parameter, and every callback; one test file
  `test/<group>/<name>_test.dart`; one entry in `examples/widget_gallery/lib/registry.dart` with fake data and callbacks
  that log to the gallery's event panel. A gallery test fails when a barrel export has no registry entry.
- **One widget per file, no private widgets.** Every widget class is public and lives in a file named after it, even
  when it has one caller. No `class _Part extends StatelessWidget` inside another widget's file, and no
  `Widget _buildPart()` method that returns a subtree. In the package, a part of one parent lives in
  `lib/src/<group>/<parent>/<part>.dart`, imported by the parent and not exported from the barrel until something else
  needs it; a part is tested through its parent unless it has states or callbacks of its own. In the app, a page's
  parts live under `lib/widgets/<page>/`. Long files hide reusable pieces; a directory of small files shows what can be
  promoted.
- **Theme through tokens.** Colors, radii, and spacing come from `QuarkTokens` reached through the theme, never a
  hardcoded color in a widget. The gallery's theme panel edits the tokens live, so a hardcoded value is a bug you can see.
- **Error copy still comes from the app.** The package never composes a user-facing error sentence. It takes
  `String? error` and renders it; the page builds it with `Errors.message`.

In the app, a page is a thin `StatefulWidget` that owns a controller (a `ChangeNotifier` in `lib/controllers/`) and
rebuilds with `ListenableBuilder`. Every service call goes through the controller. The page maps controller state into
package widgets and wires callbacks back to controller methods. Navigation stays in the page. Controllers take their
service calls as injectable function parameters with defaults pointing at the real static methods, so a controller test
passes fakes without a mocking library.

**Pages are compositions.** A page's `build` is package widgets arranged with plain Flutter layout (`Column`, `Row`,
`Expanded`, `Padding`, `SafeArea`) and nothing else. No sizing math, no `MediaQuery` breakpoints, no `LayoutBuilder`
branching, no private `_build*` methods and no private widget classes: every named subtree is its own file. Responsive behavior lives in package layout
widgets (a page scaffold, a split view that collapses its sidebar under a breakpoint, a section, a toolbar) so every page
shares the same breakpoints and a layout fix lands once. If a page needs a layout the package lacks, add the layout
widget to the package first, then compose. The test of a good page is that a new one is mostly a list of package
widgets and reads in one screen.

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
