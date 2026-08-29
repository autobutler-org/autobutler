---
name: quark
description: Interact with a live Quark instance via its REST API. Use when asked to check quark health, list or manage files on the quark, check or trigger updates, inspect connected storage devices, run diagnostics, or perform any operation against a running Quark server. Also use when setting up auth or logging into a quark for the first time.
---

# Quark Skill

Quark is a self-hosted private cloud. This skill covers authenticating and making API calls against a live instance.

## Configuration

The quark host URL and credentials are stored in `TOOLS.md` under `## Quark (local instance)`. Always read that section before making API calls. If no host is configured, ask the user for the URL, username, and password.

## Auth Flow

All API endpoints except `/api/v0/auth/*` require `Authorization: Bearer <token>`.

### Login (normal)
```bash
curl -s -X POST $QUARK_URL/api/v0/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"<user>","password":"<pass>"}'
# → {"token":"<64-hex-char token>"}
```

Store the token in-memory for the session. Do not persist it to files.

### First boot (setup not complete)
```bash
curl -s $QUARK_URL/api/v0/auth/status
# → {"setup":false} means first boot — call /auth/setup instead
curl -s -X POST $QUARK_URL/api/v0/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"<user>","password":"<pass>"}'
# → {"token":"...","recoveryPhrase":"word-word-word-word-word-word","message":"..."}
# Show the recovery phrase to the user — it will not be shown again
```

### Recovery phrase reset
```bash
curl -s -X POST $QUARK_URL/api/v0/auth/recover \
  -H "Content-Type: application/json" \
  -d '{"recoveryPhrase":"word-word-word-word-word-word","newPassword":"<new>"}'
# → {"token":"..."}
```

## Common Operations

See [`references/api.md`](references/api.md) for the full endpoint reference.

### Health check
```bash
curl -s $QUARK_URL/api/v0/health -H "Authorization: Bearer $TOKEN"
```
Key fields: `healthy` (bool), `alerts` (array), `cpuPercent`, `memPercent`, `diskPercent`, `temperatureCelsius`.
Alert if `diskPercent > 85` or `temperatureCelsius > 70`.

### List files
```bash
curl -s "$QUARK_URL/api/v0/files" -H "Authorization: Bearer $TOKEN"
# With subdirectory: ?rootDir=Photos
# With specific device: ?serial=<serial>
```

### Check version / trigger update
```bash
curl -s $QUARK_URL/api/v0/version -H "Authorization: Bearer $TOKEN"
curl -s $QUARK_URL/api/v0/version/available -H "Authorization: Bearer $TOKEN"
curl -s -X POST $QUARK_URL/api/v0/version/update -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"version":"v1.2.3"}'
```

### Storage devices
```bash
curl -s $QUARK_URL/api/v0/storage/devices/status -H "Authorization: Bearer $TOKEN"
```

## Notes

- Tokens are valid for 30 days; re-login if you get a 401
- `$QUARK_URL` depends on how the quark is being run: `http://localhost:8080` for
  `make watch/backend`, `https://localhost` (:443, self-signed — `curl` needs `-k`) for
  `make watch/backend/secure`, and `http://<tailscale-ip>:80` over remote access
- All endpoints listed here are under the `/api/v0/` prefix — it is the only API version
- Swagger UI available at `$QUARK_URL/swagger` when the backend is running
- **Always set a `User-Agent` header** matching your agent name (e.g. `exokomodo-bot`, `sable-bot`). The quark tracks connected devices by IP + User-Agent — this is how the admin sees which agent is talking to the quark in the devices list.
