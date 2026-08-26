# Quark Widget Library

Shared UI components for the Quark Flutter app.

## Directory structure

```
lib/widgets/
  core/           Primitive, data-agnostic components
  layout/         App-level structural widgets
  file_browser/   File browser-specific widgets
  quark_brand_button.dart  (legacy location, imported by layout/)
  quark_drawer.dart
  refresh_icon_button.dart
  device_upload_picker.dart
```

## Theming

All widgets use `QuarkColors` (from `lib/theme/quark_colors.dart`) for colors, radii, and spacing. Avoid hard-coded color values in widget code.

---

## Core widgets

### `QuarkFileIcon`

`lib/widgets/core/quark_file_icon.dart`

Renders the appropriate icon for a `FileNode` based on file extension. Canonical single source for icon-per-filetype logic.

```dart
QuarkFileIcon(node: myFile)
QuarkFileIcon(node: myFile, size: 48, color: Colors.grey)
```

**Props:**

- `node` (required) — `FileNode`
- `size` — double, default `20.0`
- `color` — optional `Color`

**Static helper:** `QuarkFileIcon.iconForNode(node)` returns just the `IconData` when you need the icon without a widget.

---

### `QuarkStorageBar`

`lib/widgets/core/quark_storage_bar.dart`

Animated horizontal bar showing storage usage. Color changes at 75% (amber) and 90% (red).

```dart
QuarkStorageBar(usedFraction: 0.65)
QuarkStorageBar(usedFraction: device.usedPercent / 100, height: 10)
```

**Props:**

- `usedFraction` (required) — double 0.0–1.0
- `height` — double, default `8.0`

**Static helper:** `QuarkStorageBar.colorForFraction(fraction)` returns the `Color` for a given fraction.

---

### `EmptyStateWidget`

`lib/widgets/core/empty_state_widget.dart`

Centered empty state with icon, headline, optional subtext, and optional action widget.

```dart
const EmptyStateWidget(
  icon: Icons.folder_open_outlined,
  headline: 'No files yet',
  subtext: 'Upload files using the button above.',
)

EmptyStateWidget(
  icon: Icons.storage_outlined,
  headline: 'Connect to your Quark',
  subtext: 'Enter the address of your device on your local network.',
  action: ElevatedButton(
    onPressed: _openSettings,
    child: const Text('Add target host'),
  ),
)
```

**Props:**

- `icon` (required) — `IconData`
- `headline` (required) — `String`
- `subtext` — optional `String`
- `action` — optional `Widget` (shown below subtext)

---

## Layout widgets

### `QuarkAppBar`

`lib/widgets/layout/quark_app_bar.dart`

Standard app bar with brand button as leading widget and optional actions. Implements `PreferredSizeWidget` so it can be used directly as `Scaffold.appBar`.

```dart
QuarkAppBar(
  label: 'Health',
  icon: Icons.monitor_heart_outlined,
  actions: [
    RefreshIconButton(isRefreshing: isRefreshing, onPressed: refresh),
  ],
)

// Settings page (no actions):
const QuarkAppBar(
  label: 'Settings',
  icon: Icons.settings_outlined,
)
```

**Props:**

- `label` (required) — `String` page title shown next to the icon
- `icon` (required) — `IconData`
- `actions` — `List<Widget>`, default empty

---

### `QuarkBrandButton`

`lib/widgets/quark_brand_button.dart`

The tappable brand logo + label button used as the AppBar leading widget. Prefer `QuarkAppBar` for new pages — use this directly only if you need a custom `AppBar` layout.

```dart
QuarkBrandButton(
  label: 'Files',
  icon: Icons.storage_rounded,
  onTap: () => Scaffold.of(context).openDrawer(),
)
```

When used as `AppBar.leading`, set `AppBar.leadingWidth: QuarkBrandButton.preferredWidth`.

---

### `QuarkDrawer`

`lib/widgets/quark_drawer.dart`

Navigation drawer. Pass the active section and routing callbacks for each nav item.

---

## Utility widgets

### `RefreshIconButton`

`lib/widgets/refresh_icon_button.dart`

Animated refresh icon button. Shows a spinner when `isRefreshing` is true.

### `DeviceUploadPicker`

`lib/widgets/device_upload_picker.dart`

Radio-list dialog for selecting a target storage device before uploading. Used by the file browser upload flow when multiple devices are present.
