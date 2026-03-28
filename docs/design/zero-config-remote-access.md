# feat: Seamless zero-config remote access via embedded Tailscale SDK (no user interaction)

## Summary

Remove the manual "Enable Remote Access" toggle entirely. Remote access should be invisible infrastructure — the Pi joins the tailnet on first boot, the phone joins during pairing, and the app seamlessly switches between LAN and tailnet connectivity. Users never see a VPN prompt, never flip a switch, never think about networking.

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    AutoButler Headscale VPS                      │
│                  (165.227.215.101 or similar)                    │
│                                                                  │
│  ┌──────────────┐   ┌──────────────────────────┐                │
│  │  Headscale    │   │  Provisioning Service     │               │
│  │  :8080        │   │  :8081                    │               │
│  │              │◄──│  POST /provision           │               │
│  │  ACL policy:  │   │  POST /provision/mobile   │  ← NEW       │
│  │  per-pair     │   │                            │               │
│  │  isolation    │   │  Mints pre-auth keys for   │              │
│  └──────────────┘   │  both Pi and phone nodes   │               │
│                      └──────────────────────────┘                │
└──────────────────────┬────────────────────┬─────────────────────┘
                       │                    │
              WireGuard tunnel      WireGuard tunnel
              (always-on)           (on-demand)
                       │                    │
         ┌─────────────┴──┐      ┌──────────┴──────────┐
         │  Raspberry Pi   │      │  Mobile App          │
         │                 │      │                      │
         │  tsnet node     │      │  Embedded tsnet      │
         │  (in-process)   │      │  (Go mobile lib      │
         │  hostname:      │      │   via FFI, userspace │
         │   ab-<deviceID> │      │   WireGuard)         │
         │                 │      │                      │
         │  Gin server :80 │      │  Flutter UI          │
         │  ← proxy via    │      │  ConnectionManager   │
         │    tsnet :80    │      │  (LAN-first,         │
         └─────────────────┘      │   tailnet fallback)  │
                                  └─────────────────────┘
```

**Key invariants:**
- Every Pi is always on the tailnet (joins on first boot, reconnects on restart)
- Every paired phone is on the tailnet (joins silently during pairing)
- A phone can only reach its own Pi (enforced by Headscale ACLs)
- LAN is preferred when available; tailnet is the fallback, not the primary path
- No VPN permission dialogs — the embedded lib uses userspace WireGuard

---

## 2. Pi-Side Changes

### Current state
- `remoteutil.Start(authKey)` is called manually via `POST /api/v1/settings/remote-access`
- Auth key is provisioned on-demand, tsnet state persisted to `/var/lib/autobutler/tsnet`
- `settingsutil.SetRemoteAccess(true)` gates the behavior

### Target state
- tsnet starts **unconditionally** during `server.StartServer()`, before the Gin router begins accepting requests
- No settings toggle — remote access is always on

### Changes required

**`cmd/autobutler/serve/serve.go` / `internal/server/server.go`:**
- On startup, call a new `remoteutil.EnsureStarted()` function
- This function checks if tsnet state already exists (i.e., already provisioned):
  - **If state exists:** Call `remoteutil.Start("")` (empty auth key — tsnet reuses persisted credentials)
  - **If no state:** Call the provisioning service to mint a key, then `remoteutil.Start(authKey)`, then start the proxy
- If provisioning fails (network down, VPS unreachable), log a warning and retry on a backoff loop in the background. The local Gin server must still start and serve LAN requests.

**Idempotency:**
- `remoteutil.Start()` already no-ops if `running == true`
- The tsnet state directory (`/var/lib/autobutler/tsnet`) is the source of truth for "already provisioned"
- Check for the presence of state files before calling the provisioning service

**Environment / config:**
- `AUTOBUTLER_HEADSCALE_URL` — already exists, defaults to `http://165.227.215.101:8080`
- `AUTOBUTLER_PROVISIONING_URL` — already exists, defaults to `http://165.227.215.101:8081`
- `AUTOBUTLER_PROVISIONING_SECRET` — already exists, required
- No new env vars needed on the Pi side

