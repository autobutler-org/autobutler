-- Drops everything 032_baseline.up.sql creates: the tables, newest first, with
-- their indexes and triggers, then the functions.

DROP TABLE maintenance_runs;
DROP TABLE account_request_history;
DROP TABLE user_settings;
DROP TABLE settings;
DROP TABLE install;
DROP TABLE chat_read_markers;
DROP TABLE calendar_events;
DROP TABLE calendars;
DROP TABLE chat_reactions;
DROP TABLE chat_messages;
DROP TABLE chat_channel_events;
DROP TABLE chat_key_grants;
DROP TABLE chat_channel_keys;
DROP TABLE user_chat_keys;
DROP TABLE chat_channel_members;
DROP TABLE chat_channels;
DROP TABLE chat_servers;
DROP TABLE path_access;
DROP TABLE group_members;
DROP TABLE groups;
DROP TABLE jobs;
DROP TABLE vfs_db_entries;
DROP TABLE vfs_metadata;
DROP TABLE file_content;
DROP TABLE vault_location;
DROP TABLE vault_entries;
DROP TABLE vault_folders;
DROP TABLE vault_config;
DROP TABLE photo_hashes;
DROP TABLE photo_rotations;
DROP TABLE photo_favorites;
DROP TABLE photo_album_items;
DROP TABLE photo_albums;
DROP TABLE connected_devices;
DROP TABLE device_roles;
DROP TABLE device_names;
DROP TABLE sessions;
DROP TABLE users;

DROP FUNCTION chat_reactions_tombstone ();
DROP FUNCTION json_extract (TEXT, TEXT);
DROP FUNCTION datetime (TEXT, TEXT);
