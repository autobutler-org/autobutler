package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		setupVaultRoute,
		unlockVaultRoute,
		lockVaultRoute,
		getVaultStatusRoute,
		listVaultEntriesRoute,
		getVaultEntryRoute,
		createVaultEntryRoute,
		updateVaultEntryRoute,
		deleteVaultEntryRoute,
		listVaultFoldersRoute,
		createVaultFolderRoute,
		updateVaultFolderRoute,
		deleteVaultFolderRoute,
		generateVaultPasswordRoute,
		changeVaultPasswordRoute,
		importVaultBackupRoute,
		importVaultRoute,
		exportVaultRoute,
		getVaultStorageLocationRoute,
		setVaultStorageLocationRoute,
	}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

type generateRequest struct {
	Length         int  `json:"length"`
	Uppercase      bool `json:"uppercase"`
	Lowercase      bool `json:"lowercase"`
	Digits         bool `json:"digits"`
	Symbols        bool `json:"symbols"`
	AvoidAmbiguous bool `json:"avoidAmbiguous"`
}

type vaultStatusResponse struct {
	Initialized     bool   `json:"initialized"`
	Locked          bool   `json:"locked"`
	AutoLockSeconds int64  `json:"autoLockSeconds"`
	StorageDevice   string `json:"storageDevice"`
	DeviceConnected bool   `json:"deviceConnected"`
	LockReason      string `json:"lockReason,omitempty"`
}

type storageLocationResponse struct {
	DeviceSerial    string `json:"deviceSerial"`
	IsExternal      bool   `json:"isExternal"`
	DeviceConnected bool   `json:"deviceConnected"`
	DeviceName      string `json:"deviceName"`
}

type importBackupRequest struct {
	DeviceSerial     string `json:"deviceSerial" binding:"required"`
	RecoveryPassword string `json:"recoveryPassword" binding:"required"`
}

type setStorageLocationRequest struct {
	TargetDeviceSerial string `json:"targetDeviceSerial"`
	Username           string `json:"username" binding:"required"`
	Password           string `json:"password" binding:"required"`
}

type setupRequest struct {
	MasterPassword string `json:"masterPassword" binding:"required"`
}

type unlockRequest struct {
	MasterPassword string `json:"masterPassword" binding:"required"`
}

type createEntryRequest struct {
	Name         string        `json:"name" binding:"required"`
	URL          string        `json:"url"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Notes        string        `json:"notes"`
	TOTPSecret   string        `json:"totpSecret"`
	CustomFields []customField `json:"customFields"`
	FolderID     *int64        `json:"folderId"`
}

type createFolderRequest struct {
	Name      string `json:"name" binding:"required"`
	ParentID  *int64 `json:"parentId"`
	SortOrder int64  `json:"sortOrder"`
}
