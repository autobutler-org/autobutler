// Package vaultutil stores each account's end-to-end encrypted vault (#3456):
// its vault key, a random key made on the client and stored only wrapped under
// the login password and, optionally, the recovery phrase, and its items,
// encrypted on the client under that key. Entries and folders are both items;
// what an item is lives inside its ciphertext.
//
// The Quark stores and returns the bytes and never derives, holds, opens or
// checks a vault key. Every function acts on one account, named by its user
// id, and another account's item reads as not found. HTTP concerns stay with
// the caller, which maps the sentinel errors below onto status codes.
package vaultutil

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
)

// Sizes a stored vault key and item must keep to. The salts are libsodium's
// crypto_pwhash_SALTBYTES, as chat's are. A wrapped key, a nonce and a
// ciphertext are opaque here, so they are only bounded.
const (
	SaltBytes          = 16
	MaxWrappedKeyBytes = 512
	MaxKdfParamsBytes  = 512
	MaxNonceBytes      = 64
	// MaxKeyRequestBytes bounds a request body carrying a vault key.
	MaxKeyRequestBytes = 8 << 10
	// MaxItemRequestBytes bounds a request body carrying an item. The
	// ciphertext travels as base64, so an item holds about three quarters of
	// it.
	MaxItemRequestBytes = 1 << 20
)

// The Argon2id cost a stored vault key's kdfParams may record (#3648). The
// floor is the app's KdfParams.standard, which every wrap it has made uses,
// so a weaker cost is never stored for the next password change to copy. The
// ceiling keeps an unlock from asking for more than a phone can give.
const (
	minKdfOpsLimit = 3
	maxKdfOpsLimit = 10
	minKdfMemLimit = 64 << 20
	maxKdfMemLimit = 1 << 30
)

// kdfAlgorithms are the app's KdfParams schemes; anything else is refused.
var kdfAlgorithms = map[string]bool{
	"argon2id13":                      true,
	"argon2id13+hkdf-sha256":          true,
	"argon2id13+hkdf-sha256+recovery": true,
}

// Sentinel errors the caller maps onto a status code. Their text is written
// for the app to show.
var (
	// ErrKeyNotFound reports an account with no vault key, which is an
	// account with no vault.
	ErrKeyNotFound = errors.New("this account has no vault yet")
	// ErrKeyExists reports a second vault key for an account. The key its
	// items are encrypted under is never replaced, only re-wrapped.
	ErrKeyExists = errors.New("this account already has a vault")
	// ErrInvalidKey reports a vault key of the wrong size or shape.
	ErrInvalidKey = errors.New("vault key has the wrong size or shape")
	// ErrItemNotFound reports an item id the account has no item behind,
	// another account's included.
	ErrItemNotFound = errors.New("vault item not found")
	// ErrInvalidItem reports an item of the wrong size or shape.
	ErrInvalidItem = errors.New("vault item has the wrong size or shape")
)

