package vfs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Get returns all metadata for (namespace, path) as a map of key → JSON value.
// Returns an empty map (not an error) if no metadata is set.
func (s *SQLiteMetadataStore) Get(ctx context.Context, namespace, path string) (map[string]json.RawMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, value FROM vfs_metadata WHERE namespace=? AND path=?`,
		namespace, path,
	)
	if err != nil {
		return nil, fmt.Errorf("vfs metadata get: %w", err)
	}
	defer rows.Close()

	result := make(map[string]json.RawMessage)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("vfs metadata get scan: %w", err)
		}
		result[key] = json.RawMessage(value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vfs metadata get rows: %w", err)
	}
	return result, nil
}

// Set merges kv into existing metadata for (namespace, path).
// Keys in kv overwrite existing values; absent keys are unchanged.
func (s *SQLiteMetadataStore) Set(ctx context.Context, namespace, path string, kv map[string]json.RawMessage) error {
	if len(kv) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("vfs metadata set begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO vfs_metadata (namespace, path, key, value, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(namespace, path, key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
	)
	if err != nil {
		return fmt.Errorf("vfs metadata set prepare: %w", err)
	}
	defer stmt.Close()

	for key, value := range kv {
		if _, err := stmt.ExecContext(ctx, namespace, path, key, string(value)); err != nil {
			return fmt.Errorf("vfs metadata set exec key %q: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vfs metadata set commit: %w", err)
	}
	return nil
}

// DeleteKeys removes specific keys from metadata for (namespace, path).
// Deleting a non-existent key is a no-op.
func (s *SQLiteMetadataStore) DeleteKeys(ctx context.Context, namespace, path string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("vfs metadata delete keys begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx,
		`DELETE FROM vfs_metadata WHERE namespace=? AND path=? AND key=?`,
	)
	if err != nil {
		return fmt.Errorf("vfs metadata delete keys prepare: %w", err)
	}
	defer stmt.Close()

	for _, key := range keys {
		if _, err := stmt.ExecContext(ctx, namespace, path, key); err != nil {
			return fmt.Errorf("vfs metadata delete key %q: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vfs metadata delete keys commit: %w", err)
	}
	return nil
}

// Query returns all (namespace, path) entries where the given key equals value.
// Pass value=nil to match any entry that has the key set (existence check).
func (s *SQLiteMetadataStore) Query(ctx context.Context, namespace, key string, value json.RawMessage) ([]MetaEntry, error) {
	var rows *sql.Rows
	var err error

	if value == nil {
		rows, err = s.db.QueryContext(ctx,
			`SELECT path FROM vfs_metadata WHERE namespace=? AND key=? ORDER BY path`,
			namespace, key,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT path FROM vfs_metadata WHERE namespace=? AND key=? AND value=? ORDER BY path`,
			namespace, key, string(value),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("vfs metadata query: %w", err)
	}
	defer rows.Close()

	// Collect unique paths.
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("vfs metadata query scan: %w", err)
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vfs metadata query rows: %w", err)
	}

	// For each matched path, fetch all metadata to populate MetaEntry.Meta.
	entries := make([]MetaEntry, 0, len(paths))
	for _, p := range paths {
		meta, err := s.Get(ctx, namespace, p)
		if err != nil {
			return nil, err
		}
		entries = append(entries, MetaEntry{
			Namespace: namespace,
			Path:      p,
			Meta:      meta,
		})
	}
	return entries, nil
}
