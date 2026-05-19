package v1_vault

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		setupVaultRoute,
		unlockVaultRoute,
		lockVaultRoute,
		vaultStatusRoute,
		listEntriesRoute,
		getEntryRoute,
		createEntryRoute,
		updateEntryRoute,
		deleteEntryRoute,
		listFoldersRoute,
		createFolderRoute,
		updateFolderRoute,
		deleteFolderRoute,
		generatePasswordRoute,
		changePasswordRoute,
		importBackupRoute,
		importVaultRoute,
		exportVaultRoute,
		getStorageLocationRoute,
		setStorageLocationRoute,
	}
}
