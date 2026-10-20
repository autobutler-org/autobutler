package vaultutil

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/sqlutil"
)

// keyFromRow maps a vault_keys row to the Key the API serves.
func keyFromRow(row db.VaultKey) Key {
	return Key{
		WrappedByPassword: row.WrappedByPassword,
		SaltPw:            row.SaltPw,
		WrappedByPhrase:   row.WrappedByPhrase,
		SaltRp:            row.SaltRp,
		KdfParams:         json.RawMessage(row.KdfParams),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

// itemFromRow maps a vault_items row to the Item the API serves.
func itemFromRow(row db.VaultItem) Item {
	return Item{
		ID:            row.ID,
		Ciphertext:    row.Ciphertext,
		Nonce:         row.Nonce,
		FormatVersion: row.FormatVersion,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

// notFound turns a query's sql.ErrNoRows into sentinel and passes any other
// error through.
func notFound(err, sentinel error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return sentinel
	}
	return err
}

// keyExists turns a unique violation on vault_keys into ErrKeyExists and
// passes any other error through.
func keyExists(err error) error {
	if sqlutil.IsUniqueConstraintErr(err) {
		return ErrKeyExists
	}
	return err
}