**Hostname convention:**
- Change from the static `autobutler` hostname to `ab-<first12chars-of-deviceID>` so multiple Pis are distinguishable in Headscale

**`remote_access.go` settings endpoints:**
- `POST /settings/remote-access` — deprecate (return 200 with current status, no-op)
- `DELETE /settings/remote-access` — deprecate (return 200, log a warning, do not stop tsnet)
- `GET /settings/remote-access` — keep, update to always return `enabled: true` plus the tailnet IP
- Add `GET /settings/remote-access/info` that returns `{ tailnetIP, deviceID, hostname }` — consumed during pairing

---

## 3. Provisioning Service Changes

### Current state
- Single endpoint: `POST /provision` mints a Headscale pre-auth key under the `autobutler` user
- All nodes (Pi devices) share a single Headscale user namespace
- Keys are single-use, non-reusable, non-ephemeral, 1-hour expiry

### New requirements
- Mint keys for **two types of nodes**: Pi devices and mobile clients
- Distinguish them in Headscale for ACL enforcement
- Track which phone is paired with which Pi (for ACL generation)

### Headscale user/namespace strategy

Use **one Headscale user per Pi** rather than a single shared `autobutler` user. This provides natural isolation:

```
Headscale users:
  pi-<deviceID>     → Pi node lives here
  phone-<deviceID>  → All phones paired with this Pi live here
```

ACLs then allow `phone-<deviceID>` → `pi-<deviceID>` traffic only.

### New endpoint: `POST /provision/mobile`

```json
// Request
{
  "device_id": "abc123...",        // Pi's device ID (identifies which Pi)
  "phone_id": "def456...",         // Unique phone identifier
  "pi_provisioning_secret": "..."  // The Pi's provisioning secret (proves the Pi authorized this)
}

// Response
{
  "auth_key": "tskey-...",
  "tailnet_ip": "100.64.x.y",     // Pi's current tailnet IP
  "pi_hostname": "ab-abc123"       // Pi's tailnet hostname
}
```

**Authentication:** This endpoint is called by the Pi on behalf of the phone during pairing. The Pi authenticates with the same `X-Provisioning-Secret` header. The phone never talks to the provisioning service directly.

**Key properties for mobile keys:**
- Single-use, non-reusable
- **Ephemeral: true** — if the phone disconnects and the key expires, the node is auto-removed. This is cleaner than leaving zombie phone nodes.
- 24-hour expiry (longer than Pi keys since pairing might not complete immediately)

### Modified `POST /provision` (Pi keys)

- Add a `node_type` field (default `"pi"` for backward compat)
- Create the Headscale user `pi-<deviceID>` if it doesn't exist before minting the key
- Return the tailnet IP after the Pi joins (or let the Pi report it)

### Rate limiting adjustments
- Mobile provisioning: same 5/hour per device_id limit
- Consider a separate limit per phone_id to prevent a compromised Pi from minting unlimited phone keys

---

## 4. Mobile SDK Integration

### Approach: Go mobile library via FFI

The Tailscale `tsnet` package is pure Go. We compile it into a shared library using `gomobile bind` and call it from Flutter via `dart:ffi` (or a method channel wrapping the native bridge).

**Why not the system Tailscale app:**
- Requires VPN permission dialogs on both iOS and Android
- User needs to install and configure a separate app
- Can't control the experience

**Why embedded tsnet works without VPN permissions:**
- `tsnet` uses a userspace WireGuard implementation (`wireguard-go`)
- Traffic is routed in-process, not at the OS network layer
- No `NEVPNManager` / `VpnService` needed — it's just a library making UDP connections
- This is the same approach Tailscale's own embedded SDK demos use

### Go mobile bridge

Create a new Go module: `mobile/tsbridge/`

