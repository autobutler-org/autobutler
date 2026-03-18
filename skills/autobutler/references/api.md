# AutoButler API Reference

Base URL: configured in `TOOLS.md`. All endpoints require `Authorization: Bearer <token>` unless noted.

## Auth

| Method | Path | Auth required | Description |
|--------|------|--------------|-------------|
| GET | `/api/v1/auth/status` | No | Returns `{"setup": bool}` |
| POST | `/api/v1/auth/setup` | No | First-boot account creation. Body: `{username, password}`. Returns `{token, recoveryPhrase}` |
| POST | `/api/v1/auth/login` | No | Body: `{username, password}`. Returns `{token}` |
| POST | `/api/v1/auth/logout` | Yes | Invalidates session. Returns `{message}` |
| POST | `/api/v1/auth/recover` | No | Body: `{recoveryPhrase, newPassword}`. Returns `{token}` |

## Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | System health. Returns: `{healthy, alerts[], cpuPercent, cpuCorePercents[], cpuCoreCount, memPercent, memUsedBytes, memTotalBytes, diskPercent, diskUsedBytes, diskTotalBytes, temperatureCelsius}` |

## Files (Cirrus)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/cirrus` | List files. Query params: `rootDir` (subdirectory), `serial` (device serial) |
| GET | `/api/v1/cirrus/search` | Search files. Query params: `query` (required), `serial` |
| GET | `/api/v1/cirrus/download` | Download file. Query params: `filePath` (required), `serial` |
| POST | `/api/v1/cirrus/upload` | Upload file (multipart). Field: file |
| POST | `/api/v1/cirrus/upload/{rootDir}` | Upload to subdirectory. Query param: `serial` |
| PUT | `/api/v1/cirrus` | Move/rename file. Body: `{oldFilePath, newFilePath, oldDeviceSerial?, newDeviceSerial?}` |
| DELETE | `/api/v1/cirrus` | Delete file. Query params: `rootDir`, `filePaths`, `serial` |
| POST | `/api/v1/cirrus/folder/{folderDir}` | Create folder. Form field: `folderName` |

### File node shape
```json
{
  "name": "photo.jpg",
  "path": "/Photos/photo.jpg",
  "size": 2048576,
  "isDir": false,
  "deviceName": "Root Volume",
  "deviceSerial": ""
}
```

## Photos

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/photos` | List photos. Query params: `serial`, `rootDir` |
| GET | `/api/v1/thumbnails/{filePath}` | Get thumbnail. Path: URL-encoded file path segments |

## Storage Devices

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/storage/devices/status` | List all devices. Returns `{count, devices[]}` |
| GET | `/api/v1/storage/devices/status/{serial}` | Single device status |
| POST | `/api/v1/storage/devices/usb/{serial}` | Enable USB device |
| DELETE | `/api/v1/storage/devices/usb/{serial}` | Disable/unmount USB device |
| POST | `/api/v1/storage/devices/backup` | Trigger backup to device. Body: `{serial}` |

### Device shape
```json
{
  "name": "Root Volume",
  "devicePath": "/dev/mmcblk0p2",
  "mountPoint": "/",
  "fileSystem": "ext4",
  "totalBytes": 62284591104,
  "usedBytes": 26143559680,
  "availableBytes": 36141031424,
  "isInternal": true,
  "isEnabled": true,
  "dataDir": "/var/lib/autobutler/data",
  "cirrusDir": "/var/lib/autobutler/data/cirrus"
}
```

## Version / Updates

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/version` | Installed version. Returns `{semver, gitCommit, goVersion, buildDate}` |
| GET | `/api/v1/version/available` | List available versions. Returns `[{version, url}]`. Query param: `all=true` for full list |
| POST | `/api/v1/version/latest` | Update to latest version |
| POST | `/api/v1/version/update` | Update to specific version. Body: `{version}` |

## SBOM

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/sbom` | Software bill of materials |

## Error responses

All errors return `{"error": "<message>"}`. HTTP status codes:
- `400` — bad request / validation failure
- `401` — not authenticated or invalid token
- `500` — internal server error
