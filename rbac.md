# User Access Control UI Implementation Plan

## Overview
This document outlines the steps to build a user access control (RBAC) dashboard section within the existing Settings page at `/settings`. This is **UI-only** work and does not include backend implementation, authentication, or database integration.

---

## Current State Analysis

The settings page already exists at `/Users/brandonapol/code/autobutler/pkg/ui/views/settings.templ` with:
- A sidebar navigation with sections: General, Users & Access, Storage, Networking, Security, OpenTelemetry, Advanced
- The "Users & Access" section is already listed in the sidebar (links to `#users`)
- Existing patterns for sections (Storage, OpenTelemetry) with headers, descriptions, and content areas
- HTMX integration for dynamic content loading
- Existing component pattern (e.g., `device_manager` component)

---

## Implementation Steps

### Step 1: Create User Access Control Component Structure

**Files to create:**
1. `/Users/brandonapol/code/autobutler/pkg/ui/components/user_access/component.templ`
   - Main component file for the user access control UI
   - Should follow the pattern of `device_manager/component.templ`

**Component responsibilities:**
- Display list of users with their roles and permissions
- Show user invitation interface
- Display role management UI
- Show permission matrix/grid

---

### Step 2: Create Mock Data Types

**File to create:**
1. `/Users/brandonapol/code/autobutler/pkg/ui/components/user_access/mockdata.go`

**Mock types and data to define:**
```go
type User struct {
    ID       string
    Email    string
    Name     string
    Role     string
    Status   string // "active", "pending", "inactive"
    LastSeen time.Time
}

type Role struct {
    ID          string
    Name        string
    Description string
    Permissions []string
}

type Permission struct {
    ID          string
    Name        string
    Description string
    Resource    string // e.g., "files", "calendar", "settings"
    Action      string // e.g., "read", "write", "delete", "manage"
}

// Mock data functions
func GetMockUsers() []User { ... }
func GetMockRoles() []Role { ... }
func GetMockPermissions() []Permission { ... }
```

---

### Step 3: Build the User Access Component Template

**In `/pkg/ui/components/user_access/component.templ`**, create these sections:

#### 3.1 User List Table
- Columns: Avatar, Name, Email, Role, Status, Last Active, Actions
- Actions: Edit Role, Revoke Access, Resend Invite (for pending users)
- Empty state when no users exist
- Responsive design with horizontal scrolling on mobile

#### 3.2 Invite User Form
- Email input field
- Role dropdown selector
- Optional message field
- "Send Invitation" button
- Success/error feedback area

#### 3.3 Role Management Cards
- Card-based layout showing available roles (e.g., Admin, Editor, Viewer, Custom)
- Each card shows:
  - Role name and description
  - Number of users with this role
  - Key permissions summary
  - "Edit Role" button
- "Create Custom Role" button

#### 3.4 Permission Matrix
- Grid showing resources (rows) vs. roles (columns)
- Checkboxes indicating which role has which permission
- Resources: Files, Calendar, Photos, Settings, Storage, Users
- Actions per resource: View, Create, Edit, Delete, Manage

---

### Step 4: Add Settings Page Route Handler

**File to modify:**
`/Users/brandonapol/code/autobutler/pkg/ui/settings.go`

**Add new function:**
```go
func setupUserAccessComponent(router *gin.Engine) {
    serverutil.UiRoute(router, "/components/settings/user-access", func(c *gin.Context) templ.Component {
        // Use mock data for UI display
        users := user_access.GetMockUsers()
        roles := user_access.GetMockRoles()
        permissions := user_access.GetMockPermissions()
        
        return user_access.Component(users, roles, permissions)
    })
}
```

**Update `SetupSettingsRoutes`:**
```go
func SetupSettingsRoutes(router *gin.Engine) {
    setupSettingsView(router)
    setupThanksView(router)
    setupDeviceManagementComponent(router)
    setupUserAccessComponent(router) // Add this line
}
```