```
mobile/
  tsbridge/
    tsbridge.go       // Exported functions for gomobile bind
    go.mod
    go.sum
```

**Exported API (gomobile-compatible — no complex types, only primitives and byte slices):**

| Function | Signature | Description |
|----------|-----------|-------------|
| `Start` | `func Start(stateDir, authKey, controlURL, hostname string) error` | Initialize and start the tsnet node |
| `Stop` | `func Stop() error` | Shut down the tsnet node |
| `IsRunning` | `func IsRunning() bool` | Check if connected |
| `LocalAddr` | `func LocalAddr() string` | Return the node's tailnet IP |
| `DialHTTP` | `func DialHTTP(addr string, port int, path string, headers string) ([]byte, int, error)` | Make an HTTP request through the tailnet tunnel |

> **Note:** `DialHTTP` is the key function — rather than having Flutter's HTTP client try to reach tailnet IPs directly (which won't work since the tunnel is in-process and not at the OS level), all tailnet traffic must go through the Go bridge. The bridge opens a connection via `tsnet.Server.Dial()` and proxies the request.

### Alternative: Local SOCKS5/HTTP proxy

Instead of `DialHTTP`, the Go bridge could start a local SOCKS5 or HTTP proxy on `127.0.0.1:<random-port>` that tunnels traffic through the tsnet connection. Flutter's HTTP client would be configured to use this proxy when targeting the tailnet IP.

