// Package deputil carries the server's dependency graph: the Dependencies
// interface every layer is handed, and the constructors that build it either
// empty (for tests) or wired to the real database, storage, and VFS.
package deputil

import (
	"fmt"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/iosemutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/util/uploadutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/workerutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

type Dependencies interface {
	Database() *db.DatabaseSqlc
	EventBus() *eventbus.Bus
	FileIndex() *storageutil.FileIndex
	HealthDatabase() *db.DatabaseRaw
	IOSemaphore() *iosemutil.Semaphore
	StorageService() *storageutil.StorageService
	UploadSessions() *uploadutil.SessionStore
	VaultDB() *db.DatabaseSqlc
	VaultSession() *vaultcrypto.VaultSession
	Worker() workerutil.Worker
	WithDatabase(database *db.DatabaseSqlc) Dependencies
	WithEventBus(b *eventbus.Bus) Dependencies
	WithFileIndex(idx *storageutil.FileIndex) Dependencies
	WithHealthDatabase(healthDatabase *db.DatabaseRaw) Dependencies
	WithIOSemaphore(sem *iosemutil.Semaphore) Dependencies
	MetadataStore() vfs.MetadataStore
	VFSRegistry() vfs.Registry
	WithMetadataStore(s vfs.MetadataStore) Dependencies
	WithStorageService(s *storageutil.StorageService) Dependencies
	WithUploadSessions(store *uploadutil.SessionStore) Dependencies
	WithVFSRegistry(r vfs.Registry) Dependencies
	SetVaultDB(database *db.DatabaseSqlc)
	ClearVaultDB()
	WithVaultSession(session *vaultcrypto.VaultSession) Dependencies
	WithWorker(worker workerutil.Worker) Dependencies
}

func NewDependencies() Dependencies {
	// The upload session store is built here rather than in
	// DefaultDependencies so every dependency graph — every test engine
	// included — has a non-nil store. It allocates a map and nothing else:
	// no goroutine, no directory. StartSweeper, called once from server
	// startup, is what gives it a heartbeat (#1629).
	return &dependencies{
		uploadSessions: uploadutil.NewSessionStore(uploadutil.NewSessionStoreParams{}),
	}
}

func DefaultDependencies() (Dependencies, error) {
	deps := NewDependencies()                                // coverage: ignore - requires database connection
	if database, err := db.ConnectToDatabase(); err == nil { // coverage: ignore - requires database connection success
		deps.WithDatabase(database)
	} else { // coverage: ignore - requires database connection failure
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if database, err := db.ConnectToHealthDatabase(); err == nil { // coverage: ignore - requires health database connection success
		deps.WithHealthDatabase(database)
	} else { // coverage: ignore - requires health database connection failure
		return nil, fmt.Errorf("failed to connect to health database: %w", err)
	}
	svc := storageutil.NewStorageService(storageutil.NewDetector()) // coverage: ignore
	deps.WithStorageService(svc)                                    // coverage: ignore
	registry := vfs.NewRegistry()                                   // coverage: ignore
	_ = registry.Register(vfs.Namespace{                            // coverage: ignore
		ID:          "files",                            // coverage: ignore
		Description: "Primary vault file store (files)", // coverage: ignore
	}, vfs.NewStorageServiceVFS(svc, "files")) // coverage: ignore
	deps.WithVFSRegistry(registry)                                         // coverage: ignore
	deps.WithMetadataStore(vfs.NewSQLiteMetadataStore(deps.Database().Db)) // coverage: ignore
	deps.WithEventBus(eventbus.New())                                      // coverage: ignore
	deps.WithVaultSession(vaultcrypto.NewVaultSession())                   // coverage: ignore
	deps.WithIOSemaphore(iosemutil.New())                                  // coverage: ignore
	return deps, nil                                                       // coverage: ignore - requires database connection
}