// Key is an account's vault key as the Quark stores it: wrapped on the client
// under the login password and, optionally, the recovery phrase, as chat's
// identity is. Byte fields travel as base64.
type Key struct {
	WrappedByPassword []byte `json:"wrappedByPassword"`
	SaltPw            []byte `json:"saltPw"`
	// WrappedByPhrase and SaltRp are both present or both absent.
	WrappedByPhrase []byte `json:"wrappedByPhrase,omitempty"`
	SaltRp          []byte `json:"saltRp,omitempty"`
	// KdfParams is the client's JSON object recording the Argon2id cost.
	KdfParams json.RawMessage `json:"kdfParams" swaggertype:"object"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// ItemFields is what a client sends for an item. Byte fields travel as base64.
type ItemFields struct {
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	// FormatVersion is the client's version of the plaintext layout, at
	// least 1.
	FormatVersion int64 `json:"formatVersion"`
}

// Item is one of an account's stored items.
type Item struct {
	ID            int64     `json:"id"`
	Ciphertext    []byte    `json:"ciphertext"`
	Nonce         []byte    `json:"nonce"`
	FormatVersion int64     `json:"formatVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ValidateKey checks the sizes and shape of a vault key a client sent, so a
// malformed one is refused before anything is stored. Its kdfParams must name
// one of the app's schemes at a cost between the bounds above.
func ValidateKey(key Key) error {
	wrapped := func(b []byte) bool { return len(b) > 0 && len(b) <= MaxWrappedKeyBytes }
	within := func(n *int64, low, high int64) bool { return n != nil && *n >= low && *n <= high }
	// The pointers tell a missing cost from a zero one.
	var params struct {
		Alg      string `json:"alg"`
		OpsLimit *int64 `json:"opsLimit"`
		MemLimit *int64 `json:"memLimit"`
	}
	switch {
	case !wrapped(key.WrappedByPassword), len(key.SaltPw) != SaltBytes,
		(key.WrappedByPhrase == nil) != (key.SaltRp == nil),
		key.WrappedByPhrase != nil && (!wrapped(key.WrappedByPhrase) || len(key.SaltRp) != SaltBytes),
		len(key.KdfParams) > MaxKdfParamsBytes, json.Unmarshal(key.KdfParams, &params) != nil,
		!kdfAlgorithms[params.Alg],
		!within(params.OpsLimit, minKdfOpsLimit, maxKdfOpsLimit),
		!within(params.MemLimit, minKdfMemLimit, maxKdfMemLimit):
		return ErrInvalidKey
	}
	return nil
}

// ValidateItem checks the shape of an item a client sent. The ciphertext's
// size is bounded by the request body, not here.
func ValidateItem(fields ItemFields) error {
	if len(fields.Ciphertext) == 0 || len(fields.Nonce) == 0 || len(fields.Nonce) > MaxNonceBytes || fields.FormatVersion < 1 {
		return ErrInvalidItem
	}
	return nil
}

// GetKeyParams asks for an account's vault key.
type GetKeyParams struct {
	Ctx     context.Context
	Queries db.Querier
	UserID  int64
}

// GetKeyResult is the account's stored vault key.
type GetKeyResult struct {
	Key Key
}

// GetKey returns an account's vault key, or ErrKeyNotFound. Serve it only to
// that account, or to a recovery that proved the account's recovery key.
func GetKey(params GetKeyParams) (GetKeyResult, error) {
	row, err := params.Queries.GetVaultKey(params.Ctx, params.UserID)
	if err != nil {
		return GetKeyResult{}, notFound(err, ErrKeyNotFound)
	}
	return GetKeyResult{Key: keyFromRow(row)}, nil
}

// CreateKeyParams sets an account's vault up with its first key.
type CreateKeyParams struct {
	Ctx     context.Context
	Queries db.Querier
	UserID  int64
	Key     Key
}

// CreateKeyResult is the key as stored.
type CreateKeyResult struct {
	Key Key
}

// CreateKey validates and stores an account's first vault key, or returns
// ErrKeyExists: a client setting the vault up again must not be able to
// replace the key its items are encrypted under.
func CreateKey(params CreateKeyParams) (CreateKeyResult, error) {
	if err := ValidateKey(params.Key); err != nil {
		return CreateKeyResult{}, err
	}
	row, err := params.Queries.CreateVaultKey(params.Ctx, db.CreateVaultKeyParams{
		UserID:            params.UserID,
		WrappedByPassword: params.Key.WrappedByPassword,
		SaltPw:            params.Key.SaltPw,
		WrappedByPhrase:   params.Key.WrappedByPhrase,
		SaltRp:            params.Key.SaltRp,
		KdfParams:         string(params.Key.KdfParams),
	})
	if err != nil {
		return CreateKeyResult{}, keyExists(err)
	}
	return CreateKeyResult{Key: keyFromRow(row)}, nil
}

// PutKeyParams stores an account's vault key re-wrapped by a credential
// change. Queries may be a transaction's, so the key moves with the password
// or recovery key that wraps it.
type PutKeyParams struct {
	Ctx     context.Context
	Queries db.Querier
	UserID  int64
	Key     Key
}

// PutKeyResult is empty: the caller already holds what it stored.
type PutKeyResult struct{}

// PutKey validates and stores an account's vault key, replacing the wraps of
// the one it has, or storing it as the first when it has none, as chat keys
// are stored at a credential change.
func PutKey(params PutKeyParams) (PutKeyResult, error) {
	if err := ValidateKey(params.Key); err != nil {
		return PutKeyResult{}, err
	}
	k := params.Key
	updated, err := params.Queries.UpdateVaultKey(params.Ctx, db.UpdateVaultKeyParams{
		UserID:            params.UserID,
		WrappedByPassword: k.WrappedByPassword,
		SaltPw:            k.SaltPw,
		WrappedByPhrase:   k.WrappedByPhrase,
		SaltRp:            k.SaltRp,
		KdfParams:         string(k.KdfParams),
	})
	if err != nil || updated > 0 {
		return PutKeyResult{}, err
	}
	_, err = CreateKey(CreateKeyParams{Ctx: params.Ctx, Queries: params.Queries, UserID: params.UserID, Key: k})
	return PutKeyResult{}, err
}

// DeleteVaultParams names the account whose vault goes.
type DeleteVaultParams struct {
	Ctx      context.Context
	Database *db.DatabaseSqlc
	UserID   int64
}

// DeleteVaultResult is empty.
type DeleteVaultResult struct{}

// DeleteVault removes an account's items and vault key together, so an
// account locked out of its vault can start over. An account with no vault
// is already in that state, which is not an error.
func DeleteVault(params DeleteVaultParams) (DeleteVaultResult, error) {
	err := db.InTx(params.Ctx, params.Database.Db, func(q db.Querier) error {
		if err := q.DeleteVaultItems(params.Ctx, params.UserID); err != nil {
			return err
		}
		return q.DeleteVaultKey(params.Ctx, params.UserID)
	})
	return DeleteVaultResult{}, err
}

// ListItemsParams asks for every item of an account.
type ListItemsParams struct {
	Ctx     context.Context
	Queries db.Querier
	UserID  int64
}

// ListItemsResult is the account's items, oldest first, never nil.
type ListItemsResult struct {
	Items []Item
}

// ListItems returns every item of an account.
func ListItems(params ListItemsParams) (ListItemsResult, error) {
	rows, err := params.Queries.ListVaultItems(params.Ctx, params.UserID)
	if err != nil {
		return ListItemsResult{}, err
	}
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, itemFromRow(row))
	}
	return ListItemsResult{Items: items}, nil
}

