Status: Proposed

<!-- cspell:words IPTC phototagutil -->

# Photo tags and filters

Design for [#3567](https://github.com/autobutler-org/quark/issues/3567): tags a person types onto their own photos
("receipts", "school", "garden"), and filtering the library and albums by those tags and by favorites, combined
with the file name search that ships today (JN-PH-023) and the metadata filters planned in #3496. Nothing here is
built yet. The owner approves this page, or sends it back, before any ticket below is labeled.

## Rules from the issue

These are fixed; the design is built around them, not the other way round.

1. **People type every tag.** Nothing is inferred from image content, file names, folders, dates or EXIF. No model,
   no recognition, no suggestion that the account did not type itself at some point.
2. **The file is never modified.** No embedded keywords, no sidecar files next to the photo, no change to its bytes
   or its modification time.
3. **Tags are per-account by default**, like favorites since #1912. Whether a tag can be shared is an open question
   ([Q1](#open-questions-for-the-owner)); this design ships per-account tags only and keeps the door open.
4. **A photo only shows up in tag results, tag counts or tag suggestions for an account that can already open it.**
5. **Filters keep the chosen sort** (JN-PH-018) and work inside an album as well as across the whole library.

## How favorites and albums work today

Tags are the third piece of per-account photo metadata, so they copy the two that exist.

- **`photo_favorites`** (`004_photos`, rebuilt by `014_per_user_photos`) is one row per
  `(user_id, device_serial, rel_path)`, `ON DELETE CASCADE` from `users`. `sql/queries/photo_favorites.sql` holds
  the queries; `pkg/util/favoritesutil` the logic; `internal/server/api/v0/favorites` serves `/photos/favorite` and
  `/photos/favorites`. Every read is filtered through `accessutil.Access.Check(..., Read)` in Go, so a favorite on a
  photo the account can no longer read stays in the table and is simply not returned.
- **Albums** (`photo_albums`, `photo_album_items`) are pointers, not copies: an item is
  `(album_id, device_serial, rel_path)`, and the album carries `user_id`. `albumutil.ListItems` filters items by
  access and sorts them with the same sort keys as the library.
- **Both follow the file.** `fileutil.movePhotoRows` (`pkg/util/fileutil/helpers.go`) rewrites favorites and album
  items when a file or folder moves or is renamed, across devices too, with the count guard from rule 11 of
  [`sql/queries/AGENTS.md`](../../sql/queries/AGENTS.md). `fileutil.DeleteFiles` (`pkg/util/fileutil/delete.go`)
  drops them, along with rotations and hashes, when a path is deleted, including a move to the trash.
- **Playback positions** (`pkg/util/playbackutil`) are a second shape of per-account state: a JSON value in
  `vfs_metadata` under the key `playback.<user id>`. Its package doc says it plainly: state is keyed by path, and
  nothing rewrites `vfs_metadata` on a move, so a rename drops it.
- **The library listing** (`photoutil.ListPhotos`, `GET /photos`) walks every device, keeps readable images whose
  name matches `q`, sorts, then pages; `total` counts the matches (#3523). The **Favorites tab** in the app filters
  the pages already loaded against the favorite keys (`PhotosController._visible`), so on a large library it only
  shows the favorites among the pages scrolled so far. A server-side favorites filter fixes that as a side effect.

## Where tags live

### The options

| Option                                                          | Verdict  | Why                                                                                                                                                                                                                                                                                                                                                                                 |
| --------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Keywords embedded in the file (EXIF/IPTC/XMP)                   | Rejected | Breaks rule 2. Also shared with every account that can read the file, which breaks rule 3.                                                                                                                                                                                                                                                                                          |
| A sidecar file beside each photo (`IMG_1.jpg.xmp`)              | Rejected | Not the file, but it is a new file in the user's folders: it shows up in Files, gets copied by every backup and sync, litters USB drives, and is readable by every account that can read the folder. Per-account sidecars would leak account ids into the tree.                                                                                                                |
| A JSON file per account in the data dir                         | Rejected | On disk, so it honors the letter of "database last". But moves and deletes already update the photo tables in one place (`movePhotoRows`, `DeleteFiles`), and this would need a second, non-transactional update path with its own locking; renaming a tag rewrites the file; account deletion needs manual cleanup; a PostgreSQL install with more than one instance (#2964) would race on it. |
| `vfs_metadata`, key `photos.tags.<user id>`, a JSON list        | Rejected | No migration, and playback is a precedent. But tag identity (rename one tag everywhere) needs a list of tags somewhere else anyway; "photos with tag X" means decoding every row the account has; there is no shared SQL for searching inside a JSON array (only `json_extract(column, '$.key')` is defined on PostgreSQL, rule 16); no foreign key, so a deleted account leaves rows; and nothing moves `vfs_metadata` on a rename today. |
| **Two tables beside `photo_favorites`**                         | **Chosen** | See below.                                                                                                                                                                                                                                                                                                                                                                       |

### The case for a table, made against the rule

`AGENTS.md` says to add a table only as a last resort, for data that cannot be managed reliably on disk. Tags meet
that bar for the same reasons favorites and albums did, and
[`docs/architecture/data.md`](../architecture/data.md) already names "collections" as one of the few things the
database holds:

- **They cannot be derived from disk.** A tag is something a person said about a photo. There is nothing to rebuild
  it from, which is what separates it from `photo_hashes` or `music_tracks`, which are caches.
- **They cannot be written to disk without breaking rule 2 or rule 3**, as the table above shows.
- **They must survive a rename or move within Quark** (the issue's acceptance sketch). The only code that sees every
  move and delete and updates per-photo metadata in step is the database path in `fileutil`. Joining it costs two
  query calls per hook; any other store needs a second mechanism that can drift.
- **They are per-account and must go when the account goes.** `ON DELETE CASCADE` from `users` does that with no
  code, exactly as it does for favorites.
- **The query shape is relational.** "Every photo with tag A and tag B", "rename tag A", "how many photos carry each
  tag" are one indexed query each, on both engines.

What the database does **not** get: no copy of the photo, no thumbnail, no derived index, no per-photo row for an
untagged photo. A photo nobody tagged costs nothing.

### Schema

Migration `040_photo_tags` (the number above `main`'s highest today, 039; renumber if `main` moves first), in both
`internal/db/migrations/` and `internal/db/migrations_postgres/`, per [`internal/db/AGENTS.md`](../../internal/db/AGENTS.md).
SQLite shown; the PostgreSQL file maps the types with that file's table.

```sql
-- A tag one account made. name is shown as typed; name_key is its folded
-- spelling, computed in Go, so "Garden" and "garden" are one tag on both
-- engines.
CREATE TABLE photo_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE (user_id, name_key)
);

-- One tag on one photo: a pointer, like an album item.
CREATE TABLE photo_tag_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_id INTEGER NOT NULL,
    device_serial TEXT NOT NULL DEFAULT '',
    rel_path TEXT NOT NULL,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tag_id) REFERENCES photo_tags (id) ON DELETE CASCADE,
    UNIQUE (tag_id, device_serial, rel_path)
);

-- Moves, deletes and "tags on this photo" look a path up across tags.
CREATE INDEX idx_photo_tag_items_path ON photo_tag_items (device_serial, rel_path);
```

- **`name_key` instead of a case-insensitive index.** `internal/db/AGENTS.md` maps SQLite's `COLLATE NOCASE` to a
  PostgreSQL index on `LOWER(name)`, but SQLite folds ASCII only and PostgreSQL folds by locale, so "Café" and
  "CAFÉ" would be one tag on PostgreSQL and two on SQLite. Folding in Go (`strings.ToLower` after trimming) and
  storing the result gives both engines the same answer with a plain `UNIQUE`.
- **The owner sits on the tag, not the item.** An item belongs to whoever owns its tag, so every item query joins
  `photo_tags` on `user_id`. That keeps one place to scope by account, and a shared tag later (Q1) changes the tag
  row, not every item.
- **No IN-lists.** Rule 15 has no shared form for one, so a batch apply loops over its photos inside `db.InTx` and
  a multi-tag filter runs one query per tag and intersects in Go.

### Queries

`sql/queries/photo_tags.sql`, shared by both engines:

| Query                     | Shape                                                                                                 |
| ------------------------- | ----------------------------------------------------------------------------------------------------- |
| `CreatePhotoTag`          | `INSERT ... RETURNING *`; a taken `name_key` is a unique violation the service turns into a conflict   |
| `GetPhotoTag`             | by `id` and `user_id`, so another account's id is not found                                           |
| `GetPhotoTagByKey`        | by `user_id` and `name_key`, for "tag these photos `garden`" by name                                  |
| `ListPhotoTags`           | the account's tags, `ORDER BY LOWER(name), id`                                                        |
| `RenamePhotoTag`          | `UPDATE ... WHERE id AND user_id`                                                                     |
| `DeletePhotoTag`          | `DELETE ... WHERE id AND user_id`; items go with it by cascade                                        |
| `AddPhotoTagItem`         | `ON CONFLICT (tag_id, device_serial, rel_path) DO NOTHING` (rule 11)                                  |
| `RemovePhotoTagItem`      | by tag, serial and path                                                                               |
| `ListPhotoTagItems`       | every item of one tag, for a filter and for counts                                                    |
| `ListPhotoTagsForPhoto`   | the account's tags on one `(serial, path)`, for the viewer                                            |
| `MovePhotoTagItems`       | the `MoveFavorites` shape: prefix rewrite with the count guard, keyed on `tag_id` instead of `user_id` |
| `DeletePhotoTagItemsUnder`| the `DeleteFavoritesUnder` shape, for every account                                                   |

`MovePhotoTagItems` gets a case in `exerciseQuerier` (`internal/db/pgdb/adapter_test.go`), since it is the one
whose behavior, not just syntax, matters on both engines.

### What happens to a photo's tags

| What happens to the photo                                         | Its tags                                                                                                                       |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Renamed or moved in Quark, a file or a folder, on one device or across two | Follow it. `movePhotoRows` calls `MovePhotoTagItems` then `DeletePhotoTagItemsUnder` on the old path, as it does for favorites. |
| Moved to the trash                                                | Dropped, by `DeleteFiles`' cleanup, as favorites and album items are today. Restoring it brings the photo back untagged (Q4).  |
| Deleted permanently                                               | Dropped (already gone at the trash step).                                                                                      |
| Copied (`POST /photos/copy`, or a file copy)                      | The copy starts untagged, as it starts unfavorited.                                                                            |
| Rotated                                                           | Unchanged; rotation is its own row and the path does not move.                                                                 |
| On a USB drive that is unplugged                                  | Kept, keyed by serial, and back when the drive is.                                                                             |
| Changed outside Quark (the drive edited on another computer)      | The row is orphaned. It never appears in a result, because every filter is intersected with the files the listing walks, but it does count in the tag list until a sweep removes it (Q6). |
| The account loses read access                                     | Kept, and invisible to it, as with favorites. They come back if access does.                                                   |
| The account is deleted                                            | Deleted, by cascade from `users`.                                                                                              |

## API

All of it is additive: new routes, and new optional query parameters that change nothing when absent.

### Managing tags

A new handler package, `internal/server/api/v0/tags/` (`package v0_tags`), serving `/api/v0/photos/tags/*` the way
`v0_favorites` serves `/photos/favorite*`, with business logic in a new `pkg/util/phototagutil` using
Params/Result.

| Route                                  | Body / query                                                       | Answer                                                                                                                         |
| -------------------------------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| `GET /photos/tags`                     | —                                                                  | `200 [{id, name, count, createdAt}]`, the caller's own tags. `count` counts items on photos the caller can read.             |
| `POST /photos/tags`                    | `{name}`                                                           | `201 {id, name, ...}`; `400` for a bad name; `409` when the caller already has a tag that folds to it.                        |
| `PATCH /photos/tags/:id`               | `{name}`                                                           | `200`; `404` when the id is not the caller's; `409` on a collision (Q3 asks whether that should merge instead).              |
| `DELETE /photos/tags/:id`              | —                                                                  | `204`. Removes the tag from every photo; the photos are untouched.                                                            |
| `POST /photos/tags/apply`              | `{photos: [{deviceSerial, relPath}], add: [name], remove: [id]}`   | `200 {tagged, skipped, tags}`. Adds and removes on many photos in one transaction. An `add` name that does not exist yet is created. A photo the caller cannot read is skipped and counted, never tagged. |
| `GET /photos/tags/photo`               | `?serial=&relPath=`                                                | `200 [{id, name}]`, the caller's tags on that photo; `404` when the caller cannot read it, as `toggleFavorite` does.          |

- **Name rules**, in `phototagutil`: trimmed; 1 to 64 characters; no control characters; case is kept for display
  and ignored for identity. Proposed limits, to bound what one account can make large: 1,000 tags per account and
  1,000 photos per apply request (Q7).
- **Every query is scoped by the principal's `user_id`.** Admins get their own tags, like their own favorites; no
  route takes a user id.
- **Ids in URLs, names in bodies.** The request log records URLs, so a filter by tag carries an id, never a tag
  name.

### Filtering

`GET /photos` and `GET /albums/:id/items` take the same new parameters:

| Parameter       | Meaning                                                                            |
| --------------- | ---------------------------------------------------------------------------------- |
| `tag=<id>`      | Repeatable. Keeps photos that carry **every** listed tag (Q2 asks about "any").    |
| `favorite=true` | Keeps the caller's favorites.                                                      |
| `q`             | As today on `GET /photos`: file name contains, ignoring case. New on album items.  |
| #3496's filters | Date range, camera, place: names are #3496's to choose; they join the same AND.    |

- **Everything ANDs.** A photo is listed when it is an image the caller can read, its name matches `q`, it is a
  favorite if `favorite=true`, it carries every `tag`, and it passes every #3496 filter. The access check stays the
  first test, so no filter can widen what an account sees.
- **The sort is kept.** `ListPhotos` already filters before `sortPhotos` and pages after, so `sort`, `order`,
  `offset`, `limit` and `total` work unchanged: `total` is the count of matches, as it already is for `q`.
- **How the filter runs.** Before the walk, `phototagutil` loads the caller's items for each requested tag and
  intersects them into one set of `(serial, path)` keys, and `favoritesutil` does the same for favorites. The walk's
  predicate checks membership. The walk itself is unchanged and stays #2759's to replace; when that index lands, the
  same keys become a join.
- **A tag id that is not the caller's** answers `404 tag not found`, the same answer as an id that does not exist,
  so it says nothing about other accounts.
- **Not on `/files/search`.** That route is the Files page's name search, capped at 500 and not photo-specific;
  tags stay a Photos filter.

### Events

A tag change does not change the file tree, so the "mutations publish events" rule does not require one, and
favorites publish none today. Live sync of tags between one account's sessions would be a new per-account event
filtered like `EventVaultChanged` (`UserID` never sent). It is left out of the tickets (Q8).

## Flutter

At the level the tickets need; each ticket's own spec carries the detail.

- **Package widgets** (`packages/quark_widgets/lib/src/photos/tags/`), data in and callbacks out, each with its
  `///` docs, a narrow and a wide test, and a gallery entry:
  - `PhotoTagChip` — one tag, optionally removable. Key prefix `photo_tag_chip_<id>`.
  - `PhotoTagEditor` — the sheet for tagging one or many photos: the tags the photos already have, a field that
    suggests from the caller's own tags (passed in) as they type, and add and remove callbacks. It never invents a
    suggestion.
  - `PhotoFilterBar` — the active filters as chips (favorites, each tag, later #3496's), with clear-one and
    clear-all callbacks. It takes the chips; it does not know where they came from.
  - `PhotoTagList` — the caller's tags with counts, and rename and remove callbacks, for managing them.
- **A glyph** for "tag" and one for "filter" in `QuarkIcons`, aliased in `packages/quark_icons/lib/quark_icons.dart`
  if missing. The filter opens from a `QuarkBarIconButton` with a tooltip and a `ValueKey`; tagging a selection is
  an action on the existing selection bar.
- **Data layer.** `lib/services/photo_tags_service.dart` (throws `ApiException` / `throwApiError`),
  `lib/models/photo_tag.dart`, `lib/models/photo_filter.dart` (favorites flag, tag ids, room for #3496's fields),
  and a `PhotoTagsController` in `lib/controllers/` that owns the tag list and the apply, rename and remove calls,
  with its service calls injected as function parameters.
- **Listing.** `FilesService`'s photo listing and `AlbumService.listAlbumItems` pass the filter's parameters;
  `PhotosController` takes a `PhotoFilter` and lists again when it changes. The Favorites tab moves to
  `favorite=true`, so it pages over the whole library rather than the pages loaded. Device photos (the Mobile tab)
  have no Quark record and cannot be tagged; a tag filter shows Quark photos only.
- **Error copy** from `Errors` in `lib/utils/error_text.dart`: `Errors.message(error, 'tag the photos')`,
  `'rename the tag'`, `'remove the tag'`, `'load your tags'`, and one new `static` branch for `409`, "You already
  have a tag with that name."
- **Demo mode** tags the sample library on the device, the way it searches it.
- **User journeys.** Three new journeys, numbered from the highest in use when the ticket lands (`JN-PH-027` on
  today's `main`): tag photos, filter by tags and favorites, and rename or remove a tag, in [`docs/user-journeys/photos.md`](../user-journeys/photos.md).

## Privacy and visibility

What the design decides, all following favorites:

- **Only the account that made a tag sees it**: its name, its count, which photos carry it, and its suggestions.
  No route takes another account's id.
- **Admins see their own tags and nobody else's** through the API, as with favorites ("every account has its own
  favorites, admins included"). An admin can read every photo, so an admin's own tag results span the library.
- **Counts and results are computed per request against the caller's access**, so a grant revoked a moment ago
  drops those photos from the next count.

What the owner decides (flagged, not decided here):

- **Shared tags (Q1).** If a tag can be shared with everyone who can open a photo, then: who may rename or remove
  it; whether a shared tag's count is computed per viewer (it must be, or it leaks the existence of photos the
  viewer cannot open); and whether a viewer sees who added it. The schema leaves room: the owner sits on
  `photo_tags`, so a later migration adds a scope there without touching items.
- **Database-level visibility (Q5).** Tag names sit in plain text in `quark.db`, as album names and favorites do.
  Anyone with the disk or a shell as root can read every account's tags. That matches today's photo metadata; say
  so if tags should be held to a higher bar.

## Migration, backup and export

- **Migration.** `040_photo_tags` creates two empty tables and an index on both engines; no backfill, nothing
  rewritten. Its `.down.sql` drops exactly those. Nothing goes on the
  [does-not-work-on-PostgreSQL list](../architecture/data.md#what-does-not-work-on-postgresql-yet): every query is
  shared. The `004`/`014` rows of data.md's migration table gain a `040` sibling.
- **Backup.** A snapshot (`pkg/backup`) holds the files, the chat export and the vault export. The rest of
  `quark.db`, albums and favorites included, is not in it ([backup.md](../architecture/backup.md)), and tags would
  be lost with the internal disk the same way. Whether per-account photo metadata (tags, favorites, albums) should
  join the snapshot as a `photos_backup.db` is Q4's companion and is not in these tickets.
- **Export.** No per-account export of favorites or albums exists. A `GET /photos/tags/export` is easy to add later
  and is not proposed now.
- **Factory reset** clears tags with the rest of the schema. **Moving from SQLite to PostgreSQL** (#2962) has to
  carry these two tables when that tool exists.

## Test plan

Backend, `make test/unit/backend`:

- `phototagutil`: name rules (trim, folding, length, control characters), create and rename collisions, apply that
  adds and removes in one call, apply skipping unreadable photos, the limits.
- `fileutil`: a tagged file renamed, moved into another folder, moved to another device; a folder moved with tagged
  photos inside; the destination already carrying the tag (the count guard); trash and delete dropping tags.
- `photoutil.ListPhotos`: tag, two tags, favorite, `q` and a tag together, each under every sort; `total` and paging
  over a filtered set.
- `internal/db/prepare_test.go` prepares every new statement on SQLite automatically; `exerciseQuerier` gains
  `MovePhotoTagItems`.

Integration, `make test/integration/backend`:

- `v0_tags` routes: create, list with counts, rename, `409`, delete, apply, per-photo tags.
- Access: account B gets `404` for A's tag id on every route and as a `tag=` filter; A's count excludes a photo A
  cannot read; an admin sees only the admin's own tags; deleting an account removes its tags.
- `GET /photos` and `GET /albums/:id/items` with `tag`, `favorite` and `q` combined, sort kept.

PostgreSQL, `make test/db/postgres`: `TestPostgresSchemaMatchesSQLite`, `TestPostgresMigrationsRunDownAndUpAgain`,
the PostgreSQL prepare test, and the `exerciseQuerier` case.

Frontend, `make test/unit/frontend`:

- Widget tests for the four package widgets at 360x640 and 1280x800; the gallery registry test.
- `PhotoTagsService` and `PhotoTagsController` with fakes; `PhotosController` listing again on a filter change and
  the Favorites tab paging through `favorite=true`.
- `test/utils/error_text_test.dart` stays green; the `409` branch gets a case.

## Tickets, in landing order

Each is one reviewable PR. Paths are disjoint between tickets; a ticket that consumes another's API is stacked on
it as "blocked by". Backend tickets 3 and 4 change the API, so the Flutter tickets that consume them sit above them
in the same stack, per the AGENTS.md rule on backend API changes.

1. **Schema and queries for photo tags.** `internal/db/migrations/040_photo_tags.*`,
   `internal/db/migrations_postgres/040_photo_tags.*`, `sql/queries/photo_tags.sql`, the regenerated
   `internal/db/` and `internal/db/pgdb/`, the `exerciseQuerier` case, the migration row in
   `docs/architecture/data.md`.
   **Difficulty: standard.** The schema is decided here; the work is two engines kept in step.
2. **Tag service.** `pkg/util/phototagutil/` (new package: create, rename, delete, list with counts, apply, tags for
   a photo, the filter key set), with its tests, and `phototagutil` in `.vscode/cspell.json`. Blocked by 1.
   **Difficulty: standard.** A well-scoped package with Params/Result and an access check per photo.
3. **Tag routes.** `internal/server/api/v0/tags/` (new), its mount in `internal/server/routes.go`, regenerated
   `docs/swagger/`, integration tests. Blocked by 2.
   **Difficulty: standard.** Six handlers in the house layout.
4. **Tags follow moves and deletes.** `pkg/util/fileutil/helpers.go`, `pkg/util/fileutil/delete.go` and their tests.
   Blocked by 1; can land beside 2 and 3.
   **Difficulty: mechanical.** Two calls in each of two hooks, mirroring favorites.
5. **Tag and favorite filters on the photo listings.** `pkg/util/photoutil/list.go`, `pkg/util/albumutil/`,
   `internal/server/api/v0/photos/list_photos.go`, `internal/server/api/v0/albums/list_album_items.go`, swagger,
   tests. Blocked by 2.
   **Difficulty: standard.** Threads one predicate through two listings without changing sort or paging.
6. **Tag widgets.** `packages/quark_widgets/lib/src/photos/tags/`, the barrel export, the gallery registry and
   regenerated `docs.g.dart`, the icon aliases in `packages/quark_icons/`, tests. No backend dependency; can land
   first.
   **Difficulty: standard.** Four presentational widgets under the package rules.
7. **Tag data layer in the app.** `lib/services/photo_tags_service.dart`, `lib/models/photo_tag.dart`,
   `lib/models/photo_filter.dart`, `lib/controllers/photo_tags_controller.dart`, the `Errors` branch in
   `lib/utils/error_text.dart`, their tests. Blocked by 3.
   **Difficulty: standard.** Service, model and controller in the injected-function pattern.
8. **Filtered listing in the app.** `lib/services/files_service.dart` (photo listing parameters),
   `lib/services/album_service.dart`, `lib/controllers/photos_controller.dart`, `lib/services/demo_photos_service.dart`,
   their tests. Blocked by 5 and 7.
   **Difficulty: standard.** Moves the Favorites tab to the server filter and relists on a filter change.
9. **Tagging and filtering on the Photos page.** `lib/pages/photos_page.dart`, `lib/pages/image_viewer_page.dart`,
   `lib/widgets/photos/` (the filter and tag bar wiring), `docs/user-journeys/photos.md`, page tests. Blocked by 6
   and 8.
   **Difficulty: standard.** Composition of package widgets and controller state; no new layout.

## Open questions for the owner

1. **Shared tags.** Ship per-account only (this design), or let a tag be shared with everyone who can open the
   photo? If shared: who may rename or remove it, and does a viewer see who added it?
2. **Two tags: all or any?** This design ANDs repeated `tag=` ("receipts" and "2024"). Is "any of" needed in the
   first version?
3. **Renaming onto an existing tag.** `409` (this design), or merge the two tags?
4. **Tags through the trash.** Today favorites and album items are dropped when a photo is trashed, so a restored
   photo comes back bare. Keep that for tags, or carry tags (and favorites and album items, in one fix) through the
   trash? Also: should per-account photo metadata join the snapshot backup?
5. **Plain text in `quark.db`.** Tag names are as readable to someone with the disk as album names are. Acceptable?
6. **Orphaned rows.** A photo changed outside Quark leaves its tag rows behind; they never show in results but do
   inflate the tag count. Accept that, or add a sweep (on listing, or as a job)?
7. **Limits.** 64 characters per name, 1,000 tags per account, 1,000 photos per apply request: right numbers?
8. **Live sync between one account's sessions.** Add a per-account `photo_tags_changed` event now, or wait until
   favorites need the same?
