# Codebase Organization Review

> **What this is:** A one-time architecture audit, run at a contributor's request on 2026-08-28,
> looking across `lib/` (Flutter), `cmd/`/`internal/`/`pkg/` (Go backend), and the supporting
> `packages/`/`scripts/`/CI for **AI drift** — places where a long run of individually reasonable,
> individually reviewed PRs added up to something sloppier than any one of them looks on its own.
> It's a snapshot and a set of observations, not a roadmap, a spec, or anyone's action item list.
> Every specific claim below was checked directly against the code at the time this was written;
> treat it as a starting point for discussion and expect parts of it to drift out of date as the
> codebase keeps moving.

Nothing here failed review on its own merits — the point is what the sum looks like. Findings are
grouped by theme, each with file references so they're checkable rather than taken on faith. A few
things are called out as done *well*, on purpose — the drift isn't uniform, and the good examples are
the template for fixing the bad ones.

## The recurring shape of the problem

Almost every finding below reduces to the same root cause: **a good pattern gets introduced once,
for one PR's problem, and is never rolled out to the rest of the codebase that has the identical
problem.** A shared helper exists, but three other call sites hand-roll their own version instead of
finding it. A rename touches most of the codebase, but takes two or three follow-up hotfix commits
over the following weeks to find the stragglers. A widget gets built and documented as the answer to
"empty/disconnected state," and the next three pages that need exactly that build their own instead.
No single commit is wrong. Nobody is looking at the aggregate.

## Go backend

### Renames don't finish in one pass

The AutoButler → Quark rename is still live in the tree in more than one place:

- `cmd/provisioning/main.go:138` still hardcodes the Headscale tenant username as `"autobutler"`,
  while `pkg/util/storageutil/dir.go:21` checks the local Unix service account against `"quark"`.
  These are genuinely different namespaces — a coordination-server tenant vs. a system user — so
  this is not a single identity disagreeing with itself, and the provisioning string cannot simply
  be flipped without breaking already-provisioned devices. It is still visible rename residue: the
  name a new deployment gets on the coordination server is the old product name.
- `pkg/util/updateutil/types.go:11-30` documents a real bug the rename caused: a
  `DefaultUpdateSources` entry pointed at `autobutler-org/quark.org`, a repository that never
  existed. It's fixed now, with a regression test (`sources_test.go:54`) — but it's the second
  documented cleanup pass on this rename, after a separate "Change remnant butler strings to quark"
  commit landed the day before.

