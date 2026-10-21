Status: Proposed

<!-- cspell:words davfs GVFS PROPFIND PROPPATCH MKCOL AppleDouble webdavs davs davutil davlockutil apppassutil gowebdav emersion Dolphin rclone NFD NFC FileSizeLimitInBytes BasicAuthLevel getetag getcontentlength lockdiscovery multistatus oplocks inplace httptest ENOSPC GROND -->

# Mounting Quark as a network drive

Design for the epic [#3619](https://github.com/autobutler-org/quark/issues/3619): open your Quark files from macOS
Finder, Windows File Explorer and Linux file managers with nothing extra installed. No code is written until the owner
approves this page or sends it back. Each **Owner call** below is a decision this page lays out with a recommendation
and leaves open; the tickets at the end start from the recommendations, and the owner's answers replace them where
they differ.

## Why the last two attempts died

| Attempt | Added | Removed | What killed it |
| --- | --- | --- | --- |
| WebDAV | #854 | #1559 / #1570, 2026-08-21 | Sign-in was never hardened. #859 (Basic auth), #860 (app passwords) and #861 (token as password) all sat open, and the `/dav` carve-out in `requireAuth` was the riskiest line in the middleware. |
| SMB | #417, #782, #783 | #1703 / #1731, 2026-09-04 | Setup needed root (`apt-get install samba`, `smb.conf`, `smbpasswd`, `systemctl`) and Quark runs as `quark` since #836. Samba was not on the image, the image firewall did not open 445, the share pointed at the internal drive even with USB storage, and #1541 found the setup routes were not admin-only. Samba also read the disk directly, around every permission Quark has. |

Three lessons follow, and every section below is shaped by them:

1. **Nothing outside the `quark` process.** No package to install, no system config to write, no second daemon, no new
   port. If it cannot run as more routes on the server Quark already runs, it does not ship.
2. **Sign-in is the feature, not a follow-up.** The mount ships with its own credential, its lockout and its tests in
   the first release, or not at all.
3. **One permission model.** Every byte goes through `pkg/vfs` and `accessutil`, so the mount can do exactly what the
   app can do for the same person and nothing else.

## What exists today

- **Files.** `pkg/vfs` has one namespace per device: `files` for the internal drive and `files:<serial>` for each USB
  drive (`vfs.FilesNamespace`). `vfs.File` seeks and reads at an offset, `VFS.Write` is atomic (temp file, then rename,
  in `hostWrite`), and a user delete goes through `vfs.Trasher` (`pkg/vfs/vfs.go`).
- **Services the app already calls.** `fileutil.DeleteFiles` trashes and publishes `EventDelete` and
  `EventTrashChanged`; `fileutil.MoveFile` moves, carries access rows with `accessutil.MoveRows` and publishes
  `EventMove`; `fileutil.CreateFolder` publishes `EventNewFolder`; `uploadutil.Destination.WriteFile` streams one file
  in, runs `BeforeOverwrite` (the version snapshot), and publishes `EventUpload`.
- **Access.** `accessutil.Load` reads a principal's grants once per request; `Access.Check(serial, path, level)`
  answers `Readable` and `Allowed`, and resolves symlinks so a link cannot reach outside what was shared. Homes are
  `users/<name>`, group folders `groups/<name>`; `IsHomeRoot` and `IsGroupRoot` refuse moving or trashing them. Admins
  bypass the table (`docs/guides/accounts-and-sharing.md`).
- **Sign-in.** The app derives a 32-byte auth key from the password with Argon2id and never sends the password
  (`docs/auth.md`, #2687). HTTP Basic on `/api/` takes that auth key as the password, behind `LoginGuard`
  (`internal/server/middleware/middleware.go:352`). An OS mount dialog cannot run Argon2id, so today a person would have
  to run `quark auth-key` and paste the result, which is also the key to their whole account.
- **Lockout.** `ratelimitutil.LoginGuard` counts failures per (account, address), per address and per account, with
  backoff, in memory (#1861). #3578 tracks making it persistent and alerting admins. Failures log as
  `sign-in failed ip=…` for fail2ban (`pkg/util/authutil/helpers.go`).
- **TLS.** The server serves HTTPS on 443 with a self-signed leaf certificate that is regenerated yearly and when the
  hostname changes (`pkg/util/tlsutil`, `certValidity = 365 days`). The app pins it on first use (#2721).
- **Remote access.** An embedded tailnet node (`pkg/util/remoteutil`) listens on the tailnet's port 80 and reverse
  proxies plain HTTP over loopback to the local server, setting `X-Forwarded-For` so lockouts key on the real peer.
  WireGuard encrypts the tailnet hop. Loopback is the default trusted proxy (`serverutil.TrustedProxies`). There is no
  public listener and no router port.
- **Discovery.** `quark install` writes an Avahi service file advertising `_quark._tcp` on 443
  (`internal/install/dns_sd.go`).
- **Beta switches.** `featureflagutil` is the registry of admin-controlled beta flags with a sunset issue each.
- **The vault** is per-account ciphertext in `quark.db` (`pkg/util/vaultutil`), not a file tree. No files namespace
  can reach it.
- **Dependencies.** `golang.org/x/net` is already in `go.mod` as indirect (it was direct while the old WebDAV package
  existed), and so is `golang.org/x/text`.

## Summary of the proposal

| Question | Recommendation |
| --- | --- |
| Protocol | WebDAV (RFC 4918, class 1 and 2) served by the Go server itself, on the existing HTTPS port, under `/dav/`. |
| Library | `golang.org/x/net/webdav` for XML, PROPFIND, LOCK and the method plumbing, with Quark's own handlers for PUT, DELETE, MOVE and COPY. **Owner call** (dependency). |
| Credential | Per-device **app passwords**: random, shown once, stored hashed in `quark.db`, scoped to the mount (read-only or read-write), revocable one by one, never accepted on `/api/`. The account password and auth key are never accepted on `/dav/`. **Owner call** (auth model). |
| Lockout | Every `/dav/` sign-in goes through the same `LoginGuard` and log line as the sign-in page. A revoked password is recognized and refused without counting as a guess, so a forgotten mount cannot lock its owner out of the app. |
| Permissions | The mount shows the real tree, filtered by `accessutil` exactly as the app's Files page does. An admin sees what **All files** shows. |
| Delete | Moves to the trash through `fileutil.DeleteFiles`. Nothing on the mount deletes for good. |
| Events | Every change goes through the same `fileutil` / `uploadutil` service the app calls, so the same events fire. |
| Transport | HTTPS, or the tailnet hop. Plain HTTP on the LAN is refused before any credential is asked for. |
| Default | Off. A beta feature flag an admin turns on; with it off, `/dav/` answers 404. **Owner call**. |
| Windows | Works over HTTPS only with a certificate Windows trusts. **Owner call** (certificate) decides whether Windows is in the first release. |

## Protocol

### Options

| Option | No root, no daemon | Built into every OS | Goes through `pkg/vfs` | Verdict |
| --- | --- | --- | --- | --- |
| **WebDAV in the Quark process** | Yes: routes on the existing server | Finder, Explorer (WebClient), GVFS, KIO, davfs2 | Yes | **Recommended** |
| SMB through Samba | No: root to install and configure, a daemon, port 445 | Yes | No: Samba reads the disk | Removed once (#1703); the same blockers stand |
| SMB in Go, in process | Port 445 is privileged; the unit has `CAP_NET_BIND_SERVICE`, so binding is possible | Yes | Yes | No maintained Go SMB server library; SMB2/3 signing, oplocks and leases are a large, security-critical surface to own. Not now. |
| SFTP in process | Needs a second port and the SSH stack | No: Finder and Explorer cannot mount SFTP without extra software | Yes | Fails "nothing to install" |
| NFS | Kernel server, root, export files | Not on Windows Home | No | No |
| A Quark sync client | No daemon on the Quark | No: something to install on every computer | Yes | A different feature; the app already uploads |

WebDAV is the only option that is both built into the three desktop OSes and possible from an unprivileged process
on a port Quark already serves. It is also plain HTTPS, so it rides the remote-access connection with no new
listener.

### Library

`golang.org/x/net/webdav` (checked at v0.60.0, the version already in the module graph) gives `Handler`,
`FileSystem`, `LockSystem` with an in-memory `NewMemLS`, PROPFIND and PROPPATCH, LOCK and UNLOCK, and answers
`DAV: 1, 2`, which Finder needs before it mounts read-write. Reading its source shows gaps that matter here:

- **PUT can commit a partial file.** `handlePut` copies the body into the file, then closes it whatever the copy
  returned (`webdav.go:294-296`). An adapter that commits on `Close` keeps a truncated upload. Quark serves PUT itself.
- **PUT ignores `If-Match` / `If-None-Match`** (a `TODO` in `handlePut`) and answers 201 even for an overwrite.
- **DELETE calls `FileSystem.RemoveAll`.** Quark needs a trash with `TrashedBy`, so DELETE is Quark's own.
- **MOVE is `FileSystem.Rename`** and COPY recurses through `OpenFile`. Neither carries access rows nor publishes, so
  both are Quark's own, calling `fileutil.MoveFile` and `VFS.Copy`.
- **PROPFIND accepts `Depth: infinity`**, a walk of the whole tree for one request. Quark refuses it with 403 and the
  RFC's `propfind-finite-depth` precondition, which every client listed here handles by falling back to depth 1.
- **No quota properties** (`quota-available-bytes`, `quota-used-bytes`). Finder and Explorer show free space from them.
  They can be added on our side; it is a nice-to-have.
- **Locks are global and token-only.** `NewMemLS` lets anyone holding a token unlock it. Quark wraps it (see Locks).

So the library carries the protocol (XML, multistatus, `If:` header parsing, lock bookkeeping) and Quark keeps every
method that changes data. The alternatives:

| Option | For | Against |
| --- | --- | --- |
| **`golang.org/x/net/webdav`, Quark handles the writing methods** | Already in the module graph as indirect, so promoting it adds no new module. Go team maintained, used by the removed implementation. Ships a litmus test server. | Slow-moving; the gaps above. |
| `github.com/emersion/go-webdav` | Cleaner API; also has CalDAV and CardDAV, which a calendar sync could use later. | A new module. Its README does not list LOCK support, and Finder mounts read-only without class 2; to be confirmed before choosing it. |
| Hand-written WebDAV on gin | No dependency at all. | Several thousand lines of RFC 4918 XML and `If:` header parsing to write and keep correct, which is where most WebDAV servers have their bugs. |

**Recommendation:** promote `golang.org/x/net` to a direct requirement for its `webdav` package, and serve PUT,
DELETE, MOVE, COPY and MKCOL from Quark's own handlers. **Owner call:** approve the direct dependency, or pick another
row. Two smaller dependency questions come with this design and are flagged where they arise: Unicode normalization
(`golang.org/x/text/unicode/norm`, already indirect) and a WebDAV client for the test suite.

## Where it lives

- **URL.** `https://<quark>/dav/` on the existing port. Outside `/api/` on purpose: `requireAuth` only guards `/api/`,
  and `/dav/` must accept a different credential and refuse the ones `/api/` takes. Its own middleware chain is the
  only one on the group, so there is no carve-out in `requireAuth` like the one #1570 removed.
- **Drives.** `/dav/files/` is the internal drive; `/dav/drives/<serial>/` is each attached USB drive, one per
  `files:<serial>` namespace. The Settings card shows each as its own address. A Finder or Explorer mount is one URL, so
  a second drive is a second mount rather than a folder that disappears when the drive is unplugged.
- **Packages.** The router is `internal/server/api/v0/dav/` (package `v0_dav`), the directory the old implementation
  used, so `scripts/check-go-structure.bash` applies to it. The logic sits under `pkg/util/`, with Params/Result
  services and dependencies from `deputil`:
  - `pkg/util/apppassutil/`: create, list, revoke and authenticate app passwords.
  - `pkg/util/davutil/`: the `webdav.FileSystem` over `vfs` and `accessutil`, the writing methods, name rules and
    hidden paths.
  - `pkg/util/davlockutil/`: the `webdav.LockSystem` wrapper.
- **No new `Dependencies` field** beyond the lock system, which goes on `deputil.Dependencies` behind a
  `WithDAVLocks(...)` builder like the other stores.

## Authentication

Native clients speak HTTP Basic and nothing else useful: Finder, Explorer, GVFS and davfs2 store a username and a
password and send them on every request. Digest needs a plaintext-equivalent secret on the server, and none of them
can do OAuth or cookies. So the question is what the password is.

### Options

| Option | Revocable per device | Survives a stolen laptop | Typeable in an OS dialog | Verdict |
| --- | --- | --- | --- | --- |
| The account password (#859) | No | No: the OS keychain holds the account | Yes | Refused. The server never sees the password since #2687, and the epic forbids it. |
| The account auth key (what `/api/` Basic takes today) | No: only a password change | No: it is the account, chat wrap aside | No: 44 base64 characters from `quark auth-key` | Refused on `/dav/`. |
| **App passwords (#860)** | Yes, each one | Yes: revoke that one; it never reaches `/api/`, settings or the vault | Yes, if the format is built for typing | **Recommended** |
| Long-lived API token as password (#861) | Yes | Only if scoped | Barely | The same mechanism as app passwords with a worse format. Folded in: an app password *is* a scoped token. |

### App passwords

- **Made in the app.** Settings → Network drive → **New app password**, with a label ("MacBook", "Work PC") and a
  scope. The Quark generates it, the app shows it once with a copy button, and only its hash is kept.
- **Making one asks for the password again.** The create request carries the caller's auth key, as
  `PUT /auth/password` and `PUT /auth/recovery-key` do, so a stolen session cannot turn itself into a credential that
  outlives the session. A wrong key is 403 and counts toward the guard. Listing and revoking need only the session.
- **Format.** 20 characters from a 32-letter alphabet with no letters that look alike, shown in groups of four with dashes
  (`k7mq-…`): 100 bits, typeable from a phone screen into a Windows dialog. The dashes are ignored on input.
- **Storage.** A new `app_passwords` table in `quark.db`: id, user id, label, scope, `sha256:` hash, created, last
  used, revoked. A credential cannot live on the data drive (#1861: a drive moved to new hardware must not carry
  sign-in state), so this is what the database is for. A fast hash is right for a random 100-bit secret, as #2765
  reasoned for auth keys; it is compared in constant time.
- **Lookup.** Hash the presented password and find the row by hash, then check the row's user is the Basic username.
  One indexed read per request, no bcrypt per request.
- **Last used** is written at most once a minute per password, so a busy mount does not write on every request.
- **Scopes.** `read` or `read-write`, for the mount only. A read-only password gets 403 on every method that writes.
  Nothing else: an app password never authenticates `/api/`, the events stream, the vault, or settings, because only
  the `/dav/` middleware looks at the table.
- **Revocation.** Each one is revoked on its own from the same card. All of an account's app passwords are revoked
  when the account recovers with its phrase (`/auth/recover`, which already signs out every session) and when an admin
  disables or deletes the account. A password change does not revoke them; independence from the password is the
  point. A disabled account's passwords stop working at once, because the middleware checks the account on every
  request as it does for sessions.
- **Usernames** are compared lowercased; Quark usernames are lowercase and Windows sends what was typed.
- **The username is the account's.** An app password does not get a username of its own, so the dialog asks for the
  same username the person signs in with.

### Lockout and rate limiting

- **Same guard, same log line.** The `/dav/` middleware calls `LoginGuard.Check` before looking anything up, and
  `RecordFailure` / `RecordSuccess` after, keyed by the Basic username and the client address from `c.ClientIP()`. A
  locked-out request gets 429 with `Retry-After`. A failure logs through the same `loginFailed` helper, so the line
  fail2ban matches is the same. #3578's persistent lockouts and admin alerts land in the guard and apply here with no
  second change; the alert should say the attempt came through the network drive.
- **A forgotten mount must not lock out its owner.** Finder and Explorer retry a rejected password many times, in
  parallel. Counted as guesses, a laptop with a revoked password would lock the account from that address, including
  the app's sign-in. So a revoked password stays in the table (hash, revoked time) for 90 days and is answered 401
  without counting as a failure: it is not a guess, its hash is already known to have been issued. An unknown password
  counts normally.
- **No `/api/` rate-limit token is spent**, as for Basic on `/api/` today: the guard bounds the guessing, and a
  100-bit secret is not guessable inside it.

### Transport

Basic sends the secret on every request, so it only travels encrypted:

- **HTTPS** on the LAN: accepted.
- **The tailnet hop**: accepted. It arrives as plain HTTP over loopback from the in-process proxy, after WireGuard.
  The proxy marks it with a header the middleware believes only from loopback (`X-Quark-Transport: tailnet`, set in
  `remoteutil.newProxy` after stripping any inbound copy); `X-Forwarded-Proto` cannot serve, because the tailnet leg is
  HTTP.
- **Loopback** itself (development, `make watch/backend`): accepted.
- **Anything else on plain HTTP**: 403 with a sentence saying to use the `https://` address, and no
  `WWW-Authenticate` challenge, so a client is never prompted to send a password in the clear. A client that sends
  Basic unprompted over plain HTTP gets the same 403 and that request is not authenticated.
- **Cookies and bearer tokens are ignored on `/dav/`.** A browser attaches the session cookie by itself, so honoring it
  would let any page the person visits drive their files with PROPFIND and DELETE. `/dav/` sends no CORS headers.

## Permissions and per-user roots

### Options

| Option | For | Against |
| --- | --- | --- |
| **The real tree, filtered by `accessutil`** | The same paths as the app's breadcrumbs and links; nothing new to keep consistent; shares, groups and admins already work. | A member mounting `/dav/files/` sees `users` and `groups` at the top and opens `users/<name>` to reach their files. |
| A virtual root (My files, Groups, Shared with me) | Matches the app's shortcuts. | Virtual folders cannot be written to, moved into or renamed; a second tree to keep in step with the app; two shares with the same name collide. |
| Each user's home as the mount root | One click less. | Group folders and shares are out of reach, unless they become a second mount. |

**Recommendation: the real tree.** A member sees `users` holding only their own home plus homes holding something
shared with them (`Access.Visible`, the same rule the Files listing uses), `groups` holding their groups and
`everyone`, and anything else on the drive only if it was shared with them. The Settings card offers the root as the
address and the home, `https://<quark>/dav/files/users/<name>/`, as a second one for people who only want their own
files. An admin's mount shows what **All files** shows, because it is the same person with the same rights; an admin
who wants less makes a read-only password or mounts their home.

How each request is checked:

1. The path is canonicalized with `accessutil.Canonical`, so `..`, `%2e%2e`, doubled slashes and a trailing slash all
   name one row and cannot climb out. The VFS refuses an escaping path too (`ErrPermissionDenied`); two guards.
2. `accessutil.LoadRequest` once per request, then `Access.Check(serial, path, level)`: `Read` for GET, HEAD and
   PROPFIND; `Write` for PUT, MKCOL, DELETE, PROPPATCH and LOCK, and for both ends of MOVE and the destination of COPY.
3. **Not readable is 404, not 403**, as `delete_files.go` answers, so a guessed path says nothing about whether it
   exists. Readable but not writable is 403.
4. A PROPFIND listing drops children `Access.Visible` hides (`accessutil.VisibleChildren`), so other homes never appear.
5. `IsHomeRoot` and `IsGroupRoot` refuse DELETE and MOVE of a home or group folder for everyone, as the app does
   (JN-FB-037); `users` and `groups` themselves are not writable.
6. New files and folders get `accessutil.GrantOwnerIfNeeded`, as an app upload does, so something made in a folder
   shared with you is yours to manage.
7. A USB drive that is not attached is 404 for its whole prefix, as `Access.Check` already says.

**The vault is not reachable** because the adapter only resolves `files` and `files:<serial>` namespaces from the
registry, never any other registry entry, and the vault is not a namespace at all. A test proves no `/dav/` path
reaches a vault item.

**Hidden paths.** The trash (`.trash`, `storageutil.IsTrashPath`), the version store (`.quark-versions`) and Quark's
own probe and staging files are absent from every listing and 404 for every method. The trash is reached in the app.

## Writing, deleting and events

| Method | Calls | Publishes |
| --- | --- | --- |
| PUT | `uploadutil.Destination.WriteFile` with the body as the reader, `Overwrite: true`, the version snapshot in `BeforeOverwrite` | `EventUpload` |
| MKCOL | `fileutil.CreateFolder` | `EventNewFolder` |
| DELETE | `fileutil.DeleteFiles` with `TrashedBy` | `EventDelete`, `EventTrashChanged` |
| MOVE | `fileutil.MoveFile` (carries access rows with `accessutil.MoveRows`, across drives too) | `EventMove` |
| COPY | `VFS.Copy` / `vfs.CopyBetween` per file, under a bound on entries | `EventUpload` |
| PROPPATCH | Accepted for `Win32LastModifiedTime` and friends, applied as the file's mtime where the VFS can, otherwise accepted and dropped | none |

- **Deletes go to the trash**, restorable in the app until the retention runs out, exactly like a delete in Files.
- **Overwrite keeps a version.** A PUT over an existing file, and a MOVE with `Overwrite: T` onto an existing file
  (how Office and many editors save: write a temp file, then rename it over the original), snapshot the replaced file
  through the same version path an app overwrite takes, rather than trashing it. Otherwise every save would add a
  trash item. **Owner call** only if versions are not wanted here.
- **Events stay in step with the app,** so thumbnails, the photo index and search keep up, because the mount calls
  the same services. A Finder copy of a thousand small files is a thousand `EventUpload`s; the services already publish
  one per upload, so this is no worse than an app upload of the same set. Coalescing them is a measured follow-up, not
  part of the first release.

## Locks

Finder mounts read-only unless the server is class 2, and Office will not save without LOCK. So LOCK and UNLOCK
ship in the first release.

- **One lock table per Quark, owned per user.** `davlockutil` wraps `webdav.NewMemLS` and records which user took
  each token. A lock blocks other users' writes to that file, which is its purpose; but a token is only honored for the
  user who took it: `Confirm`, `Refresh` and `Unlock` with someone else's token behave as if the token did not exist
  (412 / 409), so a token read from `lockdiscovery` or a log is useless to anyone else.
- **Owner details stay private.** `lockdiscovery` shows another user's lock without the `owner` XML the client sent,
  which can carry a full name.
- **They expire.** Timeouts are capped at 10 minutes and clients refresh; `Infinite` becomes the cap. A user holds at
  most 1000 locks. Locks live in memory and a restart drops them, which clients recover from by locking again.
- **The app does not honor them.** A file locked by Office on a mount can still be overwritten from the app. That is
  acceptable for a first release and is said in the docs. Honoring DAV locks in the app's upload path is a follow-up.
- **Lock-null resources.** LOCK on a missing name creates an empty file (RFC 4918), which Finder does before its PUT.
  It goes through the same access check, owner grant and event as a PUT.

## Large files, streaming and memory

- **PUT streams the body into `VFS.Write`**, which writes a temp file and renames it, so nothing is held in memory and
  a dropped connection leaves no file behind: an early end of the body is `io.ErrUnexpectedEOF` from the reader, the
  write fails, and the temp file is removed. A test drops a connection halfway and finds no file and no temp file.
- **Length.** `Content-Length` sets `WriteOptions.ExpectedSize`. Finder sends chunked bodies with
  `X-Expected-Entity-Length`; that is used the same way, never trusted beyond being a hint for the space check.
- **Space.** When the length is known, a PUT that would not fit in the drive's free space answers 507 Insufficient
  Storage before reading the body; one that runs out midway (`ENOSPC` from the write) answers 507 and leaves nothing.
  The app's upload path has no such check today, so this is new, and it belongs in `uploadutil` where both can use it.
- **GET and HEAD** serve the `vfs.File` through `http.ServeContent`: ranges, `If-Range`, `If-Modified-Since`, no
  buffering. `ETag` is the VFS content hash when it has one, otherwise size and mtime, and PUT honors `If-Match` and
  `If-None-Match`.
- **Bodies that are not file content are bounded.** PROPFIND, PROPPATCH and LOCK bodies go through
  `http.MaxBytesReader` (1 MiB). Go's `encoding/xml` expands no external entities, so XXE is not a concern.
- **Timeouts.** `serverutil.NewHTTPServer` sets no read or write timeout, only header and idle ones, so a long
  transfer is not cut off. Nothing to change.
- **Windows' 50 MB limit.** Explorer refuses downloads over `FileSizeLimitInBytes` (50,000,000 bytes by default) with
  error 0x800700DF. The server cannot change it. The Settings card and the docs say so and give the registry value;
  this is the one place the docs touch the registry, and it raises a limit rather than weakening security. Quark never
  tells anyone to change `BasicAuthLevel`.

## Names across macOS, Windows and Linux

The Quark's disk is case-sensitive ext4 (USB drives may be exFAT or NTFS). Finder and Explorer are case-insensitive
and case-preserving.

- **Case.** Creating a name that differs only in case from a sibling (`Report.docx` beside `report.docx`) is refused on
  `/dav/` with 409. A rename that only changes the case of the same item is allowed. Existing pairs, made in the app or
  on Linux, are listed as they are; Explorer and Finder may open the wrong one, and the docs say so. The app is not
  changed.
- **Unicode normalization.** A name typed on one OS can arrive composed (NFC) and the same name from another
  decomposed (NFD), which makes two files that look identical. New names are stored NFC; a lookup that misses tries the
  NFC form of the requested name before answering 404. This needs `golang.org/x/text/unicode/norm`, already an indirect
  requirement; **flagged** for the dependency call. Which clients send NFD today is a test to run, not an assumption.
- **Windows-reserved names and characters** (`\ : * ? " < > |`, trailing dot or space, `CON`, `NUL`, `COM1` …) are
  legal on the Quark and pass through untouched. Explorer may fail to open them; the docs say so. No name is rewritten.
- **Percent-encoding.** Every `href` is percent-encoded per segment; names with `%`, `#`, `+`, `?`, spaces and emoji are
  in the test set.

### OS metadata files

Finder writes `._<name>` (AppleDouble: Finder tags, quarantine and other extended attributes) next to files and
`.DS_Store` in folders; Explorer writes `desktop.ini` and sometimes `Thumbs.db`.

| Option | For | Against |
| --- | --- | --- |
| Refuse them (403) | Nothing to hide | Finder copies can fail (error -36) when `._` writes are refused, and tags are lost |
| Keep them in memory only | Nothing on disk | Lost on restart; Finder tags vanish |
| **Store them, hide them from Quark** | Finder and Explorer round-trip what they wrote | One shared rule every listing, index and search must apply |

**Recommendation: store them and hide them.** One predicate, `storageutil.IsOSMetadataName`, is applied by the
Files listings, the photo and file indexes, search, thumbnails and the recent list, and the mount itself publishes no
event for them. They are still real files: they move and trash with their folder, and they count toward space. The
docs mention `dot_clean` and Finder's option to stop writing `.DS_Store` to network volumes as optional tidying, never
as a setup step.

## TLS, certificates and remote access

### The LAN

| Client | Quark's self-signed certificate today |
| --- | --- |
| macOS Finder | Asks once whether to trust it; the choice can be kept in the Keychain. Works. |
| GNOME Files (GVFS), KDE Dolphin (KIO) | Ask once. Work. |
| davfs2 | Asks on the first mount, or the certificate is placed in its trust directory. Works. |
| Windows Explorer (WebClient) | Fails with no prompt. Works only once the certificate is installed in the computer's trusted root store, which needs an administrator on the PC. |

Two facts make Windows worse than a one-time import: the certificate is a leaf regenerated every year and on every
hostname change (`tlsutil`), so a Windows import breaks yearly; and it names only `localhost`, the hostname, its
`.local` name and the machine's addresses, so it fails for any other name the box is reached by. Separately, Microsoft
lists the WebClient service as deprecated. It still ships, but Windows support rests on it, and its status should be
checked again before Windows work starts.

**Owner call: the Windows certificate.**

| Option | Effort | For | Against |
| --- | --- | --- | --- |
| A. Ship Windows on the LAN with a documented certificate import | Docs only | No infrastructure | An administrator step for non-power users, repeated yearly; the docs page is long |
| B. A long-lived per-Quark root, made once, that signs the yearly leaf | `tlsutil` change; the app's pinning (#2721) must pin the root, not the leaf | Import once, ever | Still an import; a private root on the PC can sign for any name unless name-constrained, so it must carry a name constraint to the Quark's own names |
| C. A publicly trusted certificate per Quark (a project-run DNS name per device, DNS-01) | A hosted service and per-device enrollment | Works everywhere with no prompt | New infrastructure the project runs; out of scope for this epic |
| D. macOS and Linux first; Windows in a later release | None now | Ships the parts that work well | Windows users wait |

**Recommendation: D, then B.** Ship for macOS and Linux, where the self-signed certificate works with a prompt; make B
its own ticket, because it changes the app's certificate pinning and deserves its own review; then turn on Windows
with a one-time import. C is the right end state if the project ever runs a DNS service per device.

### Remote access

- **Only through remote access.** The mount is reachable from outside the home only through the tailnet node already
  running in-process. No new listener, no public address, no router port; #3578's remote-access checks cover `/dav/`
  because it is the same listener. Lockouts key on the tailnet peer's address, which the proxy already forwards.
- **macOS and Linux** connect to the tailnet address over `http://`; the hop is encrypted by the tailnet, and the
  middleware accepts Basic there (see Transport). Finder shows its warning about an unencrypted connection, which the
  docs explain.
- **Windows** refuses Basic over `http://` at its default `BasicAuthLevel`, so remote Windows mounts need HTTPS on the
  tailnet with a certificate Windows trusts: the same certificate call above.
- **User-facing copy** says "remote access", names no vendor, and calls remote mounting "coming soon" until remote
  access is confirmed for friends and family (quark-project #74).

## Discovery

`quark install` adds a `_webdavs._tcp` record on 443 with TXT `path=/dav/files/` beside `_quark._tcp`
(`internal/install/dns_sd.go`), only when the feature flag is on. What each OS does with it, to be confirmed in
testing rather than promised:

- **GNOME Files and KDE Dolphin** browse DNS-SD and list WebDAV services under their network views.
- **Finder** lists SMB and AFP services in its sidebar, not WebDAV, so the Settings card and ⌘K are the route.
- **Explorer** does not browse DNS-SD for WebDAV at all. Map network drive with the address from the card.

## Settings

- **Admin switch.** A `featureflagutil` beta flag, `networkDrive`, off by default, with its own sunset issue. Off,
  `/dav/` answers 404 and the card says an admin can turn it on.
- **The Network drive card** (every account, when the flag is on): the address for each OS with copy buttons, the
  account's username, the list of app passwords (label, scope, created, last used) with revoke, and **New app
  password**. It shows the Windows notes (certificate, 50 MB limit) and, once #1655's doctor endpoint exists, why a
  connection would fail (flag off, plain HTTP, no remote access).
- **Journeys.** New `JN-ST-` journeys for the card; `JN-ST-019` stays retired.

## Threat model

| Threat | Mitigation |
| --- | --- |
| Someone on the LAN guesses passwords | 100-bit app passwords; `LoginGuard` lockout per pair, address and account; fail2ban-ready log line |
| A laptop with a saved mount is stolen | Revoke that one password; it reaches only the mount, at the scope it was made with; never `/api/`, settings or the vault |
| The OS keychain leaks the password | Same as above; the account password and auth key are never stored by the OS |
| A web page drives `/dav/` with the person's cookie | `/dav/` ignores cookies and bearer tokens and sends no CORS headers |
| A household member reads or writes another's files | `accessutil.Check` on every path, 404 for unreadable, filtered listings, home and group roots protected |
| Path traversal, encoded dots, symlinks | `accessutil.Canonical`, the VFS's own escape check, and `Access.Check`'s symlink resolution |
| A lock token is reused by another user | Tokens are honored only for the user who took them; owner XML is hidden from others |
| A stale mount locks its owner out of the app | Revoked passwords are recognized and not counted as guesses |
| Credentials sniffed on the wire | HTTPS or the tailnet only; plain HTTP is refused before any challenge |
| Resource exhaustion | `Depth: infinity` refused, XML bodies capped at 1 MiB, locks capped per user, COPY bounded, space checked before a PUT, PUT streamed |
| The mount is exposed to the internet | No new listener; remote only through the tailnet node, covered by #3578's checks |
| The vault leaks through the mount | Not a namespace; the adapter resolves file namespaces only; tested |
| A half-written file after a dropped upload | Atomic `VFS.Write`; tested |
| A brute-force lockout lost on restart | Today's behavior for every sign-in path; #3578 fixes it in one place for all of them |

Out of scope, and said so: a client computer that is already compromised can do whatever its mount can do, at the
scope of its password. That is why the read-only scope exists.

## Test plan

Everything runs in `make test/integration/backend` (handlers) or `make test/unit/backend` (services) with no network
and no root.

- **Unit, `pkg/util/apppassutil`:** create shows the secret once and stores only the hash; authenticate by hash and
  username; dashes and case ignored; revoked passwords refused without a guard failure; recovery and account disable
  revoke all; `last_used` throttled; read-only scope recorded.
- **Unit, `pkg/util/davutil`:** over `vfs.MemVFS` and a temp-dir `LocalVFS`: access filtering, 404 versus 403,
  home and group roots, hidden paths, case collisions, NFC lookup, metadata names, COPY bounds.
- **Unit, `pkg/util/davlockutil`:** a token is only honored for its user; owner XML hidden; timeout cap; per-user cap.
- **Integration, `internal/server/api/v0/dav`:** a real gin engine on an `httptest` TLS server and a real filesystem,
  driven by a WebDAV client. Cases: sign-in with an app password; refusal of the auth key and of a session cookie;
  revoke takes effect on the next request; lockout after wrong passwords, with 429 and `Retry-After`, counted toward
  `/auth/login` from the same address; a revoked password not counted; per-user isolation, including guessed and
  encoded paths; household sharing at each level; vault unreachable; DELETE lands in the trash and restores from the
  app's trash API; MOVE carries a share; events published for each method; LOCK, a second user's write refused with
  423, another user's token refused, expiry; a multi-gigabyte streamed PUT with the heap measured, as #1705 did; a
  connection dropped mid-PUT leaving nothing; 507 on a full disk; range GET; plain HTTP refused with no challenge; the
  tailnet marker accepted only from loopback; read-only scope; flag off answers 404.
- **The client in those tests.** Hand-built `net/http` requests need no dependency and give exact control over methods
  and headers, so the suite starts there. `github.com/studio-b12/gowebdav` is already in `go.sum` as a transitive
  entry; using it would make it a direct test dependency. **Flagged**, not decided.
- **Compliance.** The `litmus` suite run against a dev server, as a documented manual step for each release that
  touches `/dav/`, and in CI once the owner agrees to install it there. `golang.org/x/net/webdav` ships a litmus test
  server, which shows the expected baseline.
- **Real clients,** by hand for each release that touches `/dav/`, recorded in the PR: Finder (mount, copy in and out,
  rename, delete, Office and TextEdit save, tags survive); Explorer (map drive, 50 MB limit message, Office save, long
  paths, after the certificate work); GNOME Files and KDE Dolphin (`davs://`, DNS-SD listing); davfs2 (mount, `cp`,
  `rsync --inplace`, `vim` save). Remote: the same from Finder and GNOME Files over remote access.

## Tickets

In landing order. Each is one reviewable PR with its own paths; a backend API change is stacked with the frontend that
consumes it, as AGENTS.md asks. None is filed until the owner approves this page.

1. **Hide OS metadata files from Quark.** `storageutil.IsOSMetadataName` and its use in Files listings, the recent
   list, the file and photo indexes, search and thumbnails. Useful on its own, since such files arrive by USB drive
   too. Paths: `pkg/util/storageutil/os_metadata.go`, and the listing and index call sites in `pkg/util/fileutil/`,
   `pkg/util/indexutil/`, `pkg/util/searchutil/`, `pkg/util/thumbnailutil/`.
   **Difficulty: standard (one predicate, several call sites to find).**
2. **App passwords, backend.** Migration pair in `internal/db/migrations/` and `internal/db/migrations_postgres/` at
   the next free number, `sql/queries/app_passwords.sql`, regenerated `internal/db/`, `pkg/util/apppassutil/`, and
   create, list and revoke routes under `/api/v0/auth/app-passwords` as new handler files in
   `internal/server/api/v0/auth/`, plus revocation on recover and on account disable. No `/dav/` yet.
   **Difficulty: hard (a new credential and its lockout interplay).**
3. **App passwords and Network drive card, frontend.** Stacked on 2. `lib/services/app_password_service.dart`,
   `lib/controllers/network_drive_controller.dart`, the card and the password list in
   `packages/quark_widgets/lib/src/settings/` with tests and gallery entries, the settings page wiring, new `JN-ST-`
   journeys. Until ticket 7 adds the flag, the card lists app passwords and says the drive is not available yet. **Difficulty: standard.**
4. **The DAV filesystem.** `pkg/util/davutil/`: the `webdav.FileSystem` over `vfs` and `accessutil`, the writing
   methods calling `fileutil` and `uploadutil`, versions on overwrite, hidden paths, name rules, COPY bounds. Promotes
   `golang.org/x/net` (and `golang.org/x/text` if approved) to direct in `go.mod`. Unit tests only.
   **Difficulty: hard (permission mapping and write semantics).**
5. **Per-user locks.** `pkg/util/davlockutil/` and the `WithDAVLocks` builder in `pkg/util/deputil/`.
   **Difficulty: standard.**
6. **Tailnet transport marker.** `remoteutil.newProxy` strips any inbound `X-Quark-Transport` and sets it to
   `tailnet`; a helper in `pkg/util/serverutil/` believes it only from loopback. Paths: `pkg/util/remoteutil/helpers.go`,
   `pkg/util/serverutil/transport.go`. **Difficulty: mechanical.**
7. **The `/dav/` endpoint.** `internal/server/api/v0/dav/`, its own middleware chain (app passwords only, transport
   check, `LoginGuard`, no cookies, no CORS), the `networkDrive` flag in `pkg/util/featureflagutil/`, PROPFIND depth
   refusal and body caps, the mount in `internal/server/routes.go`, and the integration suite above.
   **Difficulty: hard (security boundary).**
8. **Discovery.** `_webdavs._tcp` in `internal/install/dns_sd.go`, written only when the flag is on, with its test.
   **Difficulty: mechanical.**
9. **Security review (GROND)** of 2, 4, 5, 6 and 7 together before the flag is offered to anyone: sign-in, lockout with
   #3578, isolation, traversal, vault, transport, remote. No paths; findings become tickets.
   **Difficulty: hard (careful review).**
10. **Compliance and client matrix.** The litmus run as a documented step and the manual client checklist, in
    `docs/architecture/network-drive.md` (where this page's decisions move once built).
    **Difficulty: standard.**
11. **User docs (Bilbo).** Site pages for macOS, Windows and Linux with screenshots, the Windows notes, and how to
    revoke an app password; Settings help text. **Difficulty: standard.**
12. **Windows certificate** (only if the owner picks B): a long-lived, name-constrained per-Quark root in
    `pkg/util/tlsutil/`, the app's pinning moved to the root, the Windows import in the docs, then Windows enabled in
    the card. **Difficulty: hard (TLS and the app's pinning).**

## Open questions for the owner

1. **New dependency.** Promote `golang.org/x/net` to a direct requirement for `webdav`, or choose
   `github.com/emersion/go-webdav` or a hand-written server? And may `golang.org/x/text/unicode/norm` (NFC names) and
   `github.com/studio-b12/gowebdav` (test client) become direct, or should both be avoided?
2. **Auth model.** App passwords as described: per device, two scopes, revoked on recovery and account disable but not
   on a password change, and never valid on `/api/`. Is a revoked password's 90-day "not a guess" grace acceptable, and
   should an admin's app password be able to see **All files**, or only what a member would see?
3. **Off by default.** A beta flag an admin turns on, off on every Quark until the security review closes. Or on by
   default once it leaves beta?
4. **Windows certificate.** Ship macOS and Linux first and add a per-Quark root later (recommended), document a yearly
   import of today's certificate, or wait for a publicly trusted certificate?
5. **Overwrite.** Version the replaced file on PUT and on MOVE over an existing file (recommended), or send it to the
   trash, or neither?
6. **CI.** May the litmus suite, and later a davfs2 mount, run in CI, or do they stay manual release steps?
