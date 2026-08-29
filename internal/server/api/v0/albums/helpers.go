package v0_albums

// buildTree converts a flat album list into a nested tree.
// Uses a child-index map to avoid value-copy aliasing issues when building
// multi-level hierarchies.
func buildTree(albums []AlbumJSON) []AlbumJSON {
	// Map album ID → its children IDs.
	childIDs := make(map[int64][]int64, len(albums))
	byID := make(map[int64]AlbumJSON, len(albums))
	for _, a := range albums {
		byID[a.ID] = a
	}
	for _, a := range albums {
		if a.ParentID != nil {
			childIDs[*a.ParentID] = append(childIDs[*a.ParentID], a.ID)
		}
	}

	// build recursively populates Children before returning a copy.
	var build func(id int64) AlbumJSON
	build = func(id int64) AlbumJSON {
		a := byID[id]
		for _, cid := range childIDs[id] {
			a.Children = append(a.Children, build(cid))
		}
		return a
	}

	roots := []AlbumJSON{}
	for _, a := range albums {
		if a.ParentID == nil {
			roots = append(roots, build(a.ID))
		}
	}
	return roots
}
