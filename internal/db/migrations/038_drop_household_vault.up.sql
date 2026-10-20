-- The household vault (005_vault), which one master password opened for every
-- account, is gone: each account keeps its own ciphertext vault in vault_keys
-- and vault_items (037), and nothing reads these tables any more (#3460).

DROP INDEX IF EXISTS idx_vault_entries_url_host;

DROP INDEX IF EXISTS idx_vault_entries_folder;

DROP TABLE IF EXISTS vault_location;

DROP TABLE IF EXISTS vault_entries;

DROP TABLE IF EXISTS vault_folders;

DROP TABLE IF EXISTS vault_config;
