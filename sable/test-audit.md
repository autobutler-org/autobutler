# Test Audit — AutoButler Backend

> Sable, 2026-03-17. Based on `go test ./pkg/... -coverprofile` run on main.

## Summary

**Overall coverage: 30.3%** — low, but misleading. The biggest untested packages are
`botel` (observability/tracing) and the API handlers layer, which have zero tests.
The core utility packages that actually move files and data are better covered but have
meaningful gaps.

---

## Coverage by Package

| Package | Coverage | Notes |
|---------|----------|-------|
| `ctxutil` | 100% | ✅ |
| `maputil` | 100% | ✅ |
| `stringutil` | 100% | ✅ |
| `bookutil` | 93.8% | ✅ good |
| `serverutil` | 85.7% | ✅ good (+ HttpError tests I added) |
| `photoutil` | 79.8% | ✅ decent |
| `versionutil` | 80.0% | ✅ decent |
| `workerutil` | 69.2% | ⚠️ `GetBackupToDeviceChannel` = 0% |
| `storageutil` | 52.4% | ⚠️ several critical paths at 0% |
| `updateutil` | 44.2% | ⚠️ core update path mostly untested |
| `deputil` | 47.6% | ⚠️ `DefaultDependencies`, `WithWorker`, `Worker` = 0% |
| `githubutil` | 30.6% | ⚠️ `FetchLatestRelease` = 0% |
| `usbutil` | 0.0% | ❌ no tests at all |
| `botel` | 0.0% | ❌ no tests (observability infra) |
| `botel/exporters/botelsqlite` | 0.0% | ❌ no tests |
| `calendar` | 0.0% | ❌ no tests |
| `migration` | 0.0% | ❌ no tests |

**API handlers (`internal/server/api/v1/`):** zero test files across all handlers.

---

## Critical Gaps (prioritized)

### 🔴 High Priority

**1. `storageutil` — 0% on core paths**
- `BackupToDevice` — 0%
- `UploadFilesStreamed` — 0%
- `SetupCirrusDir` — 0%
- `FindUsbDeviceBySerial` — 0%
- `isDeviceMounted` — 0%
- `UsbDevice` getters — all 0%
- `GetMountsDir` — 0%
These are the heart of the product. Need tests.

**2. `updateutil` — update path barely tested**
- `GetLatestVersionFromDefaultSources` — 0%
- `GetLatestVersion` — 0%
- `UpdateFromDefaultSources` — 0%
- `ListPossibleUpdatesFromDefaultSources` — 0%
- `replaceSelf` — 34.4%
The actual update flow is essentially untested. Given we have auto-update coming (PR #639),
this needs to be much tighter.

**3. API handlers — completely untested**
All of `cirrus`, `version`, `photos`, `storage`, `thumbnails`, `auth`, `books`, `metrics`, `migration`
have zero tests. These are the HTTP-facing entry points. At minimum the happy path and
key error paths should be tested for each handler.

### 🟡 Medium Priority

**4. `githubutil` — `FetchLatestRelease` = 0%**
Used by the update system. Should be mocked and tested.

**5. `workerutil` — `GetBackupToDeviceChannel` = 0%**
The backup worker channel accessor is untested.

**6. `deputil` — dependency wiring untested**
`DefaultDependencies`, `WithWorker`, `Worker` all at 0%. These wire up the app on startup.

### 🟢 Lower Priority (but worth noting)

**7. `usbutil`** — platform-specific code, hard to test without hardware. Could add
interface-level tests and mock the OS calls.

**8. `botel`/`botelsqlite`** — observability infrastructure. Lower risk if untested
but ideally would have integration tests for the SQLite exporter.

**9. `calendar`** — 0%. No tests at all. Unknown scope/risk.

**10. `migration`** — 0%. DB migrations are risky to leave untested.

---

## Flutter Tests

Single test file: `test/widget_test.dart`
- Tests that `FileBrowserPage` renders with "Cirrus", "Name", "Device", "Size" visible
- That's the entire Flutter test suite

**Gap:** No tests for `AppSettings`, `CirrusService`, any page logic, any service layer.
Given we're adding features (auto-update toggle, SBOM page), the Flutter side needs
at least basic service/model tests.

---

## Recommended Action Plan

### Phase 1 — Patch the update path (ties to PR #639)
- Add tests for `updateutil.GetLatestVersion` (mock HTTP)
- Add tests for `updateutil.UpdateFromDefaultSources` (mock)
- Add tests for `githubutil.FetchLatestRelease`

### Phase 2 — storageutil gaps
- `BackupToDevice`, `UploadFilesStreamed`, `SetupCirrusDir`
- `UsbDevice` getters (can test without hardware via struct construction)
- `FindUsbDeviceBySerial`, `isDeviceMounted` (mock filesystem)

### Phase 3 — API handler tests
- Start with `cirrus` handlers (most used)
- Use `httptest` + mock service layer
- Then `version`, `storage`, `photos`

### Phase 4 — Flutter basics
- `AppSettings` unit tests (preferences load/save)
- `CirrusService` tests (mock HTTP responses)

### Phase 5 — Infrastructure
- `deputil` wiring tests
- `migration` smoke tests
- `calendar` package

---

## Notes on Existing Tests

The existing tests are generally well-written — table-driven where appropriate,
good use of `httptest`, real filesystem ops for storage tests (using `t.TempDir()`).
James's style: concrete, not overly abstracted. Match that pattern.

One thing to watch: `storageutil_test.go` is 1499 lines. It's thorough for what it covers
but the gaps above are real.