**Tradeoffs:**
- Pro: Flutter HTTP client code stays identical for LAN and tailnet
- Pro: Works with any HTTP library, not just the bridge
- Con: Extra hop (minimal latency impact since it's loopback)
- Con: Port management complexity

**Recommendation:** Start with the local proxy approach — it's cleaner for the Flutter side and avoids reimplementing HTTP semantics in Go.

### Build integration

- `gomobile bind -target=android -o mobile/tsbridge.aar mobile/tsbridge`
- `gomobile bind -target=ios -o mobile/Tsbridge.xcframework mobile/tsbridge`
- Outputs are checked into the repo (or built in CI) and referenced as native dependencies
- Android: drop `.aar` into `android/app/libs/`, reference in `build.gradle`
- iOS: embed `.xcframework` in the Xcode project

### Flutter integration layer

New Dart files:
- `lib/services/tailnet_service.dart` — wraps the native bridge via method channels
- Manages lifecycle: start on pairing, stop on unpair, persist auth key in secure storage
- Exposes `Stream<TailnetStatus>` (connecting, connected, disconnected, error)

### Binary size impact

The `tsnet` + `wireguard-go` dependency adds approximately **8-12 MB** to each platform binary (compressed). This is significant but acceptable for the value delivered. Stripping debug symbols and using `-ldflags="-s -w"` helps.

---

## 5. Pairing Flow Changes

### Current pairing flow
1. User opens app, enters Pi's LAN IP manually (or discovers via mDNS in future)
2. App adds `HostEntry` to `AppSettings.hosts`
3. User creates account / logs in via `AuthService`
4. Done — app talks to Pi over LAN only

### New pairing flow
1. User opens app, discovers Pi (mDNS / QR code / manual IP entry)
2. App connects to Pi over LAN, creates account / logs in
3. **NEW:** App calls `GET /api/v1/settings/remote-access/info` on the Pi
4. **NEW:** Pi returns `{ deviceID, tailnetIP, hostname }`
5. **NEW:** App generates a unique `phone_id` (UUID, persisted in secure storage)
6. **NEW:** App calls `POST /api/v1/pairing/tailnet` on the Pi with `{ phone_id }`
7. **NEW:** Pi calls provisioning service (`POST /provision/mobile`) with its own device_id + the phone_id
8. **NEW:** Pi returns to app: `{ authKey, controlURL, tailnetIP, piHostname }`
9. **NEW:** App initializes the embedded tsnet node with the received auth key
10. **NEW:** App persists `{ authKey (spent, for reference), controlURL, tailnetIP, piHostname, phone_id }` in secure storage
11. App updates `HostEntry` to include both LAN IP and tailnet IP
12. Done — app is now reachable on both networks

### New Pi endpoint: `POST /api/v1/pairing/tailnet`

```json
// Request (from phone, over LAN)
{
  "phone_id": "uuid-of-phone"
}

// Response
{
  "auth_key": "tskey-...",
  "control_url": "http://165.227.215.101:8080",
  "tailnet_ip": "100.64.0.5",
  "pi_hostname": "ab-abc123def456"
}
```

This endpoint:
- Authenticates via the existing session token (must be logged in)
- Calls the provisioning service to mint a mobile key
- Returns everything the phone needs to join the tailnet
- Is rate-limited (max 3 phone pairings per Pi per hour)

### Key storage on mobile

Stored in `FlutterSecureStorage` (already used for session tokens):
- `tailnet_auth_key` — the original key (consumed on first use, kept for debugging)
- `tailnet_control_url` — Headscale URL
- `tailnet_pi_ip` — Pi's tailnet IP for direct connection
- `tailnet_phone_id` — this phone's unique ID
- `tailnet_state_dir` — path to tsnet state (app documents directory)

---

## 6. Connection Manager

### Design

New class: `lib/services/connection_manager.dart`

Replaces the current approach of Flutter directly using `AppSettings.activeHost` as a static base URL. Instead, the connection manager dynamically resolves the best available route.

### Connection strategy

```
1. Try LAN IP (e.g., http://192.168.1.50:80)
   - TCP connect with 2-second timeout
   - If success → use LAN, mark as "local"

2. If LAN fails → try tailnet IP (e.g., http://100.64.0.5:80)
   - Ensure embedded tsnet is running (start if needed)
   - Route through local proxy
   - TCP connect with 5-second timeout
   - If success → use tailnet, mark as "remote"

3. If both fail → mark as "offline"
   - Retry on exponential backoff (2s, 4s, 8s, 16s, cap at 60s)
   - Also retry immediately on network state change (wifi connected, cellular toggle)
```

### State machine

```
         ┌──────────┐
    ┌───►│  Probing  │◄──── network change event
    │    └─────┬─────┘
    │          │
    │    ┌─────┴─────┐
    │    ▼           ▼
    │ ┌──────┐  ┌─────────┐
    │ │ LAN  │  │ Tailnet │
    │ └──┬───┘  └────┬────┘
    │    │           │
    │    ▼           ▼
    │  Connected   Connected
    │  (local)     (remote)
    │    │           │
    │    └─────┬─────┘
    │          │ connection lost
    │          ▼
    │    ┌──────────┐
    └────│ Retrying  │
         └──────────┘
              │ all attempts exhausted
              ▼
         ┌──────────┐
         │ Offline   │──── retry on backoff / network change
         └──────────┘
```

### UI surface

- Status indicator in app bar: 🟢 Connected (local) / 🟡 Connected (remote) / 🔴 Offline
- No "Enable Remote Access" toggle — it's gone
- Settings page shows tailnet IP for debugging purposes only (collapsed/advanced section)

### Periodic LAN re-probe

When connected via tailnet, probe the LAN IP every 30 seconds. If LAN comes back (user returned home), switch transparently. This avoids routing through the VPS when unnecessary.

### HTTP client changes

Currently every service (`CirrusService`, `AuthService`, etc.) constructs its own `Uri` from `AppSettings.activeHost`. This needs to be centralized:

- `ConnectionManager` exposes `Uri get activeBaseUri` and `Map<String, String> get proxyHeaders`
- All services use `ConnectionManager.instance.activeBaseUri` instead of `AppSettings.activeHost`
- When on tailnet, the HTTP client is configured with the local SOCKS5 proxy
- `ConnectionManager` listens to `connectivity_plus` for network state changes

---

## 7. Headscale Infrastructure

### Current state
- Single Headscale instance on DigitalOcean droplet (`165.227.215.101`)
- Single user namespace: `autobutler`
- No ACL policy configured

### Target state

**Server:** Same droplet is fine for now. Headscale is lightweight — a single instance can handle thousands of nodes.

**Deployment:**
- Run Headscale via Docker Compose with auto-restart
- Nginx reverse proxy with Let's Encrypt TLS (HTTPS for the control plane is important — auth keys transit over this connection)
- Automated backups of `/var/lib/headscale/db.sqlite` to a DO Space or S3 bucket (daily)

**TLS is critical:** Currently the control URL is `http://...` — auth keys are sent in plaintext. Before shipping this to real users, the Headscale control server **must** be behind TLS. The provisioning service should also be TLS-terminated.

### ACL policy

```json
{
  "acls": [
    {
      "action": "accept",
      "src": ["group:phone-*"],
      "dst": ["group:pi-*:80"]
    }
  ],
  "groups": {
    // Dynamically generated — provisioning service updates ACLs
    // when a new pairing is created
    "group:phone-abc123": ["phone-abc123"],
    "group:pi-abc123": ["pi-abc123"]
  },
  "tagOwners": {
    "tag:pi": ["autobutler"],
    "tag:phone": ["autobutler"]
  }
}
```

**Actually — simpler approach with Headscale users:**

Since we're using one Headscale user per Pi (`pi-<deviceID>`) and its paired phones (`phone-<deviceID>`), and Headscale supports user-based ACLs:

```json
{
  "acls": [
    {
      "action": "accept",
      "src": ["phone-<deviceID>"],
      "dst": ["pi-<deviceID>:80"]
    }
  ]
}
```

The provisioning service appends a new ACL rule each time a new Pi is provisioned. This keeps the policy simple and auditable.

**Port restriction:** Phones can only reach port 80 on their Pi. No SSH, no other services. Minimizes attack surface.

### Operational concerns

- **Monitoring:** Health check endpoint on Headscale (`/health`), alert if down
- **Scaling:** Single instance handles ~10k nodes easily. Only needs HA if AutoButler grows significantly.
- **Cost:** ~$6/mo for a basic DO droplet. Bandwidth is the variable — WireGuard handshakes are tiny, but media streaming through DERP relay would be expensive. Tailscale's default DERP servers handle relay traffic, not our Headscale instance.
- **DERP:** Headscale can optionally run an embedded DERP server. For v1, rely on Tailscale's public DERP infrastructure. If that becomes a concern, self-host DERP on the same droplet.

---

## 8. Security Model

### Threat model

| Threat | Mitigation |
|--------|------------|
| Cross-user access (phone A reaches Pi B) | Headscale ACLs: per-pair isolation. Phone keys are minted under `phone-<piDeviceID>` namespace. |
| Stolen phone | Ephemeral keys — node is removed when disconnected long enough. User can also unpair from Pi's web UI (future). |
| Compromised Pi | Pi can only mint keys for its own paired phones. Provisioning service validates device_id ownership. |
| MITM on provisioning | TLS on both Headscale and provisioning service (must-have before GA). |
| Auth key interception | Keys are single-use and expire in 1-24 hours. Transmitted over LAN during pairing (local network only). |
| Replay of spent auth key | Single-use keys are invalidated by Headscale after first use. |

### Key rotation

- **Pi keys:** Long-lived (tsnet state persists across reboots). If the Pi is re-provisioned (factory reset), the old node is orphaned in Headscale. The provisioning service should deregister the old node when minting a new key for the same device_id.
- **Phone keys:** Ephemeral — Headscale auto-cleans disconnected ephemeral nodes. If a phone needs to reconnect after key expiry, the app re-initiates the pairing flow over LAN.
- **Re-keying cadence:** For v1, no automatic rotation. The tsnet state (WireGuard keys) persists indefinitely. Future: rotate WireGuard keys every 90 days via tsnet's built-in key rotation.

### Unpair flow

When a user removes a Pi from their app (or vice versa):
1. App calls `DELETE /api/v1/pairing/tailnet` on the Pi (over LAN or tailnet)
2. Pi calls provisioning service to deregister the phone node from Headscale
3. App stops its embedded tsnet, deletes local state from secure storage
4. Pi updates its ACL to remove the phone's access

If the phone can't reach the Pi (already offline), the ephemeral node will auto-expire.

### Multi-phone support

A single Pi may be paired with multiple phones (household members). Each phone gets its own key and node identity. ACLs allow all phones in `phone-<piDeviceID>` to reach that Pi.

---

## 9. Rollout / Migration

### Existing users with manual remote access enabled

- These users already have a tsnet node running on their Pi under the `autobutler` Headscale user
- **Migration path:**
  1. On upgrade, detect existing tsnet state in `/var/lib/autobutler/tsnet`
  2. If state exists and `settingsutil.GetRemoteAccess() == true`, the Pi is already on the tailnet — keep running as-is
  3. On next provisioning service update, migrate the node to the new `pi-<deviceID>` namespace (Headscale API: rename user or re-register node)
  4. Old `autobutler` namespace can be cleaned up after migration window

### Feature flags / gradual rollout

- **Phase 1-2 (Pi auto-start):** Ship behind an env var `AUTOBUTLER_AUTO_REMOTE=true` (default false initially)
- **Phase 3-4 (mobile SDK):** Ship behind a feature flag in `AppSettings` (`tailnet_enabled`, default false)
- **GA:** Remove flags, make everything default

### Backward compatibility

- The `POST /settings/remote-access` endpoint continues to work (returns current status, no-op enable)
- The `DELETE /settings/remote-access` endpoint becomes a no-op (can't disable what's always on)
- Old app versions that don't have the embedded SDK still work over LAN — they just won't have remote access
- New app + old Pi: app detects that `/api/v1/pairing/tailnet` returns 404, skips tailnet setup, operates LAN-only

---

## 10. Open Questions / Risks

### App Store / Play Store review

**Risk level: Medium**

- Embedded network libraries are not uncommon (Firebase, gRPC, etc. all include native networking)
- The critical distinction: we're **not** using `NEVPNManager` (iOS) or `VpnService` (Android), so we don't need VPN entitlements
- `tsnet` uses standard UDP sockets — same as any game or VoIP app
- **iOS concern:** Apple's Network Extension entitlement is only required if you're creating a system-wide VPN/proxy. Userspace WireGuard in a single app's process should not require it. **Needs validation with a test build submitted to TestFlight.**
- **Android concern:** Lower risk. No special permissions needed for userspace networking.

### iOS Network Extension entitlement

- If Apple requires the Network Extension entitlement even for in-process userspace networking, we'd need to apply for it (slow process, ~2-4 weeks)
- **Fallback:** If the entitlement is rejected, the iOS app could fall back to instructing users to install the Tailscale app and join via a tailnet invite link. Not zero-config, but better than nothing.

### Go mobile binary size

- Estimated +8-12 MB per platform (compressed)
- Current app size: ~15-20 MB → would roughly double
- Mitigation: strip symbols, use `-trimpath`, potentially split the Go bridge into a dynamically loaded library that's downloaded on first pairing (adds complexity, probably not worth it)

### Battery impact (mobile)

- The embedded tsnet node maintains a WireGuard tunnel with periodic keepalives
- When on LAN (primary path), the tsnet node should be in a low-power idle state
- On iOS, background execution is limited — the tsnet node may be suspended. Reconnection on app foreground needs to be fast (~1-2 seconds).
- **Recommendation:** Only start the tsnet node when the LAN probe fails, not permanently. Stop it when LAN is restored. This minimizes battery impact.

### DERP relay costs

- When direct WireGuard connections can't be established (symmetric NAT on both sides), traffic routes through DERP relay servers
- We're using Tailscale's public DERP infrastructure — no cost to us, but also no SLA
- For media-heavy operations (photo sync, file downloads) over DERP, performance may be poor
- **Mitigation:** Document that remote access works best when at least one side has a public IP or supports NAT traversal. Most home networks do.

### Multi-Pi households

- A user may have multiple Pis. Each Pi is independent with its own tailnet namespace.
- The phone would need separate tsnet nodes per Pi — or a single tsnet node that's authorized on multiple namespaces.
- **For v1:** Support one Pi per phone. Multi-Pi is a future enhancement.

### Headscale single point of failure

- If the Headscale VPS goes down, new pairings fail and disconnected nodes can't reconnect
- Already-connected nodes continue to work (WireGuard tunnels persist without the control server)
- **Mitigation:** Automated health checks, DO droplet monitoring, documented recovery procedure

---

## Implementation Phases

### Phase 1: Pi auto-start (Pi-side only, no mobile changes)

**Scope:**
- New `remoteutil.EnsureStarted()` function with idempotent provisioning
- Call during `server.StartServer()` startup, before Gin router starts
- Background retry loop if provisioning fails (don't block LAN serving)
- Per-Pi hostnames (`ab-<deviceID>` instead of `autobutler`)
- Deprecate the enable/disable settings endpoints (no-op wrappers)
- Add `GET /api/v1/settings/remote-access/info` endpoint
- Env var gate: `AUTOBUTLER_AUTO_REMOTE=true`

**Deliverable:** Every Pi auto-joins the tailnet on boot. No user interaction needed on the Pi side.

### Phase 2: Provisioning service v2 (server-side)

**Scope:**
- Per-Pi Headscale user namespaces (`pi-<deviceID>`, `phone-<deviceID>`)
- New `POST /provision/mobile` endpoint for phone key minting
- ACL policy generation and management
- TLS termination for both Headscale and provisioning service (Let's Encrypt)
- Migration logic for existing nodes from the shared `autobutler` namespace
- Headscale operational hardening (backups, monitoring, auto-restart)

**Deliverable:** Provisioning service can mint isolated keys for both Pi and phone nodes with proper ACL enforcement.

### Phase 3: Go mobile bridge (native library)

**Scope:**
- New `mobile/tsbridge/` Go module with gomobile-compatible API
- Local SOCKS5 proxy approach for tunneling HTTP traffic
- Build scripts for Android (`.aar`) and iOS (`.xcframework`)
- CI integration to build the native libraries
- Flutter method channel or FFI wrapper (`lib/services/tailnet_service.dart`)
- Binary size measurement and optimization

**Deliverable:** A working Flutter plugin that can start/stop a tsnet node and proxy HTTP traffic through it.

### Phase 4: Pairing flow + connection manager (mobile app)

**Scope:**
- New `POST /api/v1/pairing/tailnet` Pi endpoint
- Pairing flow integration — phone silently joins tailnet during setup
- `ConnectionManager` class replacing static host resolution
- LAN-first / tailnet-fallback connection strategy
- Periodic LAN re-probe when on tailnet
- Status indicator in app bar (local / remote / offline)
- Unpair flow (deregister phone node)
- Key and state persistence in secure storage
- Remove the "Remote Access" settings toggle from the UI

**Deliverable:** Complete zero-config remote access. User pairs the app, remote access just works.

### Phase 5: Hardening + GA (polish)

**Scope:**
- TestFlight / Play Store internal testing submission to validate review policies
- Battery impact profiling and optimization (lazy tsnet start)
- Edge case handling (expired keys, re-pairing after factory reset, multi-phone)
- Migration path for users upgrading from manual remote access
- Remove feature flags, make auto-remote the default
- Documentation update
- Monitoring dashboard for Headscale node count, connection health

**Deliverable:** Production-ready, all flags removed, shipped to all users.
