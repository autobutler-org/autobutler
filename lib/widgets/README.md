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

All widgets take colors, radii, and spacing from `QuarkTokens` in `packages/quark_widgets` (`QuarkTokens.of(context)`, or the legacy `QuarkColors` facade). Avoid hard-coded color values in widget code.

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

### `QuarkDisconnectedView` / `QuarkDisconnectedBanner`

`lib/widgets/core/quark_disconnected_state.dart`

The "can't reach your Quark right now" state, in two sizes. Use it whenever a load fails
with `isQuarkUnreachableError(error)` (`lib/utils/connection_error.dart`) — never render the
exception itself.

Pick the shape by what is left of the page:

- **`QuarkDisconnectedView`** — the page has nothing to show, so the state takes it over.
- **`QuarkDisconnectedBanner`** — the page stays usable (Settings, login), so the state sits
  above it. Defaults to `quarkTroubleshootingStepsInPlace`, which drops "in Settings" rather
  than sending the user to a page they are already on.

Neither variant says *where* on the screen the address is. Direction does not survive contact
with a real layout — login puts the host card above this state, Settings puts host management
below it — so name a destination ("in Settings") or nothing at all.

```dart
if (isQuarkUnreachableError(error)) {
  return QuarkDisconnectedView(
    onRetry: manualRefresh,
    onManageHosts: () => context.go(AppRoutes.settings),
  );
}

// A page that keeps working while disconnected:
if (_disconnected) QuarkDisconnectedBanner(onRetry: _load),
```

**`QuarkDisconnectedView` props:**

- `onRetry` — optional `VoidCallback`; omit and no "Try again" button is shown
- `onManageHosts` — optional `VoidCallback`; omit on pages that already show host management
- `manageHostsLabel` — `String`, default `'Check the address'`
- `steps` — `List<String>`, default `quarkTroubleshootingSteps`

**`QuarkDisconnectedBanner` props:**

- `onRetry` — optional `VoidCallback`
- `steps` — `List<String>`, default `quarkTroubleshootingStepsInPlace`

**Shared copy:** `quarkDisconnectedHeadline`, `quarkDisconnectedBody`, `quarkDisconnectedInline`
(one line, for a form's error text) and `quarkDisconnectedShort` (`'Not connected'`, for a row
under a banner that already explains it).

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