---

### Step 5: Update Settings Template with Users & Access Section

**File to modify:**
`/Users/brandonapol/code/autobutler/pkg/ui/views/settings.templ`

**Location:** Add new section after the Storage section (around line 140), before OpenTelemetry

**Section structure:**
```templ
<section id="users">
    <div class="settings-section-header">
        <svg><!-- Users icon --></svg>
        <h2>Users & Access Control</h2>
    </div>
    <div class="settings-section-description">
        <p>Manage user accounts, roles, and permissions for Autobutler.</p>
    </div>
    <div
        id="user-access-content"
        hx-get="/components/settings/user-access"
        hx-trigger="load, refresh-users from:body"
        hx-swap="innerHTML"
    >
        <p style="text-align: center; padding: 2rem; color: var(--color-gray-500);">
            Loading users...
        </p>
    </div>
</section>
```

---

### Step 6: Create CSS Styles

**File to create or modify:**
Consider adding styles to the existing stylesheet (look for where `.settings-page`, `.device-manager` styles are defined)

**CSS classes needed:**
```css
/* User Access Container */
.user-access-container { }

/* User Table */
.user-access-table { }
.user-access-table-row { }
.user-access-table-cell { }
.user-access-avatar { }
.user-access-status-badge { }

/* Invite Form */
.user-invite-form { }
.user-invite-input { }
.user-invite-button { }

/* Role Cards */
.role-card-grid { }
.role-card { }
.role-card-header { }
.role-card-stats { }
.role-card-permissions { }

/* Permission Matrix */
.permission-matrix { }
.permission-matrix-header { }
.permission-matrix-row { }
.permission-matrix-cell { }
.permission-checkbox { }

/* Action Buttons */
.user-action-menu { }
.user-action-button { }
```

Follow the existing design system patterns from the Storage section and device manager.

---

### Step 7: Add HTMX Interactions (UI Placeholders Only)

**Note:** These HTMX attributes are for UI structure only - they won't be functional without backend endpoints.

#### 7.1 Invite User Form
```html
<form 
    hx-post="/api/v1/users/invite" 
    hx-target="#user-invite-feedback"
    hx-swap="innerHTML"
>
    <!-- Form fields -->
</form>
```

#### 7.2 Update User Role
```html
<select 
    hx-put="/api/v1/users/{userId}/role"
    hx-trigger="change"
    hx-swap="none"
>
    <!-- Role options -->
</select>
```

#### 7.3 Revoke User Access
```html
<button
    hx-delete="/api/v1/users/{userId}"
    hx-confirm="Are you sure you want to revoke access for this user?"
    hx-swap="none"
>
    Revoke Access
</button>
```

#### 7.4 Toggle Permission
```html
<input 
    type="checkbox"
    hx-post="/api/v1/roles/{roleId}/permissions/{permissionId}"
    hx-trigger="change"
    hx-swap="none"
/>
```

---

### Step 8: Add Visual Feedback Elements

**Elements to include:**

#### 8.1 Empty States
- "No users yet" state with invitation CTA
- "No custom roles" state with create role CTA

#### 8.2 Loading States
- Skeleton loaders for user table rows
- Spinner for form submission
- Loading indicator for role cards

#### 8.3 Status Indicators
- Badge for user status (active/pending/inactive)
- Color-coded role badges
- Permission enabled/disabled indicators

#### 8.4 Tooltips
- Hover tooltips explaining permissions
- Help text for role descriptions
- Action button tooltips

---

### Step 9: Implement Search and Filtering (UI Only)

**Features to add:**

#### 9.1 User Search
```html
<input 
    type="search"
    placeholder="Search users by name or email..."
/>
```

#### 9.2 Role Filter
```html
<select name="filter-role">
    <option value="">All Roles</option>
    <option value="admin">Admin</option>
    <option value="editor">Editor</option>
    <option value="viewer">Viewer</option>
</select>
```

