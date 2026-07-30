package v0_photos

import (
	"fmt"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/photoutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// DuplicateGroupJSON represents a group of photos that are exact or near
// duplicates of each other.
type DuplicateGroupJSON struct {
	// Kind is "exact" (identical content hash) or "near" (perceptual hash
	// Hamming distance within the configured threshold).
	Kind string `json:"kind"`
	// Photos is the list of photos in this duplicate group.
	Photos []DuplicatePhotoJSON `json:"photos"`
}

// DuplicatePhotoJSON is a minimal photo reference inside a duplicate group.
type DuplicatePhotoJSON struct {
	DeviceSerial string `json:"deviceSerial"`
	RelPath      string `json:"relPath"`
}

// listDuplicates godoc
// @Summary List duplicate photos
// @Description Returns groups of exact duplicates (same SHA-256 content hash) and near-duplicates (perceptual dHash Hamming distance within threshold). Requires photo hashes to have been computed via the thumbnail or hash-index endpoints.
// @Tags photos
// @Produce json
// @Param threshold query int false "Hamming distance threshold for near-duplicates (default 10, max 20)"
// @Success 200 {object} object{groups=[]DuplicateGroupJSON}
// @Failure 500 {object} serverutil.Response
// @Router /photos/duplicates [get]
func listDuplicates(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
	}

	threshold := 10
	if raw := c.Query("threshold"); raw != "" {
		var t int
		if _, err := fmt.Sscanf(raw, "%d", &t); err == nil && t >= 0 && t <= 20 {
			threshold = t
		}
	}

	ctx := c.Request.Context()
	queries := deps.Database().Queries

	var groups []DuplicateGroupJSON

	// --- Exact duplicates (same SHA-256 content hash) ---
	exactRows, err := queries.ListExactDuplicates(ctx)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to list exact duplicates: %w", err))
	}

	// Group rows by content_hash.
	exactGroups := map[string][]DuplicatePhotoJSON{}
	for _, row := range exactRows {
		if !row.ContentHash.Valid {
			continue
		}
		key := row.ContentHash.String
		exactGroups[key] = append(exactGroups[key], DuplicatePhotoJSON{
			DeviceSerial: row.DeviceSerial,
			RelPath:      row.RelPath,
		})
	}
	for _, photos := range exactGroups {
		if len(photos) > 1 {
			groups = append(groups, DuplicateGroupJSON{Kind: "exact", Photos: photos})
		}
	}

	// --- Near-duplicates (dHash Hamming distance <= threshold) ---
	// ListNearDuplicates returns rows ordered by dhash. Adjacent rows in
	// sorted order are the most likely near-duplicate candidates — compare
	// each consecutive pair and group matches.
	nearRows, err := queries.ListNearDuplicates(ctx)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to list near duplicates: %w", err))
	}

	// Simple O(n²) cluster detection for a small library. For large libraries
	// a VP-tree or BK-tree would be more efficient, but n is bounded by the
	// number of photos indexed and this runs server-side rarely.
	seen := map[int]bool{}
	for i := 0; i < len(nearRows); i++ {
		if seen[i] || !nearRows[i].Dhash.Valid {
			continue
		}
		var group []DuplicatePhotoJSON
		group = append(group, DuplicatePhotoJSON{
			DeviceSerial: nearRows[i].DeviceSerial,
			RelPath:      nearRows[i].RelPath,
		})
		for j := i + 1; j < len(nearRows); j++ {
			if !nearRows[j].Dhash.Valid {
				continue
			}
			d := photoutil.HammingDistanceHex(nearRows[i].Dhash.String, nearRows[j].Dhash.String)
			if d >= 0 && d <= threshold {
				group = append(group, DuplicatePhotoJSON{
					DeviceSerial: nearRows[j].DeviceSerial,
					RelPath:      nearRows[j].RelPath,
				})
				seen[j] = true
			}
		}
		if len(group) > 1 {
			groups = append(groups, DuplicateGroupJSON{Kind: "near", Photos: group})
		}
		seen[i] = true
	}

	if groups == nil {
		groups = []DuplicateGroupJSON{}
	}

	return serverutil.Ok().WithData(gin.H{"groups": groups})
}

var listDuplicatesRoute = serverutil.ApiRoute("GET", "/photos/duplicates", listDuplicates)
