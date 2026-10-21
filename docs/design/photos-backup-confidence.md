<!-- cspell:words PhotoKit MediaStore nomedia HEIC exFAT NTFS photosourceutil -->

# Photos backup confidence

**Status: Proposed.** Nothing here is built. The owner approves this before any ticket below is filed.

Part of #3569. Sits beside the iPhone backup epic (#285) and the import epic (#904), and duplicates neither
engine. Related: #3533 (a failed batch does not say which files landed), #1231 (Photos polish).

## The question this answers

Someone opens Photos and wants to know, for each phone they copy photos from: *are my photos safe at home?*
The view answers four things per phone:

1. When photos were last copied from it to the Quark.
2. How many photos from it are on the Quark now.
3. The newest photo from it that is on the Quark.
4. Whether the phone has photos newer than that, which means they are not safe at home yet.

Where the phone cannot tell the app what is in its own library, the view says so. It never implies that
everything is safe when it does not know. This is a reporting surface over the uploads Quark already does. It
is not a new backup engine, does not upload anything in the background, and touches no cloud service.

### Naming

"Backup" already means something in Quark: the snapshot backup of the whole Quark to a drive (`pkg/backup`,
`backup_jobs`, the `backup_due` and `backup_stale` reminders, JN-SD-008). To keep the two apart in code, this
design calls a phone or browser that uploads photos a **photo source**, and the tables, package and routes use
that name (`photo_sources`, `photosourceutil`, `/photos/sources`). The app's copy can still say "backed up to
your Quark", because that is what people call it. The tables and routes keep the source name either way.

## What exists today

What the code records about an upload, as of `5af98420`:

