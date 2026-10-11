# Dev Onboarding

Get Quark running locally in about 10 minutes.

## Prerequisites

You need: Go, Flutter, Make, [air](https://github.com/air-verse/air), sqlc, swag.

The Makefile installs most of these for you:

```bash
make setup
```

This runs `setup/gotools`, `setup/air`, `setup/sqlc`, `setup/swag`, and `setup/flutter` in one shot. If you only
need the backend (no Flutter), run `make setup/gotools setup/air setup/sqlc setup/swag` instead.

### Shared agent skills

Skills that are not specific to Quark — `autobutler:resolve-issue`, `autobutler:gh-stack` — live in
[autobutler-org/agent-plugins](https://github.com/autobutler-org/agent-plugins), a Claude Code plugin shared by
every repository in the org. `.claude/settings.json` names the marketplace and enables the plugin, so Claude Code
offers to install it the first time you open the repository; accept the prompt. To install it by hand:

```text
/plugin marketplace add autobutler-org/agent-plugins
/plugin install autobutler@autobutler-org
```

Skills about this codebase (`architecture`, the `quark-widgets-*` pair) stay in `.claude/skills/`.

## Clone and build

```bash
git clone https://github.com/autobutler-org/quark.git
cd quark
make setup
make generate
make build
```

`make generate` runs sqlc and swag to produce generated files. Run it after any database schema change or API
annotation change. The CI will fail if you forget — it checks for a clean working tree after running `make
generate`.

## Run the backend

There are two modes, and they differ in both scheme and port. Pick one explicitly:

```bash
make watch/backend         # plain HTTP  on http://localhost:8080
make watch/backend/secure  # HTTPS       on https://localhost (:443)
```

Both hot-reload via air — edit Go files and the server restarts automatically. The `serve/backend` and
`serve/backend/secure` targets are the same two modes without the file watcher.

Which one you want depends on what you're working on:

| | `watch/backend` | `watch/backend/secure` |
| --- | --- | --- |
| URL | `http://localhost:8080` | `https://localhost` (:443) |
| TLS | none | self-signed, generated on first boot into `<dataDir>/certs/` |
| `curl` | works as-is | needs `-k` (the cert is not in any trust store) |
| Root needed | no | on Linux, yes — `:443` is privileged |

The self-signed certificate is never added to your system trust store. The Flutter client opts out of chain
verification for private and loopback addresses, and for a local name (`quark.local`, `quark.lan`) only once it is
the Quark you chose, so it can talk to it; browsers and `curl` will still warn.

If you need root — for USB device mounting on Linux, or to bind `:443` in secure mode — prefix any backend
target with `AS_ROOT=1`, which runs it under `sudo`:

```bash
make watch/backend AS_ROOT=1
make watch/backend/secure AS_ROOT=1
```

## Run the frontend

Web:

```bash
make serve/frontend
```

Serving to a browser on another machine? Use `make serve/frontend FLUTTER_BUILD_MODE=release`. The default
debug build ships thousands of unbundled modules and is very slow over a network; release serves one minified
bundle, at the cost of hot reload.

Mobile (pick an emulator first):

```bash
make emulate          # default platform
make emulate/android  # or emulate/ios
make serve/frontend/mobile
```

Linux desktop:

```bash
make build/frontend/linux                  # release bundle
./build/linux/x64/release/bundle/quark
flutter run -d linux                       # or run it with hot reload
```

The bundle is the `quark` binary plus its `lib/` and `data/` directories; move them together. The build needs
Ninja, CMake, Clang, pkg-config, and the GTK 3 and libsecret development files:

```bash
# Arch / Omarchy: gtk3, libsecret, clang, cmake and pkg-config are usually already installed
sudo pacman -S --needed ninja gtk3 libsecret clang cmake pkgconf

# Debian / Ubuntu
sudo apt-get install ninja-build libgtk-3-dev libsecret-1-dev clang cmake pkg-config
```

The first build downloads PDFium (for `pdfrx`) and compiles libsodium (for `sodium`), so it needs network access
and takes a few minutes; later builds reuse both.

At runtime the app keeps its sign-in in the desktop keyring through the Secret Service, so GNOME Keyring or
KWallet has to be running and unlocked. Without one the app still opens, but signing in fails with a message
saying the keyring is missing.

## Running Quark as a Linux desktop app

`make install/frontend/linux` builds the release bundle and installs it for your user, with no `sudo`. Quark then
shows up in the launcher (Super and type "quark" on Omarchy, or the GNOME and KDE app grids) and opens in its own
window with its icon.

```bash
make install/frontend/linux      # bundle in ~/.local/share/quark-app, launcher entry and icons under ~/.local/share
make uninstall/frontend/linux    # removes all of it
```

Run it again to upgrade in place. Set `PREFIX=<dir>` to install somewhere other than `~/.local`. A `quark`
symlink goes in `$PREFIX/bin`, and the launcher entry points at it by absolute path, so `~/.local/bin` does not
have to be on your `PATH`. The window class is `org.autobutler.quark`, which `linux/quark.desktop` carries as
`StartupWMClass` so Hyprland and docks match the window to its launcher entry.

## Lint and test

```bash
make check          # lint
make test/unit/backend  # Go tests
make test/unit/frontend # Flutter tests
```

Run `make check` before pushing. CI runs it and will fail if the linter is unhappy.

## Database changes

Quark uses SQLite with golang-migrate for schema migrations and sqlc for query generation.

1. Add a migration file in `internal/db/migrations/` (name it `NNN_description.up.sql` and `NNN_description.down.sql`)
2. Add queries to `sql/queries/`
3. Run `make generate` to regenerate `internal/db/*.sql.go`
4. Commit the generated files

Don't edit the generated files by hand — they'll be overwritten by `make generate`.

### PostgreSQL

SQLite is the default and needs nothing. To run against PostgreSQL instead, with Docker installed:

```bash
make serve/postgres                      # start the development database
make serve/backend DB_BACKEND=postgres   # or watch/backend
```

The server then runs on PostgreSQL. A few features answer `501` there for now; the
[list](architecture/data.md#what-does-not-work-on-postgresql-yet) names each with its issue.

[PostgreSQL for local development](architecture/postgres-dev.md) has the rest: the `DB_BACKEND` switch,
resetting the database, `psql`, and running both backends side by side.

## Project structure

```text
cmd/quark/         Entry point
internal/
  db/                   sqlc-generated database layer + migrations
  server/api/v0/        API handlers (one file per route group) — mounted at /api/v0
    albums/             Photo album CRUD
    files/              File browser and file-type listing
    favorites/          Favorites toggle, list, and check
    photos/             Photo listing, metadata, rotation
    thumbnails/         Thumbnail generation and cache
  server/middleware/    Gin middleware (auth, OTEL, etc.)
pkg/util/               Shared utilities
  authutil/             Auth: hashing, session tokens, recovery phrases
  favoritesutil/        Favorites toggle and smart-album sync logic
  serverutil/           HTTP helpers: Response type, WrapApiRoute, HttpError
  sqlutil/              DB helpers: FormatTime, NullInt64, NullStringPtr
  storageutil/          File system and device utilities
  updateutil/           Version/update logic
lib/                    Flutter frontend
  models/               Data models (FileNode, PhotoAlbum, etc.)
  pages/                UI pages (photos, docs, sheets, file browser, etc.)
  services/             API clients (FilesService, FavoritesService, etc.)
  utils/                Shared utilities (SafeSetStateMixin, path helpers)
  widgets/              Reusable components
    layout/             AppBar, drawer, brand button
    photos/             Album sidebar, album tree tile, photo grid
docs/                   This documentation
packages/data_table/    Pure-Dart headless spreadsheet engine (used by sheets editor)
sql/queries/            sqlc SQL queries
```

## Common Makefile targets

Run `make help` to see everything. The most common ones:

| Target                          | What it does                          |
| ------------------------------- | ------------------------------------- |
| `make watch/backend`            | Backend hot reload, http://localhost:8080 |
| `make watch/backend/secure`     | Backend hot reload, https://localhost |
| `make serve/frontend`           | Flutter web dev server                |
| `make serve/frontend/mobile`    | Flutter mobile (after `make emulate`) |
| `make emulate/android`          | Launch Android emulator               |
| `make emulate/ios`              | Launch iOS simulator                  |
| `make build/frontend/linux`     | Linux desktop release bundle          |
| `make build`                    | Build everything                      |
| `make generate`                 | Regenerate sqlc + swagger             |
| `make generate/backend/sqlc`    | Regenerate DB layer only              |
| `make generate/backend/swagger` | Regenerate API docs only              |
| `make check`                    | Full lint + format check              |
| `make format`                   | Auto-format code                      |
| `make test/unit/backend`        | Go unit tests with coverage           |
| `make test/unit/frontend`       | Flutter unit tests                    |
| `make test/integration/backend` | Go integration tests                  |
| `make tidy`                     | Tidy Go + Flutter dependencies        |
| `make serve/postgres`           | Start the development PostgreSQL      |
| `make clean/postgres`           | Stop it (`clean/postgres/data` resets) |

## Notes

- USB device mounting requires root on Linux, as does binding `:443` in secure mode (`AS_ROOT=1`)
- Swagger UI is at `http://localhost:8080/swagger` in insecure mode, or `https://localhost/swagger` in secure mode
- `make generate` must be run before committing schema or query changes — CI enforces this
