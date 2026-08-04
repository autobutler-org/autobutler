-- User role system (#1204).
--
-- role: 'owner' | 'admin' | 'user'
--
-- 'owner' is the first account created at setup — there is always exactly one.
-- 'admin' can approve/revoke devices and promote other users.
-- 'user' is a standard authenticated user.
--
-- The first signup (via /auth/setup) inserts with role = 'owner'.
-- Subsequent signups (if multi-user is ever enabled) default to 'user'.
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'owner'
    CHECK (role IN ('owner', 'admin', 'user'));
