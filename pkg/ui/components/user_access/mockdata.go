package user_access

import "time"

type User struct {
	ID       string
	Email    string
	Name     string
	Role     string
	Status   string // "active", "pending", "inactive"
	LastSeen time.Time
	Avatar   string
}

type Role struct {
	ID          string
	Name        string
	Description string
	Permissions []string
	UserCount   int
}

type Permission struct {
	ID          string
	Name        string
	Description string
	Resource    string // e.g., "files", "calendar", "settings"
	Action      string // e.g., "read", "write", "delete", "manage"
}

func GetMockUsers() []User {
	return []User{
		{
			ID:       "1",
			Email:    "admin@autobutler.ai",
			Name:     "Admin User",
			Role:     "Admin",
			Status:   "active",
			LastSeen: time.Now().Add(-30 * time.Minute),
			Avatar:   "AU",
		},
		{
			ID:       "2",
			Email:    "jane.editor@autobutler.ai",
			Name:     "Jane Editor",
			Role:     "Editor",
			Status:   "active",
			LastSeen: time.Now().Add(-2 * time.Hour),
			Avatar:   "JE",
		},
		{
			ID:       "3",
			Email:    "john.viewer@autobutler.ai",
			Name:     "John Viewer",
			Role:     "Viewer",
			Status:   "active",
			LastSeen: time.Now().Add(-1 * time.Hour),
			Avatar:   "JV",
		},
		{
			ID:       "4",
			Email:    "pending@autobutler.ai",
			Name:     "Pending User",
			Role:     "Viewer",
			Status:   "pending",
			LastSeen: time.Time{},
			Avatar:   "PU",
		},
		{
			ID:       "5",
			Email:    "inactive@autobutler.ai",
			Name:     "Inactive User",
			Role:     "Editor",
			Status:   "inactive",
			LastSeen: time.Now().Add(-720 * time.Hour),
			Avatar:   "IU",
		},
	}
}

func GetMockRoles() []Role {
	return []Role{
		{
			ID:          "admin",
			Name:        "Admin",
			Description: "Full system access with user management",
			Permissions: []string{"files.read", "files.write", "files.delete", "files.manage", "calendar.read", "calendar.write", "calendar.delete", "calendar.manage", "photos.read", "photos.write", "photos.delete", "photos.manage", "storage.read", "storage.write", "storage.manage", "settings.read", "settings.write", "settings.manage", "users.read", "users.write", "users.manage"},
			UserCount:   1,
		},
		{
			ID:          "editor",
			Name:        "Editor",
			Description: "Create and edit content without system changes",
			Permissions: []string{"files.read", "files.write", "files.delete", "calendar.read", "calendar.write", "calendar.delete", "photos.read", "photos.write", "photos.delete", "storage.read"},
			UserCount:   2,
		},
		{
			ID:          "viewer",
			Name:        "Viewer",
			Description: "Read-only access to content",
			Permissions: []string{"files.read", "calendar.read", "photos.read", "storage.read"},
			UserCount:   2,
		},
	}
}

func GetMockPermissions() []Permission {
	return []Permission{
		// Cirrus
		{ID: "files.read", Name: "View Cirrus", Description: "View and download files", Resource: "Cirrus", Action: "read"},
		{ID: "files.write", Name: "Edit Cirrus", Description: "Upload and modify files", Resource: "Cirrus", Action: "write"},
		{ID: "files.delete", Name: "Delete Cirrus Files", Description: "Delete files and folders", Resource: "Cirrus", Action: "delete"},
		{ID: "files.manage", Name: "Manage Cirrus", Description: "Configure file storage settings", Resource: "Cirrus", Action: "manage"},

		// Calendar
		{ID: "calendar.read", Name: "View Calendar", Description: "View calendar events", Resource: "Calendar", Action: "read"},
		{ID: "calendar.write", Name: "Edit Calendar", Description: "Create and modify events", Resource: "Calendar", Action: "write"},
		{ID: "calendar.delete", Name: "Delete Events", Description: "Delete calendar events", Resource: "Calendar", Action: "delete"},
		{ID: "calendar.manage", Name: "Manage Calendar", Description: "Configure calendar settings", Resource: "Calendar", Action: "manage"},

		// Photos
		{ID: "photos.read", Name: "View Photos", Description: "View photo library", Resource: "Photos", Action: "read"},
		{ID: "photos.write", Name: "Edit Photos", Description: "Upload and organize photos", Resource: "Photos", Action: "write"},
		{ID: "photos.delete", Name: "Delete Photos", Description: "Delete photos", Resource: "Photos", Action: "delete"},
		{ID: "photos.manage", Name: "Manage Photos", Description: "Configure photo settings", Resource: "Photos", Action: "manage"},

		// Storage
		{ID: "storage.read", Name: "View Storage", Description: "View storage devices", Resource: "Storage", Action: "read"},
		{ID: "storage.write", Name: "Edit Storage", Description: "Enable/disable devices", Resource: "Storage", Action: "write"},
		{ID: "storage.manage", Name: "Manage Storage", Description: "Configure storage settings", Resource: "Storage", Action: "manage"},

		// Settings
		{ID: "settings.read", Name: "View Settings", Description: "View system settings", Resource: "Settings", Action: "read"},
		{ID: "settings.write", Name: "Edit Settings", Description: "Modify system settings", Resource: "Settings", Action: "write"},
		{ID: "settings.manage", Name: "Manage Settings", Description: "Full settings control", Resource: "Settings", Action: "manage"},

		// Users
		{ID: "users.read", Name: "View Users", Description: "View user list", Resource: "Users", Action: "read"},
		{ID: "users.write", Name: "Edit Users", Description: "Invite and modify users", Resource: "Users", Action: "write"},
		{ID: "users.manage", Name: "Manage Users", Description: "Full user management", Resource: "Users", Action: "manage"},
	}
}

// Helper function to check if a role has a specific permission
func RoleHasPermission(role Role, permissionID string) bool {
	for _, perm := range role.Permissions {
		if perm == permissionID {
			return true
		}
	}
	return false
}

// Helper function to group permissions by resource
func GroupPermissionsByResource(permissions []Permission) map[string][]Permission {
	grouped := make(map[string][]Permission)
	for _, perm := range permissions {
		grouped[perm.Resource] = append(grouped[perm.Resource], perm)
	}
	return grouped
}
