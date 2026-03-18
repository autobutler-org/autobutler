---
name: autobutler
description: Interact with a live AutoButler instance via its REST API. Use when asked to check butler health, list or manage files on the butler, check or trigger updates, inspect connected storage devices, run diagnostics, or perform any operation against a running AutoButler server. Also use when setting up auth or logging into a butler for the first time.
---

# AutoButler Skill

AutoButler is a self-hosted private cloud. This skill covers authenticating and making API calls against a live instance.

## Configuration

The butler host URL and credentials are stored in `TOOLS.md` under `## AutoButler (local instance)`. Always read that section before making API calls. If no host is configured, ask the user for the URL, username, and password.

## Auth Flow

All API endpoints except `/api/v1/auth/*` require `Authorization: Bearer <token>`.

### Login (normal)
```bash
curl -s -X POST $BUTLER_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"<user>","password":"<pass>"}'
# → {"token":"<64-hex-char token>"}
```

Store the token in-memory for the session. Do not persist it to files.

### First boot (setup not complete)
```bash
curl -s $BUTLER_URL/api/v1/auth/status
# → {"setup":false} means first boot — call /auth/setup instead
curl -s -X POST $BUTLER_URL/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"<user>","password":"<pass>"}'
# → {"token":"...","recoveryPhrase":"word-word-word-word-word-word","message":"..."}
# Show the recovery phrase to the user — it will not be shown again
```

### Recovery phrase reset
```bash
curl -s -X POST $BUTLER_URL/api/v1/auth/recover \
  -H "Content-Type: application/json" \
  -d '{"recoveryPhrase":"word-word-word-word-word-word","newPassword":"<new>"}'
# → {"token":"..."}
```

## Common Operations

See [`references/api.md`](references/api.md) for the full endpoint reference.

### Health check
```bash
curl -s $BUTLER_URL/api/v1/health -H "Authorization: Bearer $TOKEN"
```
Key fields: `healthy` (bool), `alerts` (array), `cpuPercent`, `memPercent`, `diskPercent`, `temperatureCelsius`.
Alert if `diskPercent > 85` or `temperatureCelsius > 70`.

### List files
```bash
curl -s "$BUTLER_URL/api/v1/cirrus" -H "Authorization: Bearer $TOKEN"
# With subdirectory: ?rootDir=Photos
# With specific device: ?serial=<serial>
```

### Check version / trigger update
```bash
curl -s $BUTLER_URL/api/v1/version -H "Authorization: Bearer $TOKEN"
curl -s $BUTLER_URL/api/v1/version/available -H "Authorization: Bearer $TOKEN"
curl -s -X POST $BUTLER_URL/api/v1/version/latest -H "Authorization: Bearer $TOKEN"
```

### Storage devices
```bash
curl -s $BUTLER_URL/api/v1/storage/devices/status -H "Authorization: Bearer $TOKEN"
```

## Notes

- Tokens are valid for 30 days; re-login if you get a 401
- The butler runs on port 80 locally; may be on a different port remotely
- All endpoints are under `/api/v1/` prefix
- Swagger UI available at `$BUTLER_URL/swagger` when the backend is running
