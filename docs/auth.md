# Authentication

AutoButler uses local username/password auth. No cloud, no OAuth required — your credentials live on your device.

## First boot

When you start AutoButler for the first time, there are no users. Everything is wide open until you run setup — so do that first.

```bash
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username": "you", "password": "your-password"}'
```

You'll get back a session token and a **recovery phrase**. Write the phrase down somewhere safe. You won't see it again, and it's the only way to reset your password if you forget it.

```json
{
  "token": "abc123...",
  "recoveryPhrase": "wagon-river-flame-orbit-cedar-stone",
  "message": "Setup complete. Store your recovery phrase somewhere safe — it will not be shown again."
}
```

## Logging in

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "you", "password": "your-password"}'
```

Returns a session token. Sessions last 30 days.

## Using the token

Pass it as a Bearer token:

```bash
curl http://localhost:8080/api/v1/some-endpoint \
  -H "Authorization: Bearer <your-token>"
```

Or it gets set automatically as a cookie if you're going through the browser.

## Logging out

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <your-token>"
```

This kills the session server-side. The cookie gets cleared too.

## Forgot your password?

Use your recovery phrase:

```bash
curl -X POST http://localhost:8080/api/v1/auth/recover \
  -H "Content-Type: application/json" \
  -d '{"recoveryPhrase": "wagon-river-flame-orbit-cedar-stone", "newPassword": "new-password"}'
```

This resets your password, invalidates all existing sessions, and gives you a fresh token. Your recovery phrase stays the same.

## Check setup status

```bash
curl http://localhost:8080/api/v1/auth/status
```

Returns `{"setup": true}` or `{"setup": false}`. Useful for the frontend to know whether to show the onboarding flow.

## Notes

- Passwords must be at least 8 characters
- The API is wide open until setup is complete (so finish setup before exposing the port)
- There's no multi-user support yet — one owner account per device
