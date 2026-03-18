# Release Process

## Overview

AutoButler releases are distributed via two sources:

- **Azure Blob Storage** — primary source for device auto-updates
  (`autobutlerrelease.blob.core.windows.net/releases/autobutler/<version>/`)
- **GitHub Releases** — secondary source; also used for manual downloads

Releases are created automatically by the `.goreleaser.yaml` CI pipeline on tag push.

## Releasing

```bash
git tag v0.X.Y
git push origin v0.X.Y
```

GoReleaser handles building, packaging, and uploading to both Azure and GitHub.

## Yanking a release

Use this when a release ships with a critical bug and must be pulled immediately.

### Azure (primary — blocks auto-update)

Delete the blob directory for the broken version. The version listing is
blob-presence-driven — once the blob is gone, AB devices stop seeing it.

```bash
make release/yank VERSION=v0.X.Y
```

Or manually via Azure CLI:

```bash
az storage blob delete-batch \
  --account-name autobutlerrelease \
  --source releases \
  --pattern "autobutler/v0.X.Y/*"
```

Or via the Azure Portal: Storage account → `autobutlerrelease` → Containers →
`releases` → `autobutler/` → delete the version folder.

### GitHub Releases

Mark the release as a pre-release so `GET /repos/.../releases/latest` skips it:

```bash
gh release edit v0.X.Y --prerelease --repo autobutler-org/autobutler
```

Or delete it entirely if the assets should not be downloadable at all:

```bash
gh release delete v0.X.Y --repo autobutler-org/autobutler
```

### After yanking

1. Ship a patch release (`v0.X.Y+1`) with the fix ASAP
2. Devices that already installed the bad version need the patch — the auto-update
   mechanism will pick it up on next check once the patch is `latest`

## Blob path convention

```
releases/autobutler/<version>/<artifact>
```

Example:
```
releases/autobutler/v0.15.0/autobutler_linux_arm64.tar.gz
```

The artifact name is constructed from `ConstructArchiveName()` in
`pkg/util/updateutil/updateutil.go`.