#### 9.3 Status Filter
- Filter by active/pending/inactive users
- Show count badges for each filter option

---

### Step 10: Build Responsive Mobile View

**Considerations:**

1. **User Table → Card Layout**
   - Convert table to stacked cards on mobile
   - Show avatar, name, role, and key actions
   - Expandable detail view for full information

2. **Permission Matrix → Accordion**
   - Each role becomes an accordion item
   - Permissions grouped by resource
   - Toggle switches instead of checkboxes

3. **Invite Form → Full-screen Modal**
   - Modal overlay for better mobile UX
   - Larger touch targets for form fields

---

### Step 11: Add Accessibility Features

**Implement:**

1. **ARIA Labels**
   - Label all interactive elements
   - Add role="table" and related ARIA attributes for user table
   - Proper heading hierarchy

2. **Keyboard Navigation**
   - Tab order for all interactive elements
   - Enter key to submit forms
   - Escape key to close modals

3. **Screen Reader Support**
   - Descriptive alt text for icons
   - Status announcements for form submissions
   - Live regions for dynamic content updates

4. **Focus Management**
   - Visible focus indicators
   - Focus trap in modals
   - Return focus after actions

---

### Step 12: Testing Checklist

**Before considering UI complete:**

- [ ] All sections render correctly with mock data
- [ ] Empty states display properly
- [ ] Loading states show appropriately
- [ ] Forms have proper validation styling
- [ ] Buttons have hover and active states
- [ ] Responsive design works on mobile (375px), tablet (768px), desktop (1920px)
- [ ] Color contrast meets WCAG AA standards
- [ ] Keyboard navigation works throughout
- [ ] Icons are consistent with existing design system
- [ ] Typography follows existing patterns
- [ ] Spacing is consistent with other settings sections

---

## File Structure Summary

```
pkg/ui/
├── components/
│   └── user_access/
│       ├── component.templ           (New - Main UI component)
│       ├── component_templ.go        (Generated by templ)
│       └── mockdata.go              (New - Mock data)
├── views/
│   └── settings.templ               (Modified - Add users section)
└── settings.go                      (Modified - Add route handler)
```

---

## Design System Consistency

**Follow existing patterns from:**
- Device Manager component for list/card layouts
- OpenTelemetry section for metrics and grids
- Existing form controls and button styles
- Current color scheme and typography
- Icon set (currently using Feather icons via inline SVG)

---

## Additional Considerations

### Icon Suggestions (using Feather Icons)
- Users list: `users` icon
- Add user: `user-plus` icon
- Role badge: `shield` icon
- Permissions: `lock` or `key` icon
- Active status: `check-circle` icon
- Pending status: `clock` icon
- Inactive status: `x-circle` icon

### Color Coding Suggestions
- Admin role: Red/Orange accent
- Editor role: Blue accent
- Viewer role: Green accent
- Active status: Green
- Pending status: Yellow/Orange
- Inactive status: Gray

### Placeholder Text
Use realistic placeholder text to help visualize the UI:
- "Enter email address to send invitation..."
- "Search by name, email, or role..."
- "Select a role to assign permissions..."

---

## Quick Start Command

Once files are created, regenerate templ files:
```bash
make generate
```

This will compile `.templ` files into Go code.

---

## Questions to Consider Before Building

1. **How many roles should be predefined?** (Recommend: Admin, Editor, Viewer, + Custom)
2. **What resources need permission control?** (Recommend: Files, Calendar, Photos, Storage, Settings, Users)
3. **Should there be a "Super Admin" role?** (For user management itself)
4. **Should users be able to see their own permissions?** (Consider a read-only view)
5. **How should pending invitations be displayed?** (Separate tab? Filtered view?)
6. **Should there be bulk user actions?** (Select multiple → Assign role, Revoke access)

---

**End of Implementation Plan**