| Fact | Where it lives | Usable for this feature? |
| --- | --- | --- |
| The file itself | the device's VFS namespace (`files`, `files:<serial>`) | Yes. Present means present. |
| Which account uploaded it | `path_access` owner row, written by `grantOwners` (`internal/server/api/v0/files/helpers.go:94`) | **No.** `GrantOwnerIfNeeded` writes nothing for an admin, or for an upload into a folder the caller already owns (`pkg/util/accessutil/accessutil.go:353`). Most uploads leave no row. |
| When it was uploaded | file mtime on disk | Weak. Nothing says the mtime is the upload time, and a later edit or copy changes it. |
| When it was taken | `photo_hashes.taken_at`, from EXIF, filled lazily by the listing and the backfill (`pkg/util/photoutil/list.go:354`, `hashes.go:281`) | Partly. It is a camera wall-clock time with no time zone, and is NULL for screenshots and anything without EXIF. |
| Which phone sent it | nothing | **No.** `connected_devices` is keyed by IP address and User-Agent, holds no account, is pruned to the newest rows (#2756), and its list is open to every user (`internal/server/api/v0/devices/devices.go`). The app's User-Agent is `Dart/…` for every phone. `sessions` has no device column. Upload sessions (`pkg/util/uploadutil/session_store.go`) live in memory for 24 hours. |
| What is in the phone's library | nothing, and nothing is sent | No. |

What the app does today:

- **Mobile upload** (JN-PH-006) goes through `FilePicker.pickFiles(type: FileType.image)` in
  `lib/pages/photos_page.dart:348`, then `PhotosController.uploadPhotos` sends a multipart
  `POST /api/v0/files/upload` with `keepBoth=true` and a client-rendered `thumbnail` part after each file
  (`lib/controllers/photos_controller.dart:1577`). The picker hands back a copy of the file. It carries no
  library asset id and no capture time.
- **The Mobile tab** (JN-PH-002) reads the phone library through `photo_manager` (`PhotosController.deviceAssets`,
  `photos_controller.dart:932`): the "All" album, images only, the first 200. It calls
  `PhotoManager.requestPermissionExtend()`, which prompts.
- **Web** drag-and-drop and picking (JN-PH-017) sends the same multipart upload. The browser has no view of any
  photo library.
- There is no automatic or background upload on any platform.

So two facts this feature needs are recorded nowhere: **which phone a photo came from**, and **what the phone
has**. Everything else (presence, capture date) can be read from what exists.

## Where the per-source data comes from

### What can be derived and what cannot

| Answer | Derivable from existing data? |
| --- | --- |
| Photos on the Quark that came from phone X | No. Nothing records the phone. Ownership rows are absent for most uploads (see above). |
| When phone X last copied photos | No. Same reason. |
| Newest photo from phone X on the Quark | Only once (1) is known. Then `photo_hashes.taken_at` or the capture time sent at upload. |
| Whether phone X has newer photos | Only the phone knows. It has to tell the Quark, or the view has to run on the phone. |

### Options considered

**A. Derive per account, not per phone, from existing rows.** No new storage. It fails the request: it cannot
split by phone, it misses every upload that left no owner row (all admin uploads, everything into a person's own
folder), and it has no upload time. Rejected.

**B. Keep it all on the phone.** The app remembers locally what it uploaded. No server storage. It fails twice.
It cannot see that a photo was later deleted or moved on the Quark, so it reports "safe" for something that is
gone. That is the one failure this feature must not have. And it cannot be seen from any other device. Rejected.

**C. Encode the source in where the file lands**, e.g. `Photos/From Mom's iPhone/`, and derive from the path.
No database. It changes where uploads land (JN-PH-006 puts them at the top level of the chosen device), and it
breaks the moment someone tidies their photos into folders. Rejected.

**D. Files in the data directory or sidecar files beside the photos.** #3083 moved per-account state out of the
data directory into the database so that every instance on one database sees the same values. Writing a new
per-account file goes against that. Sidecars in the files directory clutter the person's own files, and
`path_access` records why nothing about a file can live on the drive itself: the drives may be exFAT or NTFS
with no extended attributes (`internal/db/migrations/010_path_access.up.sql`). Rejected.

**E. Store the source in `user_settings` JSON.** No migration for the source list. But two phones uploading
at once would race a read-modify-write of one JSON blob, and per-photo provenance cannot go there at all, because
it grows with the library. Rejected for the source list, and impossible for the items.

**F. Aggregate counters on a source row.** One small table, one row per phone: last upload time, a running
count, the newest capture time ever uploaded. Nothing per photo. Cheap, but the count and the newest date never
go down. Delete last week's photos on the Quark and the view still calls them safe. This is option B's failure
moved to the server.

**G. A source table plus one row per photo uploaded from a source (recommended).** Two tables, both scoped
to the account:

- `photo_sources`: one row per account and app install. It is bounded by the number of phones and browsers a
  person uses.
- `photo_source_items`: one row per photo uploaded from a source, keyed like every other photo table by
  `(device_serial, rel_path)`. It is dropped when the photo is deleted or trashed and re-keyed when it is moved,
  so "how many" and "newest" always describe photos that are still there.

### The minimization argument

AGENTS.md makes the database a last resort. This design needs it only for the two facts that are recorded
nowhere else and cannot live on disk:

- **Which phone a photo came from.** No file, path, or existing row carries it (table above). It cannot sit on
  the drive (option D), and it has to follow a photo through moves and disappear when the photo goes, which is
  exactly what the photo tables keyed by `(device_serial, rel_path)` already do (`photo_favorites`,
  `photo_album_items`).
- **The phone's last report about its own library.** A handful of fields per phone, written by the phone, read
  from any device.

Everything else is derived at read time: presence from the items table, which is kept honest by the move and
delete paths, and the capture date from `photo_hashes.taken_at` when no exact time was sent. No counters are
stored. A count is a `COUNT(*)`, so it cannot drift.

Why not put a source column on `photo_hashes`, the existing one-row-per-photo table? Because a move drops
`photo_hashes` rows on purpose and recomputes them at the new path (#1666, `pkg/util/fileutil/helpers.go:48`).
The source has to survive a move, and the hashes deliberately do not. They live and die on different terms, so they get
different tables.

### Schema

Migration `040_photo_sources` in both `internal/db/migrations/` and `internal/db/migrations_postgres/`
(renumbered above `main`'s highest at landing). SQLite shown. PostgreSQL uses `BIGINT … IDENTITY`, `BIGINT`, and
`TIMESTAMPTZ` per `internal/db/AGENTS.md`.

```sql
-- One row per account and app install that has uploaded photos or reported
-- on its library. client_id is a random value the app generates once per
-- install; it identifies the install to its own account and nothing else.
CREATE TABLE photo_sources (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id             INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id           TEXT NOT NULL,
    label               TEXT NOT NULL,
    platform            TEXT NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    created_at          DATETIME NOT NULL DEFAULT (datetime('now')),
    last_upload_at      DATETIME,
    -- The phone's last report on its own library. library_access says how
    -- much it could see: 'full', 'limited', 'denied' or 'unsupported'. The
    -- counts are NULL unless it was 'full'.
    library_access      TEXT,
    library_count       INTEGER,
    library_newest_at   DATETIME,
    library_pending     INTEGER,
    reported_at         DATETIME,
    UNIQUE (user_id, client_id)
);

-- One row per photo uploaded from a source and still on the Quark. Dropped
-- when the photo is deleted or trashed, re-keyed when it moves.
-- created_at is the capture time the phone's photo library gave, in UTC,
-- when the upload carried one; NULL otherwise.
CREATE TABLE photo_source_items (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id      INTEGER NOT NULL REFERENCES photo_sources (id) ON DELETE CASCADE,
    device_serial  TEXT NOT NULL,
    rel_path       TEXT NOT NULL,
    created_at     DATETIME,
    uploaded_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (device_serial, rel_path)
);
CREATE INDEX idx_photo_source_items_source ON photo_source_items (source_id);
```

`UNIQUE (device_serial, rel_path)`: a file has one source. An overwrite from another phone moves the row to that
phone.

The read is one grouped query per account, joining `photo_hashes` for the EXIF fallback:

```sql
-- name: ListPhotoSourceSummaries :many
SELECT s.id, s.client_id, s.label, s.platform, s.last_upload_at,
       s.library_access, s.library_count, s.library_newest_at, s.library_pending, s.reported_at,
       COUNT(i.id)        AS photo_count,
       MAX(i.created_at)  AS newest_created_at,
       MAX(h.taken_at)    AS newest_taken_at
FROM photo_sources s
LEFT JOIN photo_source_items i ON i.source_id = s.id
LEFT JOIN photo_hashes h ON h.device_serial = i.device_serial AND h.rel_path = i.rel_path
WHERE s.user_id = sqlc.arg(user_id)
GROUP BY s.id
ORDER BY s.last_upload_at DESC, s.id;
```

The query follows `sql/queries/AGENTS.md` for both engines: every non-aggregate column belongs to `s`, which is
grouped by its primary key. If the PostgreSQL adapter wants every selected column of `s` listed in the
`GROUP BY`, list them.

## What each platform can report about its own library

This is the part that decides how honest the view can be. "Library" means the phone's own photo library as the
OS exposes it, not the cloud service the phone may also sync to.

| Platform and permission | Can know the library's count | Can know its newest photo | Notes |
| --- | --- | --- | --- |
| iOS, full access | Yes | Yes | PhotoKit counts every asset in the library, including ones whose original sits only in the phone's cloud photo library under "Optimize Storage". The count is right, but uploading those needs a download first. Hidden and shared-album photos are not in the "All" album. |
| iOS, limited access (iOS 14+) | **No** | **No** | The app sees only the photos the person picked. Nothing tells it how many others exist. `photo_manager` reports `PermissionState.limited`. |
| iOS, denied | No | No | |
| Android 13+, full (`READ_MEDIA_IMAGES`) / Android 12 and below (`READ_EXTERNAL_STORAGE`) | Yes, for what MediaStore indexes | Yes, same limit | MediaStore leaves out app-private folders, folders marked `.nomedia`, and vendor "secure folders". Photos that exist only in a cloud gallery app are not on the phone, so they are not counted. The capture time comes from `DATE_TAKEN` when the file has it. Otherwise it is when the file was added, so it can be later than the true capture time. |
| Android 14+, partial (`READ_MEDIA_VISUAL_USER_SELECTED`) | **No** | **No** | Same as iOS limited. `photo_manager` reports `limited`. |
| Web, any browser, including a phone's | **No, never** | **No, never** | A browser sees only the files the person picks or drops. `File.lastModified` is often the time of the pick, not the capture, and mobile browsers may convert HEIC to JPEG on the way. |

`photo_manager` 3.12 (already a dependency) gives everything the full-access rows need without a new package:

- `PhotoManager.getPermissionState(requestOption: …)` reads the permission **without prompting**. The backup
  view uses it, so opening the view never pops a system dialog. Only an explicit "Check this phone" button
  calls `requestPermissionExtend()`.
- `AssetPathEntity.assetCountAsync` on the `onlyAll: true` path gives the count.
- `getAssetListRange(start: 0, end: 1)` with `FilterOptionGroup(orders: [OrderOption(type: OrderOptionType.createDate, asc: false)])`
  gives the newest asset. A `createTimeCond` filter gives how many assets are newer than a given time without
  loading them.

### Where completeness cannot be known, even with full access

The view has to be honest about these, in its copy and in this design:

- **"Newest is on the Quark" is a high-water mark, not a proof.** If the newest photo is home, older ones that
  were never picked can still be missing. The view says "your newest photo is on your Quark". It never says
  "everything is on your Quark".
- **The counts are shown side by side, not compared.** The phone's count includes photos the person chose not
  to copy, and the Quark's count drops when they delete photos at home. A gap between them is shown as two
  numbers, never as a warning.
- **A report is a moment.** The phone reports when the app is open. There is no background check on any
  platform. The view always shows when the phone last checked, and an old report is shown as old.
- **Uploads from before this ships carry no source.** They are not counted for any phone. See open question 7.

## Comparing the phone with the Quark

The verdict for one source, computed in the app by a pure function (`lib/utils/photo_source_status.dart`), is
the first of these that applies:

| State | When | Copy (draft, plain words for non-technical readers) |
| --- | --- | --- |
| `never` | `photo_count == 0` and no `last_upload_at` | "No photos copied from this phone yet." |
| `notYetHome` | the phone reported `full` access and its newest photo is newer than the newest safe one (rule below) | "12 photos taken since Oct 3 are not on your Quark yet." When the pending count is unknown: "This phone has photos newer than anything on your Quark." |
| `stale` | `last_upload_at` is older than `PhotoSourceConfig.staleAfter` (draft 7 days) | "No photos copied from this phone in 9 days." |
| `cannotCheck` | the report is `limited`, `denied` or `unsupported`, or the source is `web` | limited: "You let Quark see only some of your photos, so it can't tell whether newer ones are missing." Web: "A browser can't see your phone's photos, so Quark can't check them." |
| `notChecked` | no report yet, or the newest report is older than `staleAfter` | "Open Quark on this phone to check it." |
| `upToDate` | otherwise | "Your newest photo from this phone, taken Oct 18, is on your Quark." |

Every state also shows the facts underneath: last copied, how many photos are on the Quark, and the newest safe
photo, with "Checked Oct 18" beside any phone-reported number.

**The newest safe photo** is `newest_created_at` when the source has items with a capture time sent at upload.
That time came from the same `photo_manager` field the phone's report uses, in UTC, so the comparison is exact.
Otherwise the comparison falls back to `newest_taken_at` (EXIF). EXIF time is the camera's wall clock with no
zone. The API returns it as a zone-less local date-time string, and the phone compares it against its own
newest photo **converted to its local wall clock**. A phone's camera writes EXIF in the phone's local time, so
the two agree unless the person changed time zones in between. Screenshots and downloads have no EXIF and do
not count toward the fallback. When neither time exists, the verdict is `cannotCheck`: "Quark can't tell when
these photos were taken."

**The pending count** (`library_pending`) is computed on the phone at report time: the library assets newer than
the newest safe photo it just read from the Quark. Storing it lets the web view say "12 photos waiting on Mom's
iPhone, checked Oct 18" without the phone present.

### Getting an exact capture time onto uploads

Today's mobile upload goes through the file picker, which drops the capture time. Uploads from it are tagged with
their source, but they rely on the EXIF fallback. Exact times need an upload path that starts from a
`photo_manager` asset, such as a "Copy to Quark" action on Mobile tab selections. That is a change to how people
upload, so it is open question 1 and not a ticket below. The design works without it, just less precisely.

## API (additive)

Nothing existing changes shape. An older client sends none of the new fields and gets exactly today's
behavior. **Reporting never fails an upload.** An unknown source, a malformed time, or a write error on the
items table is logged and skipped, and the file still lands. This follows the precedent of the thumbnail
sidecar (#2379).

### Upload additions

- `POST /api/v0/files/upload` and `/files/upload/{rootDir}`: optional query `photoSource=<client id>`.
- Optional multipart part `createdAt` after a file, whose filename is that file's name and whose body is an
  RFC 3339 UTC time, read through `io.LimitReader` with a 64-byte bound. It pairs with its file the way
  `thumbnail` does (`pkg/util/uploadutil/sidecar.go:20`).
- `POST /api/v0/files/upload-session`: optional `photoSource` and `createdAt` fields in the JSON body. The item
  row is written when the session commits.
- The source is looked up as `(caller's user id, client id)`. A client id belonging to another account matches
  nothing, so nobody can attribute uploads to someone else's phone.
- Only files that land as photos (`FileKind` image) get an item row. Every created or overwritten file then
  gets an item row and the source's `last_upload_at` moves. Files that landed before a failed batch count too,
  matching `grantOwners`.

### Source routes, in `internal/server/api/v0/photos/`

| Route | What it does |
| --- | --- |
| `GET /api/v0/photos/sources` | The caller's sources with their summaries (shape below). Never anyone else's, admins included. |
| `PUT /api/v0/photos/sources/{clientId}` | Register or update the calling install: `{label, platform, library?: {access, count?, newestAt?, pending?}}`. Idempotent. `reported_at` is stamped by the Quark when `library` is present. The label is trimmed and bounded to 64 characters. |
| `DELETE /api/v0/photos/sources/{clientId}` | Forget a source and its item rows. The photos stay. Used for a phone that is gone or a reinstall. |

```json
{
  "sources": [
    {
      "clientId": "6f1c…",
      "label": "Mom's iPhone",
      "platform": "ios",
      "current": true,
      "lastUploadAt": "2026-10-18T19:02:11Z",
      "photoCount": 4210,
      "newestCreatedAt": "2026-10-18T18:55:40Z",
      "newestTakenLocal": "2026-10-18T11:55:40",
      "library": {
        "access": "full",
        "count": 4388,
        "newestAt": "2026-10-20T15:01:09Z",
        "pending": 12,
        "reportedAt": "2026-10-20T15:03:00Z"
      }
    }
  ]
}
```

`current` is true for the row whose client id the request named in an optional `?clientId=` query, so the phone
can mark itself. Fields the Quark does not know are omitted rather than zero.

Business logic lives in `pkg/util/photosourceutil` with Params/Result functions: `ListSources`,
`ReportSource`, `ForgetSource`, `RecordUploads` (called by the upload handlers after `grantOwners`), and
`MoveItems` / `DeleteItemsUnder` (called by `fileutil` beside the existing photo-row moves). The handlers only
parse, call, and answer.

### Events

The upload handlers already publish `EventUpload`, which refreshes the view. A report, rename, or forget
publishes a new data-free kind, `photo_sources_changed`, with `Audience: [userID]`, using the audience mechanism
`pkg/util/eventbus` already has for chat. That way another of the person's devices showing the view refreshes,
and no other account hears about it.

## Flutter surface

High level. The tickets carry the detail.

- **Client id.** `lib/utils/photo_source_id.dart` generates 128 random bits (`Random.secure`, hex, no new
  dependency) the first time it is needed, and keeps them in `AppSettings` per Quark host. A reinstall gets a
  new id, and so a new source. That is open question 6.
- **Library probe.** `lib/services/device_library_probe.dart` declares
  `Future<LibraryReport> probe({bool prompt = false})`, split per the platform pattern: `_io.dart` uses
  `photo_manager` on iOS and Android, and `_stub.dart` returns `unsupported` on web and anything else.
  `LibraryReport` is `{access, count?, newestAt?}`. The probe never prompts unless asked.
- **Service.** `lib/services/photo_source_service.dart`: `list`, `report`, `forget`, throwing `ApiException` /
  `throwApiError` as `Errors` expects.
- **Uploads.** `PhotosController` passes `photoSource` on its library uploads (`_uploadToLibrary`,
  `uploadDroppedPhotos`), then reports once after a successful batch.
- **Controller.** `lib/controllers/photo_sources_controller.dart`, a `ChangeNotifier` with injectable service and
  probe functions. On refresh it lists the sources, probes this phone if it is one, computes the pending count
  against this source's newest safe time, reports, and re-lists.
- **Status logic.** `lib/utils/photo_source_status.dart`, a pure function from a source summary, the current
  time, and the local zone to a state plus copy, with tuning in `PhotoSourceConfig` (`staleAfter`). This is where
  the honesty rules live, and so most of the tests.
- **Package widgets** (`packages/quark_widgets`, data in, callbacks out): a `PhotoSourceItem` value type, a
  `PhotoSourceCard` (label, the facts, a status line, rename and forget callbacks), and a `PhotoSourceList` taking
  `isLoading` and `error`, with an empty state. Status is a package enum with the sentence passed in from the app,
  so the package composes no copy. Key prefixes: `photo_source_card_<clientId>`, `photo_source_forget_<clientId>`,
  `photo_source_rename_<clientId>`.
- **Page.** `/photos/sources` (`AppRoutes.photoSources`), reached from a `QuarkBarChip` labeled "Backed up?" on
  the Photos top bar (key `photos_sources_chip`). `AutoRefreshMixin`, `onRefresh` in the app bar slot, a hamburger
  menu like Photos, and `context.go` so a later notification can deep-link to it. On a phone with
  `limited`/`denied`, a "Check this phone" button asks for permission.
- **Errors.** `Errors.message(e, 'load your backup status')`, `'rename this device'`, `'forget this device'`.
  The "can't check" states are not errors and never go through `Errors`.
- **Journeys.** New `JN-PH-0xx` entries in `docs/user-journeys/photos.md` for: seeing a phone that is up to date,
  not yet home, stale, can't check (limited and web), and forgetting a phone.

## Privacy

- **Per account, enforced in the query.** Every read and write filters by the principal's user id. A client id
  from another account matches nothing on upload, report, or forget.
- **Admins see only their own sources here.** This view adds no admin listing. An admin who opens it sees their
  own phones, like anyone else. The existing connected-devices list is unchanged. Whether a household summary
  should exist is open question 3.
- **What is stored:** a label the person can change, a platform, timestamps, counts, and for each uploaded photo
  the capture time and upload time. **What is not stored:** asset ids, file names from the phone's library,
  locations, or anything about photos that were not uploaded beyond the one count and newest time.
- **Account deletion** cascades both tables. Forgetting a source deletes its item rows. The photos themselves
  are never touched.
- **Shared folders.** If someone else deletes a photo the person uploaded into a shared folder, its count drops,
  which is the truth. The summary carries no paths, so it reveals nothing about folders the person can no longer
  read.
- **Events** carry no data and go only to the account (`Audience`).

## Test plan

Backend, run under `make test/unit/backend`, `make test/integration/backend`, and `make test/db/postgres`:

- `photosourceutil`: list is scoped per account (user B never sees A's sources, and an admin sees only their
  own). Count and newest come from present items only. The EXIF fallback joins `photo_hashes`. Forget drops
  items and keeps files. Report bounds the label and validates `access`. Both engines.
- Upload handler integration (`internal/server/api/v0/files`): a tagged multipart upload writes items and moves
  `last_upload_at`. A `createdAt` part pairs by filename and is bounded. Malformed, unknown, or another
  account's source is ignored and the upload still succeeds. Non-photos get no item. A partly failed batch
  records the files that landed. The upload-session path writes on commit.
- `fileutil`: a move re-keys items (file and folder, across devices). Delete and trash drop them.
- Source routes integration: shapes, 404-free idempotent `PUT`, `DELETE` of an unknown id, and the event audience.

Frontend, run under `make test/unit/frontend`:

- `photo_source_status_test.dart`: table-driven over every state, the exact-time and EXIF wall-clock paths
  (including a phone in a different zone than the stored wall clock), an old report, and the web source.
- Probe tests through a fake `photo_manager` adapter: full, limited, denied, empty library, and that no prompt
  happens unless asked.
- Controller tests with injected fakes: refresh order, the pending count, and an error path through `Errors`.
- `PhotosController` upload tests: the `photoSource` query is sent and the report follows a successful batch.
- Package widget tests at 360x640 and 1280x800, a gallery entry, and regenerated `docs.g.dart`.
- Page test: the chip opens the route, and the refresh slot is used.

Optional e2e: a `tests/photo_sources.probe` script on Android with a seeded library.

## Tickets, in landing order

Each is one reviewable PR. A backend API change is stacked with the frontend that consumes it, per AGENTS.md.
`docs/swagger/` is generated output that tickets 3 and 4 both regenerate. The later one regenerates it on rebase
and never hand-merges it. Apart from that generated file, the paths are disjoint.

1. **`feat: store photo sources and the photos uploaded from them`**
   **Difficulty: standard** (the schema and service are fixed here; the work is two-engine care).
   Paths: `internal/db/migrations/040_photo_sources.*`, `internal/db/migrations_postgres/040_photo_sources.*`,
   `sql/queries/photo_sources.sql`, generated `internal/db/` and `internal/db/pgdb/`, `pkg/util/photosourceutil/`,
   and the migration table in `docs/architecture/data.md`.
2. **`feat: keep photo source records through moves and deletes`**
   **Difficulty: mechanical** (call the ticket 1 functions beside the existing photo-row moves).
   Paths: `pkg/util/fileutil/helpers.go`, `pkg/util/fileutil/delete.go`, and their tests. Blocked by 1.
3. **`feat: tag photo uploads with their source`**
   **Difficulty: standard** (two upload paths, and it must never fail an upload).
   Paths: `internal/server/api/v0/files/` (upload handlers, `types.go`, `helpers.go`, a new test),
   `pkg/util/uploadutil/` (the `createdAt` sidecar and session fields), `docs/swagger/`. Blocked by 1.
4. **`feat: list, report and forget photo sources`**
   **Difficulty: standard** (three handlers and an event kind).
   Paths: `internal/server/api/v0/photos/` (three handler files, `types.go`, the package doc in `photos.go`,
   tests), `pkg/util/eventbus/` (the new kind), `docs/swagger/`. Blocked by 1.
5. **`feat: photo source client and install id`**
   **Difficulty: standard**.
   Paths: `lib/services/photo_source_service.dart`, `lib/models/photo_source.dart`, `lib/utils/photo_source_id.dart`,
   `lib/services/app_settings.dart` (one key), `test/services/`, `test/models/`, `test/utils/`. Stacked on 4.
6. **`feat: read the phone library's count and newest date`**
   **Difficulty: standard** (platform permission states).
   Paths: `lib/services/device_library_probe*.dart`, `test/services/device_library_probe_test.dart`. Independent.
7. **`feat: send the photo source with Photos uploads`**
   **Difficulty: mechanical**.
   Paths: `lib/controllers/photos_controller.dart`, `lib/services/files_service.dart` (the query parameter),
   `test/controllers/photos_controller_test.dart`. Stacked on 3 and 5.
8. **`feat: photo source card and list widgets`**
   **Difficulty: standard**.
   Paths: `packages/quark_widgets/lib/src/photos/photo_source_*`,
   `packages/quark_widgets/lib/src/models/photo_source_item.dart`, the barrel, `packages/quark_widgets/test/photos/`,
   `examples/widget_gallery/lib/registry.dart`, generated `docs.g.dart`. Independent.
9. **`feat: show whether each phone's photos are on the Quark`**
   **Difficulty: hard** (the honesty rules and copy are the feature; careful review).
   Paths: `lib/controllers/photo_sources_controller.dart`, `lib/utils/photo_source_status.dart`,
   `lib/utils/photo_source_config.dart`, `lib/pages/photo_sources_page.dart`, `lib/router.dart`, the chip in
   `lib/pages/photos_page.dart`, `docs/user-journeys/photos.md`, and their tests. Stacked on 5, 6, 7 and 8.

Not ticketed until the owner answers: an asset-aware "Copy to Quark" upload (open question 1), a staleness
notification (open question 4), and content-hash verification (open question 7).

## Open questions for the owner

1. **Exact capture times.** Should mobile uploads move to a path that starts from the phone's photo library (for
   example "Copy to Quark" on Mobile tab selections), so every upload carries the capture time the comparison
   needs? Without it, the comparison relies on EXIF wall-clock times and cannot see screenshots or photos without
   EXIF.
2. **Per-photo rows or counters.** This design adds two tables, including one row per uploaded photo, so that
   counts and dates drop when photos are deleted on the Quark. The cheaper alternative, counters on the source
   row (option F), needs no per-photo rows, but goes on calling deleted photos safe. Is the per-photo table
   acceptable under the database-as-last-resort rule?
3. **What admins see.** The proposal: admins see only their own phones. Should an admin also get a household
   summary (each account's phones, last copied, and status, but no photos), or is that too much visibility into
   other people's phones?
4. **When a phone counts as stale, and whether it nags.** The draft is 7 days, in the app only. Should a stale or
   not-yet-home phone also raise a notification in the bell (a third type beside `backup_due` and `backup_stale`)?
5. **Videos.** The Mobile tab reads images only today. Should the comparison include videos, which are the
   biggest and least often copied files on a phone?
6. **New phone or reinstall.** Each install is a new source. Is "forget the old one" enough, or should a new
   install be able to take over an old source's history?
7. **Photos uploaded before this ships.** They have no source and count for no phone. Leave it, or later add a
   "verify" that hashes the phone's newest photos and asks the Quark which content hashes it has
   (`photo_hashes.content_hash`)? That only works when the uploaded bytes match the original exactly, which a
   HEIC-to-JPEG conversion on the way breaks.
8. **Trash.** The proposal counts a trashed photo as not safe at home. Is that the right call while it is still
   restorable?