// GetItemParams names one of an account's items.
type GetItemParams struct {
	Ctx     context.Context
	Queries db.Querier
	UserID  int64
	ID      int64
}

// GetItemResult is the item.
type GetItemResult struct {
	Item Item
}

// GetItem returns one of an account's items, or ErrItemNotFound.
func GetItem(params GetItemParams) (GetItemResult, error) {
	row, err := params.Queries.GetVaultItem(params.Ctx, db.GetVaultItemParams{ID: params.ID, UserID: params.UserID})
	if err != nil {
		return GetItemResult{}, notFound(err, ErrItemNotFound)
	}
	return GetItemResult{Item: itemFromRow(row)}, nil
}

// CreateItemParams adds an item to an account's vault.
type CreateItemParams struct {
	Ctx      context.Context
	Database *db.DatabaseSqlc
	UserID   int64
	Fields   ItemFields
}

// CreateItemResult is the item as stored.
type CreateItemResult struct {
	Item Item
}

// CreateItem validates and stores a new item, or returns ErrKeyNotFound when
// the account has no vault: nothing it stored could ever be opened.
func CreateItem(params CreateItemParams) (CreateItemResult, error) {
	if err := ValidateItem(params.Fields); err != nil {
		return CreateItemResult{}, err
	}
	var row db.VaultItem
	// One transaction, so a vault deleted alongside cannot leave this item
	// behind with no key. The insert comes first: on SQLite it takes the write
	// lock before anything is read, so a concurrent write waits under
	// busy_timeout, where a transaction that read first fails with
	// SQLITE_BUSY as soon as it tries to write. No key rolls the insert back.
	err := db.InTx(params.Ctx, params.Database.Db, func(q db.Querier) error {
		var err error
		row, err = q.CreateVaultItem(params.Ctx, db.CreateVaultItemParams{
			UserID:        params.UserID,
			Ciphertext:    params.Fields.Ciphertext,
			Nonce:         params.Fields.Nonce,
			FormatVersion: params.Fields.FormatVersion,
		})
		if err != nil {
			return err
		}
		if _, err := q.GetVaultKey(params.Ctx, params.UserID); err != nil {
			return notFound(err, ErrKeyNotFound)
		}
		return nil
	})
	if err != nil {
		return CreateItemResult{}, err
	}
	return CreateItemResult{Item: itemFromRow(row)}, nil
}

// UpdateItemParams replaces one of an account's items.
type UpdateItemParams struct {
	Ctx     context.Context
	Queries db.Querier
	UserID  int64
	ID      int64
	Fields  ItemFields
}

// UpdateItemResult is the item as stored.
type UpdateItemResult struct {
	Item Item
}

// UpdateItem validates and replaces one of an account's items, or returns
// ErrItemNotFound.
func UpdateItem(params UpdateItemParams) (UpdateItemResult, error) {
	if err := ValidateItem(params.Fields); err != nil {
		return UpdateItemResult{}, err
	}
	row, err := params.Queries.UpdateVaultItem(params.Ctx, db.UpdateVaultItemParams{
		ID:            params.ID,
		UserID:        params.UserID,
		Ciphertext:    params.Fields.Ciphertext,
		Nonce:         params.Fields.Nonce,
		FormatVersion: params.Fields.FormatVersion,
	})
	if err != nil {
		return UpdateItemResult{}, notFound(err, ErrItemNotFound)
	}
	return UpdateItemResult{Item: itemFromRow(row)}, nil
}

// DeleteItemParams names one of an account's items.
type DeleteItemParams struct {
	Ctx     context.Context
	Queries db.Querier
	UserID  int64
	ID      int64
}

// DeleteItemResult is empty.
type DeleteItemResult struct{}

// DeleteItem removes one of an account's items, or returns ErrItemNotFound.
func DeleteItem(params DeleteItemParams) (DeleteItemResult, error) {
	deleted, err := params.Queries.DeleteVaultItem(params.Ctx, db.DeleteVaultItemParams{ID: params.ID, UserID: params.UserID})
	if err == nil && deleted == 0 {
		err = ErrItemNotFound
	}
	return DeleteItemResult{}, err
}

// PublishChangedParams names whose vault changed, and which item.
type PublishChangedParams struct {
	EventBus *eventbus.Bus
	UserID   int64
	// ItemID is the item changed, or 0 for the key or the whole vault.
	ItemID int64
}

// PublishChanged tells the account's other sessions that its vault changed.
// Call it once the change has committed. A nil bus publishes nothing.
func PublishChanged(params PublishChangedParams) {
	if params.EventBus == nil {
		return
	}
	params.EventBus.Publish(eventbus.Event{
		Kind: eventbus.EventVaultChanged,
		Data: eventbus.VaultChanged{ItemID: params.ItemID, UserID: params.UserID},
	})
}
