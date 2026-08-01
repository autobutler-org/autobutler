-- Add admin flag to users. The first user created during setup is
-- automatically promoted to admin by the setup handler.
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;
