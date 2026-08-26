package storageutil_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTrashTestFS(t *testing.T) (filesDir string, device *storageutil.ManagedDevice) {
	t.Helper()
	tmp := t.TempDir()
	return tmp, nil // nil device → falls back to defaultFilesDir
}

func TestTrashFilesImpl_MovesFileToTrash(t *testing.T) {
	filesDir, device := newTrashTestFS(t)

	// Create a file to trash.
	fileDir := filepath.Join(filesDir, "docs")
	require.NoError(t, os.MkdirAll(fileDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(fileDir, "notes.txt"), []byte("hello"), 0o600))

	params := storageutil.TrashFilesParams{
		RootDir:   "docs",
		FilePaths: []string{"notes.txt"},
	}

	result, err := storageutil.TrashFilesImpl(params, device, filesDir)
	require.NoError(t, err)
	assert.Equal(t, "docs", result.RootDir)

	// Original file should be gone.
	_, err = os.Stat(filepath.Join(fileDir, "notes.txt"))
	assert.True(t, os.IsNotExist(err), "original file should no longer exist")

	// Trash dir should contain it.
	trashRoot := filepath.Join(filesDir, storageutil.TrashDir)
	entries, err := os.ReadDir(trashRoot)
	require.NoError(t, err)
	// One item + one metadata sidecar.
	assert.Equal(t, 2, len(entries))
}

func TestListTrashImpl_ReturnsItems(t *testing.T) {
	filesDir, device := newTrashTestFS(t)

	// Trash a file first.
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "a.txt"), []byte("x"), 0o600))
	_, err := storageutil.TrashFilesImpl(storageutil.TrashFilesParams{
		RootDir: "", FilePaths: []string{"a.txt"},
	}, device, filesDir)
	require.NoError(t, err)

	items, err := storageutil.ListTrashImpl(storageutil.ListTrashParams{}, device, filesDir)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "a.txt", filepath.Base(items[0].OriginalPath))
	assert.False(t, items[0].TrashedAt.IsZero())
}

func TestRestoreFileImpl_RestoresFile(t *testing.T) {
	filesDir, device := newTrashTestFS(t)

	subDir := filepath.Join(filesDir, "media")
	require.NoError(t, os.MkdirAll(subDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "photo.jpg"), []byte("img"), 0o600))

	_, err := storageutil.TrashFilesImpl(storageutil.TrashFilesParams{
		RootDir: "media", FilePaths: []string{"photo.jpg"},
	}, device, filesDir)
	require.NoError(t, err)

	// List to get the trash name.
	items, err := storageutil.ListTrashImpl(storageutil.ListTrashParams{}, device, filesDir)
	require.NoError(t, err)
	require.Len(t, items, 1)

	result, err := storageutil.RestoreFileImpl(storageutil.RestoreFileParams{
		TrashName: items[0].TrashName,
	}, device, filesDir)
	require.NoError(t, err)
	assert.Contains(t, result.RestoredPath, "photo.jpg")

	// File back at original location.
	_, err = os.Stat(filepath.Join(subDir, "photo.jpg"))
	assert.NoError(t, err)

	// Trash should be empty.
	items2, err := storageutil.ListTrashImpl(storageutil.ListTrashParams{}, device, filesDir)
	require.NoError(t, err)
	assert.Empty(t, items2)
}

func TestPurgeExpiredTrashImpl_DeletesOldItems(t *testing.T) {
	filesDir, device := newTrashTestFS(t)

	trashRoot := filepath.Join(filesDir, storageutil.TrashDir)
	require.NoError(t, os.MkdirAll(trashRoot, 0o700))

	// Write an already-expired item (mtime in the past).
	oldItem := filepath.Join(trashRoot, "20200101T000000Z_old.txt")
	require.NoError(t, os.WriteFile(oldItem, []byte("old"), 0o600))
	oldTime := time.Now().AddDate(0, 0, -(storageutil.TrashRetentionDays + 1))
	require.NoError(t, os.Chtimes(oldItem, oldTime, oldTime))

	// Write a recent item.
	newItem := filepath.Join(trashRoot, "20990101T000000Z_new.txt")
	require.NoError(t, os.WriteFile(newItem, []byte("new"), 0o600))

	err := storageutil.PurgeExpiredTrashImpl(device, filesDir)
	require.NoError(t, err)

	_, errOld := os.Stat(oldItem)
	assert.True(t, os.IsNotExist(errOld), "old item should be purged")

	_, errNew := os.Stat(newItem)
	assert.NoError(t, errNew, "recent item should remain")
}
