package vfs

import (
	"context"
	"encoding/json"
)

// MetadataStore stores arbitrary JSON key-value pairs keyed by (namespace, path).
// Permission enforcement (key prefix ownership) is the caller's responsibility.
type MetadataStore interface {
	// Get returns all metadata for (namespace, path).
	// Returns an empty map (not an error) if no metadata is set.
	Get(ctx context.Context, namespace, path string) (map[string]json.RawMessage, error)

	// Set merges kv into existing metadata for (namespace, path).
	// Keys in kv overwrite existing values; absent keys are unchanged.
	Set(ctx context.Context, namespace, path string, kv map[string]json.RawMessage) error

	// DeleteKeys removes specific keys from metadata for (namespace, path).
	// Deleting a non-existent key is a no-op.
	DeleteKeys(ctx context.Context, namespace, path string, keys []string) error

	// Query returns all (namespace, path) entries where the given key equals value.
	// Pass value=nil to match any entry that has the key set (existence check).
	Query(ctx context.Context, namespace, key string, value json.RawMessage) ([]MetaEntry, error)
}

// MetaEntry is a single result row from MetadataStore.Query.
type MetaEntry struct {
	Namespace string                     `json:"namespace"`
	Path      string                     `json:"path"`
	Meta      map[string]json.RawMessage `json:"meta"`
}
