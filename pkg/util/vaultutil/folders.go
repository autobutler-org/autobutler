package vaultutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/internal/db"
)

// ListFoldersParams lists the vault's folders. Folder names are stored in the
// clear, so no key is needed.
type ListFoldersParams struct {
	Queries *db.Queries
}

// ListFoldersResult carries the folders in storage order.
type ListFoldersResult struct {
	Folders []Folder
}

// CreateFolderParams creates one folder.
type CreateFolderParams struct {
	Queries *db.Queries
	Fields  FolderFields
}

// CreateFolderResult carries the stored folder.
type CreateFolderResult struct {
	Folder Folder
}

// UpdateFolderParams renames or re-parents one folder.
type UpdateFolderParams struct {
	Queries *db.Queries
	ID      int64
	Fields  FolderFields
}

// UpdateFolderResult carries the id of the folder that was updated.
type UpdateFolderResult struct {
	ID int64
}

// DeleteFolderParams removes one folder.
type DeleteFolderParams struct {
	Queries *db.Queries
	ID      int64
}

// DeleteFolderResult reports the deletion.
type DeleteFolderResult struct {
	Deleted bool
}

// ListFolders returns every folder.
func ListFolders(ctx context.Context, params ListFoldersParams) (ListFoldersResult, error) {
	rows, err := params.Queries.ListVaultFolders(ctx)
	if err != nil {
		return ListFoldersResult{}, fmt.Errorf("list folders: %w", err)
	}

	folders := make([]Folder, 0, len(rows))
	for _, r := range rows {
		folders = append(folders, Folder{
			ID:        r.ID,
			Name:      r.Name,
			ParentID:  fromNullInt64(r.ParentID),
			SortOrder: r.SortOrder,
			CreatedAt: formatTimestamp(r.CreatedAt),
		})
	}

	return ListFoldersResult{Folders: folders}, nil
}

// CreateFolder stores a new folder.
func CreateFolder(ctx context.Context, params CreateFolderParams) (CreateFolderResult, error) {
	folder, err := params.Queries.CreateVaultFolder(ctx, db.CreateVaultFolderParams{
		Name:      params.Fields.Name,
		ParentID:  nullableInt64(params.Fields.ParentID),
		SortOrder: params.Fields.SortOrder,
	})
	if err != nil {
		return CreateFolderResult{}, fmt.Errorf("create folder: %w", err)
	}

	return CreateFolderResult{Folder: Folder{
		ID:        folder.ID,
		Name:      folder.Name,
		ParentID:  fromNullInt64(folder.ParentID),
		SortOrder: folder.SortOrder,
		CreatedAt: formatTimestamp(folder.CreatedAt),
	}}, nil
}

// UpdateFolder replaces a folder's fields. It returns [ErrFolderNotFound]
// when the id has no row.
func UpdateFolder(ctx context.Context, params UpdateFolderParams) (UpdateFolderResult, error) {
	if _, err := params.Queries.GetVaultFolder(ctx, params.ID); errors.Is(err, sql.ErrNoRows) {
		return UpdateFolderResult{}, ErrFolderNotFound
	} else if err != nil {
		return UpdateFolderResult{}, fmt.Errorf("get folder: %w", err)
	}

	if err := params.Queries.UpdateVaultFolder(ctx, db.UpdateVaultFolderParams{
		Name:      params.Fields.Name,
		ParentID:  nullableInt64(params.Fields.ParentID),
		SortOrder: params.Fields.SortOrder,
		ID:        params.ID,
	}); err != nil {
		return UpdateFolderResult{}, fmt.Errorf("update folder: %w", err)
	}

	return UpdateFolderResult{ID: params.ID}, nil
}

// DeleteFolder removes one folder.
func DeleteFolder(ctx context.Context, params DeleteFolderParams) (DeleteFolderResult, error) {
	if err := params.Queries.DeleteVaultFolder(ctx, params.ID); err != nil {
		return DeleteFolderResult{}, fmt.Errorf("delete folder: %w", err)
	}
	return DeleteFolderResult{Deleted: true}, nil
}
