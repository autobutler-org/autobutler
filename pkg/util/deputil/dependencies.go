package deputil

import (
	"fmt"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/botel/exporters/botelsqlite"
	"github.com/autobutler-org/autobutler/pkg/util/eventbus"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/autobutler-org/autobutler/pkg/util/vaultcrypto"
	"github.com/autobutler-org/autobutler/pkg/util/workerutil"
)

type Dependencies interface {
	Database() *db.DatabaseSqlc
	EventBus() *eventbus.Bus
	HealthDatabase() *db.DatabaseRaw
	MetricsExporter() *botelsqlite.TraceExporter
	StorageService() *storageutil.StorageService
	VaultSession() *vaultcrypto.VaultSession
	Worker() workerutil.Worker
	WithDatabase(database *db.DatabaseSqlc) Dependencies
	WithEventBus(b *eventbus.Bus) Dependencies
	WithHealthDatabase(healthDatabase *db.DatabaseRaw) Dependencies
	WithMetricsExporter(exporter *botelsqlite.TraceExporter) Dependencies
	WithStorageService(s *storageutil.StorageService) Dependencies
	WithVaultSession(session *vaultcrypto.VaultSession) Dependencies
	WithWorker(worker workerutil.Worker) Dependencies
}

type dependencies struct {
	database        *db.DatabaseSqlc
	eventBus        *eventbus.Bus
	healthDatabase  *db.DatabaseRaw
	metricsExporter *botelsqlite.TraceExporter
	storageService  *storageutil.StorageService
	vaultSession    *vaultcrypto.VaultSession
	worker          workerutil.Worker
}

func NewDependencies() Dependencies {
	return &dependencies{}
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
	deps.WithStorageService(storageutil.NewStorageService(storageutil.NewDetector())) // coverage: ignore
	deps.WithEventBus(eventbus.New())                                                 // coverage: ignore
	deps.WithVaultSession(vaultcrypto.NewVaultSession())                              // coverage: ignore
	return deps, nil                                                                  // coverage: ignore - requires database connection
}

func (d *dependencies) WithDatabase(database *db.DatabaseSqlc) Dependencies {
	d.database = database
	return d
}

func (d *dependencies) WithEventBus(b *eventbus.Bus) Dependencies {
	d.eventBus = b
	return d
}

func (d *dependencies) WithHealthDatabase(database *db.DatabaseRaw) Dependencies {
	d.healthDatabase = database
	return d
}

func (d *dependencies) WithMetricsExporter(exporter *botelsqlite.TraceExporter) Dependencies {
	d.metricsExporter = exporter
	return d
}

func (d *dependencies) WithStorageService(s *storageutil.StorageService) Dependencies {
	d.storageService = s
	return d
}

func (d *dependencies) WithWorker(worker workerutil.Worker) Dependencies {
	d.worker = worker
	return d
}

func (d *dependencies) Database() *db.DatabaseSqlc {
	return d.database
}

func (d *dependencies) EventBus() *eventbus.Bus {
	return d.eventBus
}

func (d *dependencies) HealthDatabase() *db.DatabaseRaw {
	return d.healthDatabase
}

func (d *dependencies) MetricsExporter() *botelsqlite.TraceExporter {
	return d.metricsExporter
}

func (d *dependencies) StorageService() *storageutil.StorageService {
	return d.storageService
}

func (d *dependencies) VaultSession() *vaultcrypto.VaultSession {
	return d.vaultSession
}

func (d *dependencies) WithVaultSession(session *vaultcrypto.VaultSession) Dependencies {
	d.vaultSession = session
	return d
}

func (d *dependencies) Worker() workerutil.Worker {
	return d.worker
}