Contrast this with the **Cirrus → Files rename** (#1601), which is the model to copy:
`internal/server/api/v0/files/legacy_cirrus_alias.go` and
`pkg/util/storageutil/legacy_cirrus_dir.go` both carry explicit `TODO(pre-v1.0.0, #1601)` markers,
deprecation headers, and idempotent migration logic. The equivalent Flutter-side rename is just as
clean — see below. Renames stop leaking when the deprecated path is written down as code with a
tracking issue, not when it's just deleted and hoped for.

### The same safety check, reimplemented four times

`storageutil.SafeJoin` (`pkg/util/storageutil/storageutil.go:633-646`) exists specifically to guard
against path traversal. It is not used by:

- `internal/server/api/v0/videos/trim.go:89-93`
- `internal/server/api/v0/videos/extract_frame.go:83-85`
- `internal/server/api/v0/videos/get_metadata.go:94`
- `internal/server/api/v0/photos/get_metadata.go:142`
- `pkg/migration/extractor.go:96-97` (a fifth, slightly different variant)

Each hand-rolls the same `filepath.Clean` + `filepath.Join` +
`strings.HasPrefix(fullPath, cleanFilesDir+separator)` guard. Today they're all correct. The risk
isn't any one of them — it's that the *next* endpoint added under time pressure copies one of these
instead of finding `SafeJoin`, and drops the separator suffix or the clean step. A guard against path
traversal is exactly the kind of logic that should only exist in one place.

### An orphaned 8MB binary

`testplugin` at the repo root is a checked-in, tracked ELF binary (confirm with `git ls-files |
grep testplugin`), despite being listed in `.gitignore:126` since #1515. There is no `cmd/testplugin`
or matching plugin source anywhere in this tree — the plugin subprocess host and marketplace feature
that would produce this binary live only on an unmerged branch. Whatever built and committed this
file, it's now dead weight with nothing in the current tree that explains or rebuilds it. Worth
deleting outright, and worth checking whether `.gitignore` entries for build artifacts are actually
being respected by CI/tooling that produces them.

### Two logging conventions, coexisting indefinitely

`log/slog` (structured logging) has been adopted in 6 files, all recently touched
(`middleware.go`, `routes.go`, `healthutil.go`, `legacy_cirrus_dir.go`). The stdlib `log` package is
still used in 22 others, including core files like `internal/server/server.go` and
`pkg/util/workerutil/workerutil.go`. This reads like a migration that's happening file-by-file, as a
side effect of whichever file a given PR happened to touch, rather than as a tracked migration with
an end state. Left alone, both styles will coexist forever — new code has no signal for which one is
"current."

### Migration numbers keep colliding

`internal/db/migrations/` has gaps in its numbering (013 jumps to 016; 014/015/017/018 are missing
from the current sequence) because at least three migrations have needed manual renumbering after the
fact ("renumber video jobs migration 013 -> 024", "renumber share links migration 018 -> 023", and a
third). This isn't a one-off mistake, it's structural: parallel branches each claim the "next"
available number off a mainline that's moved by the time they merge. A timestamp-prefixed migration
naming scheme (e.g. `20260827_143000_description`) removes this entire class of merge conflict
without asking anyone to coordinate.

### API versioning that isn't really versioning

Eighteen feature packages live under `internal/server/api/v0/` (`files`, `photos`, `vault`, `videos`,
`albums`, `books`, …). Exactly one — `internal/server/api/v1/vfs` — is namespaced `v1`. That's not a
versioned API with a v0→v1 migration in progress; it's one feature that happened to get a different
prefix. Either commit to `v1` being the new baseline for what gets built next, or drop the versioning
pretense until there's an actual breaking change to version around.

### Config reads scattered, with inconsistent naming

There's no central config struct. Environment variables are read via bare `os.Getenv` in six
non-test files (eight counting tests), with no shared prefix convention: `HEADSCALE_URL`,
`PROVISIONING_SECRET`, `PORT`, `HTTPS_PORT` are unprefixed; `QUARK_HEADSCALE_URL`, `QUARK_PROVISIONING_SECRET`,
`QUARK_PROVISIONING_URL`, `QUARK_INSECURE` are `QUARK_`-prefixed — for overlapping concepts (compare
`HEADSCALE_URL` in `cmd/provisioning/main.go` against `QUARK_HEADSCALE_URL` in
`pkg/util/remoteutil/remoteutil.go:34`). `.env.example` documents 5 variables; the codebase actually
reads 15.

### A stale coverage exclusion list

`scripts/coverage-excluded-packages.txt` still lists `quark/pkg/ui/components`, `quark/pkg/ui/types`,
and `quark/pkg/ui/views` — packages that don't exist in this repository anymore. They were removed
by #482 when the project switched its server-rendered UI to Vue (2025-12-21), about ten weeks before
the Flutter app was merged into the monorepo (#622, 2026-03-05). Three of eleven entries (27%) in
a file whose entire job is "stay accurate" are dead. Nobody's job is to notice when an excluded
package is deleted.

## Flutter frontend

### One controller, in the one file that needed it least

`lib/controllers/file_browser_controller.dart` (plus `file_browser_cache.dart`) is the *only*
controller in the app, backing `file_browser_page.dart`. All 23 other pages — including several
larger and more stateful than the file browser — call services directly from `State` classes.
`settings_page.dart` alone imports 10 services directly (`app_settings`, `auth_service`,
`connected_devices_service`, `files_service`, `health_service`, `remote_access_service`,
`sbom_service`, `settings_service`, `smb_service`, `storage_service`). The extraction happened once,
for the file browser's own problem, and was never generalized into "this is how a complex page talks
to services here." A new contributor has no way to know which pattern is "correct," and this repo's
own `AGENTS.md` describes `lib/controllers/` as "UI-facing coordination/state orchestration" without
saying when a page should have one.

### The same small widget, pasted three times

An identical private `_ErrorBanner` class — same `Semantics` wrapping, same
`QuarkIcons.error_outline`, same `errorContainer` styling, ~30 lines — is defined independently in:

- `lib/pages/login_page.dart:200`
- `lib/pages/recover_page.dart:230`
- `lib/pages/setup_page.dart:608`

None of the three live in `lib/widgets/`, even though widget extraction is otherwise a followed
convention elsewhere in the app (see below). Each PR that needed an error banner added its own
instead of finding the other two — because from inside any single PR's diff, adding one small private
widget looks completely reasonable.

### Three different answers to "you're not connected"

This app already has three separate implementations of the same underlying idea:

- `settings_page.dart:581` and `:1044` both hardcode the literal string `'Not connected — add your
  Quark address below'` as inline `Text` widgets.
- `vault_page.dart:305`'s `_buildDeviceDisconnectedView()` builds a structurally different view for
  "the vault device is unreachable."
- `lib/widgets/core/empty_state_widget.dart` (`EmptyStateWidget`) exists, is documented in
  `lib/widgets/README.md` with a "Connect to your Quark" example, and is *already* the intended
  answer to exactly this case — and is used consistently elsewhere (`photos_page.dart`,
  `file_browser_page.dart`, `album_page.dart`).

So the shared component that should own this state exists, is documented, and is actively used for
other empty states — it just wasn't reached for on these three call sites. (Also relevant context:
issue #1637, filed this week, asks for a fourth "not connected" state — for when the app can't reach
a *configured* Quark at all, as opposed to any of these three. Worth building that on
`EmptyStateWidget` rather than adding a fourth bespoke implementation.)

### Renames done right: the counter-example

Unlike the backend's AutoButler/Quark residue, the Flutter side's Cirrus → Files rename (#1601) is
fully clean. The only surviving `Cirrus` references are in `lib/router.dart:25-202`, and they're
deliberate, commented legacy-alias code (`legacyCirrus`, `/cirrus` redirect) tagged
`TODO(pre-v1.0.0, #1601)`. The multiple hotfix commits visible in git history for this rename did
their job — nothing was left half-renamed. This is what "finished" looks like; the backend rename
above is what "still leaking two cleanup passes later" looks like.

### Pages that became dumping grounds

Seven pages account for 10,400+ lines: `file_browser_page.dart` (2,350 lines), `settings_page.dart`
(1,832), `photos_page.dart` (1,491), `image_viewer_page.dart` (1,328), `document_editor_page.dart`
(1,176), `vault_page.dart` (1,168), `video_viewer_page.dart` (1,067). `settings_page.dart` in
particular mixes host management, SMB config, SBOM/version info, remote access, and help/support UI
in one file with no internal sectioning. Each addition to these files was a reasonable, scoped PR;
the file just never got split as it grew, because splitting a file is never itself the task at hand.

### Test coverage tracks "what shipped most recently," not "what's risky"

`upload_manager.dart`, `resumable_upload_service.dart`, `content_search_service.dart`, and
`app_settings.dart` are all well-tested. Sizable, stateful features that have no dedicated tests:
vault/encryption (`vault_page.dart` + `vault_service.dart`, 1,682 lines combined), video playback
controls (`video_viewer_page.dart`, 1,067 lines), and SMB + remote access (`smb_service.dart` +
`remote_access_service.dart`, 148 lines, both reached only from `settings_page.dart`). Note that
`storage_devices_page.dart` (783 lines) is *not* in that last group — it imports neither service,
and `StorageDevice` itself is covered by `test/services/storage_device_test.dart`. This is the
classic shape of iteratively-built test coverage: whichever feature just got built gets a matching
test file, because that's the scope of the PR that built it; adjacent, older code that never had
that moment gets skipped indefinitely.

## Cross-cutting

- **`app/` is an empty directory** (just a `.gitignore`) sitting at the repo root with no
  explanation.
- **`packages/quark_formula`'s main library file is `lib/ab_formula.dart`** — "ab" for AutoButler,
  inside a package that's otherwise fully renamed to `quark_formula`. Small, but the same rename
  residue pattern as the backend findings above, in a third location.

## What to actually do about it

None of this needs a rewrite. It needs a handful of cheap, mechanical passes and a couple of standing
decisions:

1. **Delete `testplugin`**, fix `scripts/coverage-excluded-packages.txt`, rename `ab_formula.dart`,
   delete `app/` (or document what it's reserved for).
2. **Route the four video/photo path-traversal call sites through `storageutil.SafeJoin`.** This is
   the one finding with real security weight, not just tidiness.
3. **Extract `_ErrorBanner` into `lib/widgets/`, and rebuild the three "not connected" states on
   `EmptyStateWidget`** — including the new one from #1637, so it's a fourth *use* of the shared
   widget rather than a fourth bespoke copy.
4. **Pick one config convention (`QUARK_`-prefixed) and one logging convention (`log/slog`)** and
   apply either a lint rule or a tracked sweep issue — "migrate opportunistically" is how both got
   split in the first place.
5. **Decide, explicitly, whether the controller pattern is the standard for complex pages.** If yes,
   write it down in `AGENTS.md` and apply it to `settings_page.dart`/`photos_page.dart` next. If no,
   delete the one that exists rather than leaving it as an unexplained exception.
6. **Switch new migrations to timestamp-prefixed names** to stop the renumbering churn at the root.
7. **Decide what to do about the `autobutler` string in `cmd/provisioning/main.go`** — but note
   this one is *not* cheap cleanup. It is a Headscale tenant username on the coordination server,
   not a local identity, so flipping it breaks every device already provisioned under
   `autobutler`. If it changes at all it needs a dual-read/backfill migration, not the deprecation
   shim that worked for Cirrus → Files. The `"quark"` check in `pkg/util/storageutil/dir.go` is a
   local Unix service account and is unrelated; leaving them different is defensible.

The through-line for all seven: whenever a fix or a rename touches more than one call site, leave a
`TODO(#issue)` at every site it didn't finish, the way #1601 did — that's the difference between a
rename that resolves itself and one that needs a second archaeology pass six weeks later.
