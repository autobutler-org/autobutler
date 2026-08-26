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

```bash
make watch/backend
```

This starts the server on `http://localhost:8080` with hot reload via air. Edit Go files and the server restarts
automatically.

If you need root for USB device mounting (Linux):

```bash
make watch/backend AS_ROOT=1
```

## Run the frontend

Web:

```bash
make serve/frontend
```

Mobile (pick an emulator first):

```bash
make emulate          # default platform
make emulate/android  # or emulate/ios
make serve/frontend/mobile
```

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

## Project structure

```text
cmd/quark/         Entry point
internal/
  db/                   sqlc-generated database layer + migrations
  server/api/v1/        API handlers (one file per route group)
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
| `make watch/backend`            | Backend with hot reload               |
| `make serve/frontend`           | Flutter web dev server                |
| `make serve/frontend/mobile`    | Flutter mobile (after `make emulate`) |
| `make emulate/android`          | Launch Android emulator               |
| `make emulate/ios`              | Launch iOS simulator                  |
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

## Notes

- USB device mounting requires root on Linux (`AS_ROOT=1`)
- Swagger UI is at `http://localhost:8080/swagger` once the backend is running
- `make generate` must be run before committing schema or query changes — CI enforces this
