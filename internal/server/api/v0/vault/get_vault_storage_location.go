package v0_vault

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// getVaultStorageLocation godoc
// @Summary Get vault storage location
// @Description Returns which device the vault is stored on
// @Tags vault
// @Produce json
// @Success 200 {object} storageLocationResponse
// @Failure 500 {object} serverutil.Response
// @Router /vault/storage-location [get]
var getVaultStorageLocationRoute = serverutil.ApiRoute(
	"GET", "/vault/storage-location", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		ctx := c.Request.Context()
		serial, err := deps.Database().Queries.GetVaultLocation(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get vault location: %w", err))
		}

		resp := storageLocationResponse{
			DeviceSerial: serial,
			IsExternal:   serial != "",
		}

		if serial == "" {
			resp.DeviceConnected = true
			resp.DeviceName = "Internal Storage"
		} else {
			device, err := deps.StorageService().FindManagedDeviceBySerial(serial)
			if err == nil && device != nil {
				resp.DeviceConnected = true
				resp.DeviceName = device.Name
				if name, err := deps.Database().Queries.GetDeviceName(ctx, serial); err == nil && name != "" {
					resp.DeviceName = name
				}
			}
		}

		return serverutil.Ok().WithData(resp)
	},
)
