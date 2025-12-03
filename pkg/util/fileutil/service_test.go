package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileutil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalGetFilesDir := GetFilesDir
	GetFilesDir = func() string { return tmpDir }
	defer func() { GetFilesDir = originalGetFilesDir }()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := MoveFile(MoveFileParams{
		FilePath:    "test.txt",
		NewFilePath: "subdir/moved.txt",
	})

	if result.Error != nil {
		t.Errorf("MoveFile failed: %v", result.Error)
	}

	newPath := filepath.Join(tmpDir, "subdir/moved.txt")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Errorf("File was not moved to expected location: %s", newPath)
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("Original file still exists: %s", testFile)
	}

	if result.NewDir != "subdir" {
		t.Errorf("Expected NewDir to be 'subdir', got '%s'", result.NewDir)
	}
}

func TestCreateFolder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileutil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalGetFilesDir := GetFilesDir
	GetFilesDir = func() string { return tmpDir }
	defer func() { GetFilesDir = originalGetFilesDir }()

	result := CreateFolder(CreateFolderParams{
		FolderDir:  "parent",
		FolderName: "newfolder",
	})

	if result.Error != nil {
		t.Errorf("CreateFolder failed: %v", result.Error)
	}

	newFolder := filepath.Join(tmpDir, "parent", "newfolder")
	if stat, err := os.Stat(newFolder); os.IsNotExist(err) {
		t.Errorf("Folder was not created: %s", newFolder)
	} else if !stat.IsDir() {
		t.Errorf("Created path is not a directory: %s", newFolder)
	}

	if result.CurrentDir != "parent" {
		t.Errorf("Expected CurrentDir to be 'parent', got '%s'", result.CurrentDir)
	}
}

func TestDeleteFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileutil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalGetFilesDir := GetFilesDir
	GetFilesDir = func() string { return tmpDir }
	defer func() { GetFilesDir = originalGetFilesDir }()

	testDir := filepath.Join(tmpDir, "testdir")
	os.MkdirAll(testDir, 0755)
	file1 := filepath.Join(testDir, "file1.txt")
	file2 := filepath.Join(testDir, "file2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	result := DeleteFiles(DeleteFilesParams{
		RootDir:        "testdir",
		FilePaths:      []string{"file1.txt", "file2.txt"},
		ManagedDevices: nil,
	})

	if result.Error != nil {
		t.Errorf("DeleteFiles failed: %v", result.Error)
	}

	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Errorf("File1 was not deleted: %s", file1)
	}
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Errorf("File2 was not deleted: %s", file2)
	}
}
