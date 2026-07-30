-- Device registration and approval flow.
--
-- registered_devices tracks every device that has requested access and the
-- admin's approval decision. It is separate from connected_devices, which is
-- a passive traffic log.
--
-- identity_type: "local" (LAN device identified by IP/MAC) or "tailscale"
--                (identified by Tailscale node key / stable IP).
-- approval_status: "pending" | "approved" | "revoked"
CREATE TABLE IF NOT EXISTS registered_devices (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL DEFAULT '',
    device_type     TEXT NOT NULL DEFAULT 'unknown',   -- e.g. "phone", "laptop", "tablet"
    identity_type   TEXT NOT NULL DEFAULT 'local',     -- "local" | "tailscale"
    ip_address      TEXT NOT NULL DEFAULT '',
    mac_address     TEXT,                              -- local only; nullable
    tailscale_key   TEXT,                              -- tailscale only; nullable
    user_agent      TEXT NOT NULL DEFAULT '',
    approval_status TEXT NOT NULL DEFAULT 'pending',   -- "pending" | "approved" | "revoked"
    approved_by     TEXT,                              -- username of approving admin
    approved_at     DATETIME,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Uniqueness: same IP+UA pair is the same device (local). Same Tailscale key is the same node.
CREATE UNIQUE INDEX IF NOT EXISTS idx_registered_devices_ip_ua
    ON registered_devices (ip_address, user_agent) WHERE tailscale_key IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_registered_devices_ts_key
    ON registered_devices (tailscale_key) WHERE tailscale_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_registered_devices_status
    ON registered_devices (approval_status);
