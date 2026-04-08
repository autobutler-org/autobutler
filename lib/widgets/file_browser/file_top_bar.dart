import 'package:autobutler/services/storage_service.dart';
import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:autobutler/widgets/autobutler_brand_button.dart';
import 'package:autobutler/widgets/refresh_icon_button.dart';
import 'package:flutter/material.dart';

class FileTopBar extends StatefulWidget {
  const FileTopBar({
    required this.currentPath,
    required this.isGridView,
    required this.isUnifiedView,
    required this.onToggleUnifiedView,
    required this.isSearchMode,
    required this.isUploading,
    required this.isCreatingFolder,
    this.uploadTotal = 0,
    this.uploadCompleted = 0,
    required this.isRefreshing,
    required this.onGoHome,
    required this.onGoUp,
    this.onPathSelected,
    required this.onToggleView,
    required this.onSearchPressed,
    required this.onRefresh,
    required this.onUploadPressed,
    required this.onCreateFolderPressed,
    required this.onNewFilePressed,
    required this.onOpenDrawer,
    required this.onOpenSettings,
    this.devices,
    this.activeDevicePaths,
    this.onDeviceToggled,
    super.key,
  });

  final String currentPath;
  final bool isGridView;
  final bool isUnifiedView;
  final VoidCallback onToggleUnifiedView;
  final bool isSearchMode;
  final bool isUploading;
  final bool isCreatingFolder;
  final int uploadTotal;
  final int uploadCompleted;
  final bool isRefreshing;
  final VoidCallback onGoHome;
  final VoidCallback onGoUp;
  final ValueChanged<String>? onPathSelected;
  final VoidCallback onToggleView;
  final VoidCallback onSearchPressed;
  final VoidCallback onRefresh;
  final VoidCallback onUploadPressed;
  final VoidCallback onCreateFolderPressed;
  final VoidCallback onNewFilePressed;
  final VoidCallback onOpenDrawer;
  final VoidCallback onOpenSettings;
  final List<StorageDevice>? devices;
  final Set<String>? activeDevicePaths;
  final ValueChanged<String>? onDeviceToggled;

  @override
  State<FileTopBar> createState() => _FileTopBarState();
}

class _FileTopBarState extends State<FileTopBar> {
  final _viewsMenuController = MenuController();
  final _createMenuController = MenuController();

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      decoration: BoxDecoration(
        color: colorScheme.secondary,
        border: Border(bottom: BorderSide(color: colorScheme.outline)),
      ),
      child: SafeArea(
        bottom: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            _buildTopRow(context),
            if (!widget.isSearchMode) _buildPathRow(context),
          ],
        ),
      ),
    );
  }

  Widget _buildTopRow(BuildContext context) {
    return SizedBox(
      height: 56,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12),
        child: Row(
          children: [
            _buildBrand(context),
            const SizedBox(width: 16),
            _buildNavButtons(context),
            const Spacer(),
            _buildActionButtons(context),
          ],
        ),
      ),
    );
  }

  Widget _buildBrand(BuildContext context) {
    return AutobutlerBrandButton(label: 'Files', onTap: widget.onOpenDrawer);
  }

  Widget _buildNavButtons(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _iconBtn(
          context: context,
          icon: Icons.arrow_back_rounded,
          onTap: widget.currentPath.isEmpty ? null : widget.onGoUp,
          tooltip: 'Back',
        ),
        const SizedBox(width: 4),
        _iconBtn(
          context: context,
          icon: Icons.arrow_upward_rounded,
          onTap: widget.currentPath.isEmpty ? null : widget.onGoUp,
          tooltip: 'Up one level',
        ),
        const SizedBox(width: 4),
        RefreshIconButton(
          isRefreshing: widget.isRefreshing,
          onPressed: widget.onRefresh,
          tooltip: 'Refresh',
        ),
      ],
    );
  }

  Widget _buildActionButtons(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _iconBtn(
          context: context,
          icon: Icons.search_rounded,
          onTap: widget.onSearchPressed,
          tooltip: 'Search',
        ),
        const SizedBox(width: 4),
        _iconBtn(
          context: context,
          icon: Icons.settings_outlined,
          onTap: widget.onOpenSettings,
          tooltip: 'Settings',
        ),
      ],
    );
  }

  Widget _buildPathRow(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: colorScheme.outline, width: 0.5)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final isCompact = constraints.maxWidth < 860;

          if (isCompact) {
            return Row(
              children: [
                Expanded(child: _buildBreadcrumb(context)),
                const SizedBox(width: 8),
                _buildViewsMenu(context),
                const SizedBox(width: 6),
                _buildCreateMenu(context),
              ],
            );
          }

          // Wide layout — unchanged.
          return Row(
            children: [
              Expanded(child: _buildBreadcrumb(context)),
              if (widget.devices != null && widget.devices!.length > 1) ...[
                const SizedBox(width: 12),
                _buildDeviceChips(context),
              ],
              const SizedBox(width: 12),
              _buildActions(context),
              const SizedBox(width: 8),
              _buildViewChips(context),
            ],
          );
        },
      ),
    );
  }

  // ── Compact: "Views" popup menu ──────────────────────────────────────────

  Widget _buildViewsMenu(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final hasDeviceFilter =
        widget.devices != null && widget.devices!.length > 1;

    return MenuAnchor(
      controller: _viewsMenuController,
      style: MenuStyle(
        minimumSize: const WidgetStatePropertyAll(Size(220, 0)),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
          ),
        ),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(vertical: 8),
        ),
      ),
      menuChildren: [
        // ── Device filter section ──
        if (hasDeviceFilter) ...[
          _menuSectionHeader(context, 'Filter devices'),
          ...widget.devices!.map((device) {
            final isSelected =
                widget.activeDevicePaths?.contains(device.devicePath) ?? true;
            final label = device.name.isNotEmpty
                ? device.name
                : device.mountPoint;
            return CheckboxListTile(
              value: isSelected,
              dense: true,
              title: Text(label, style: const TextStyle(fontSize: 14)),
              controlAffinity: ListTileControlAffinity.leading,
              visualDensity: VisualDensity.compact,
              onChanged: widget.onDeviceToggled != null
                  ? (_) => widget.onDeviceToggled!(device.devicePath)
                  : null,
            );
          }),
          const Divider(height: 1, indent: 16, endIndent: 16),
        ],

        // ── Layout section ──
        _menuSectionHeader(context, 'Layout'),
        _menuRadioItem(
          context: context,
          icon: Icons.view_list_rounded,
          label: 'List',
          selected: !widget.isGridView,
          onTap: () {
            if (widget.isGridView) widget.onToggleView();
          },
        ),
        _menuRadioItem(
          context: context,
          icon: Icons.grid_view_rounded,
          label: 'Grid',
          selected: widget.isGridView,
          onTap: () {
            if (!widget.isGridView) widget.onToggleView();
          },
        ),
        const Divider(height: 1, indent: 16, endIndent: 16),

        // ── Device grouping section ──
        _menuSectionHeader(context, 'Grouping'),
        ListTile(
          dense: true,
          visualDensity: VisualDensity.compact,
          leading: Icon(
            widget.isUnifiedView
                ? Icons.folder_copy_outlined
                : Icons.device_hub_outlined,
            size: 18,
            color: colorScheme.onSurfaceVariant,
          ),
          title: Text(
            widget.isUnifiedView ? 'Unified' : 'Per-device',
            style: const TextStyle(fontSize: 14),
          ),
          trailing: Switch.adaptive(
            value: widget.isUnifiedView,
            onChanged: (_) => widget.onToggleUnifiedView(),
          ),
          onTap: widget.onToggleUnifiedView,
        ),
      ],
      child: _chip(
        context: context,
        icon: Icons.tune_rounded,
        label: 'Views',
        onTap: () {
          if (_viewsMenuController.isOpen) {
            _viewsMenuController.close();
          } else {
            _viewsMenuController.open();
          }
        },
      ),
    );
  }

  // ── Compact: "Create" popup menu ─────────────────────────────────────────

  Widget _buildCreateMenu(BuildContext context) {
    final uploadLabel = widget.isUploading
        ? (widget.uploadTotal > 0
              ? 'Uploading ${widget.uploadCompleted}/${widget.uploadTotal}…'
              : 'Uploading…')
        : 'Upload files';

    return MenuAnchor(
      controller: _createMenuController,
      style: MenuStyle(
        minimumSize: const WidgetStatePropertyAll(Size(200, 0)),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
          ),
        ),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(vertical: 8),
        ),
      ),
      menuChildren: [
        ListTile(
          dense: true,
          visualDensity: VisualDensity.compact,
          leading: widget.isUploading
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator.adaptive(strokeWidth: 2),
                )
              : const Icon(Icons.upload_rounded, size: 18),
          title: Text(uploadLabel, style: const TextStyle(fontSize: 14)),
          enabled: !widget.isUploading,
          onTap: () {
            _createMenuController.close();
            widget.onUploadPressed();
          },
        ),
        ListTile(
          dense: true,
          visualDensity: VisualDensity.compact,
          leading: const Icon(Icons.create_new_folder_outlined, size: 18),
          title: const Text('New folder', style: TextStyle(fontSize: 14)),
          enabled: !widget.isCreatingFolder,
          onTap: () {
            _createMenuController.close();
            widget.onCreateFolderPressed();
          },
        ),
        ListTile(
          dense: true,
          visualDensity: VisualDensity.compact,
          leading: const Icon(Icons.edit_document, size: 18),
          title: const Text('New file', style: TextStyle(fontSize: 14)),
          onTap: () {
            _createMenuController.close();
            widget.onNewFilePressed();
          },
        ),
      ],
      child: _chip(
        context: context,
        icon: Icons.add_rounded,
        label: 'Create',
        onTap: () {
          if (_createMenuController.isOpen) {
            _createMenuController.close();
          } else {
            _createMenuController.open();
          }
        },
      ),
    );
  }

  // ── Menu helpers ─────────────────────────────────────────────────────────

  Widget _menuSectionHeader(BuildContext context, String title) {
    final colorScheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
      child: Text(
        title.toUpperCase(),
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.8,
          color: colorScheme.onSurface.withValues(alpha: 0.45),
        ),
      ),
    );
  }

  Widget _menuRadioItem({
    required BuildContext context,
    required IconData icon,
    required String label,
    required bool selected,
    required VoidCallback onTap,
  }) {
    final colorScheme = Theme.of(context).colorScheme;
    return ListTile(
      dense: true,
      visualDensity: VisualDensity.compact,
      leading: Icon(
        icon,
        size: 18,
        color: selected ? colorScheme.primary : colorScheme.onSurfaceVariant,
      ),
      title: Text(
        label,
        style: TextStyle(
          fontSize: 14,
          color: selected ? colorScheme.primary : colorScheme.onSurface,
          fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
        ),
      ),
      trailing: selected
          ? Icon(Icons.check_rounded, size: 16, color: colorScheme.primary)
          : const SizedBox(width: 16),
      onTap: onTap,
    );
  }

  // ── Wide layout helpers (unchanged) ──────────────────────────────────────

  Widget _buildDeviceChips(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: widget.devices!.map((device) {
        final isSelected =
            widget.activeDevicePaths?.contains(device.devicePath) ?? true;
        return Padding(
          padding: const EdgeInsets.only(right: 6),
          child: _chip(
            context: context,
            icon: isSelected
                ? Icons.check_circle_outline_rounded
                : Icons.circle_outlined,
            label: device.name.isNotEmpty ? device.name : device.mountPoint,
            onTap: widget.onDeviceToggled != null
                ? () => widget.onDeviceToggled!(device.devicePath)
                : null,
            active: isSelected,
          ),
        );
      }).toList(),
    );
  }

  Widget _buildBreadcrumb(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        decoration: BoxDecoration(
          color: colorScheme.surfaceContainerHighest,
          border: Border.all(color: colorScheme.outline),
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            MouseRegion(
              cursor: SystemMouseCursors.click,
              child: InkWell(
                onTap: widget.onGoHome,
                borderRadius: BorderRadius.circular(4),
                child: Padding(
                  padding: const EdgeInsets.all(2),
                  child: Icon(
                    Icons.home_rounded,
                    size: 16,
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
            ),
            const SizedBox(width: 6),
            Text(
              '/',
              style: TextStyle(
                fontSize: 13,
                color: colorScheme.onSurfaceVariant,
              ),
            ),
            ..._buildCrumbs(context),
          ],
        ),
      ),
    );
  }

  List<Widget> _buildCrumbs(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    if (widget.currentPath.isEmpty) return [];
    final trimmed = widget.currentPath.startsWith('/')
        ? widget.currentPath.substring(1)
        : widget.currentPath;
    if (trimmed.isEmpty) return [];
    final segments = trimmed.split('/');
    final widgets = <Widget>[];

    for (var i = 0; i < segments.length; i++) {
      widgets.add(
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4),
          child: Icon(
            Icons.chevron_right_rounded,
            size: 14,
            color: colorScheme.onSurface.withValues(alpha: 0.4),
          ),
        ),
      );

      final isLast = i == segments.length - 1;
      final targetPath = '/${segments.take(i + 1).join('/')}';

      widgets.add(
        MouseRegion(
          cursor: isLast ? SystemMouseCursors.basic : SystemMouseCursors.click,
          child: InkWell(
            onTap: (isLast || widget.onPathSelected == null)
                ? null
                : () => widget.onPathSelected!(targetPath),
            borderRadius: BorderRadius.circular(4),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 1),
              child: Text(
                segments[i],
                style: TextStyle(
                  fontSize: 13,
                  color: isLast ? colorScheme.onSurface : colorScheme.primary,
                ),
              ),
            ),
          ),
        ),
      );
    }
    return widgets;
  }

  Widget _buildActions(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _chip(
          context: context,
          icon: Icons.upload_rounded,
          label: widget.isUploading
              ? (widget.uploadTotal > 0
                    ? '${widget.uploadCompleted}/${widget.uploadTotal}'
                    : 'Uploading...')
              : 'Upload',
          onTap: widget.isUploading ? null : widget.onUploadPressed,
        ),
        const SizedBox(width: 6),
        _chip(
          context: context,
          icon: Icons.create_new_folder_outlined,
          label: 'New folder',
          onTap: widget.isCreatingFolder ? null : widget.onCreateFolderPressed,
        ),
        const SizedBox(width: 6),
        _chip(
          context: context,
          icon: Icons.edit_document,
          label: 'New file',
          onTap: widget.onNewFilePressed,
        ),
      ],
    );
  }

  Widget _buildViewChips(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _segmentedToggle(
          context: context,
          segments: const [
            (icon: Icons.view_list_rounded, label: 'List'),
            (icon: Icons.grid_view_rounded, label: 'Grid'),
          ],
          selectedIndex: widget.isGridView ? 1 : 0,
          onSelected: (index) {
            final wantGrid = index == 1;
            if (wantGrid != widget.isGridView) {
              widget.onToggleView();
            }
          },
        ),
        const SizedBox(width: 4),
        _chip(
          context: context,
          icon: widget.isUnifiedView
              ? Icons.folder_copy_outlined
              : Icons.device_hub_outlined,
          label: widget.isUnifiedView ? 'Unified' : 'Per-device',
          onTap: widget.onToggleUnifiedView,
          active: widget.isUnifiedView,
        ),
      ],
    );
  }

  Widget _segmentedToggle({
    required BuildContext context,
    required List<({IconData icon, String label})> segments,
    required int selectedIndex,
    required ValueChanged<int> onSelected,
  }) {
    final colorScheme = Theme.of(context).colorScheme;
    final radius = BorderRadius.circular(AutobutlerColors.radiusLg);
    return Material(
      color: colorScheme.surfaceContainerHighest,
      shape: RoundedRectangleBorder(
        side: BorderSide(color: colorScheme.outline),
        borderRadius: radius,
      ),
      clipBehavior: Clip.antiAlias,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: List.generate(segments.length, (i) {
          final seg = segments[i];
          final isActive = i == selectedIndex;
          final isFirst = i == 0;
          final isLast = i == segments.length - 1;

          BorderRadius segRadius;
          if (isFirst && isLast) {
            segRadius = radius;
          } else if (isFirst) {
            segRadius = BorderRadius.only(
              topLeft: Radius.circular(AutobutlerColors.radiusLg),
              bottomLeft: Radius.circular(AutobutlerColors.radiusLg),
            );
          } else if (isLast) {
            segRadius = BorderRadius.only(
              topRight: Radius.circular(AutobutlerColors.radiusLg),
              bottomRight: Radius.circular(AutobutlerColors.radiusLg),
            );
          } else {
            segRadius = BorderRadius.zero;
          }

          return MouseRegion(
            cursor: isActive
                ? SystemMouseCursors.basic
                : SystemMouseCursors.click,
            child: Tooltip(
              message: seg.label,
              child: InkWell(
                onTap: isActive ? null : () => onSelected(i),
                borderRadius: segRadius,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 6,
                  ),
                  decoration: BoxDecoration(
                    color: isActive
                        ? colorScheme.primary.withValues(alpha: 0.12)
                        : Colors.transparent,
                    border: i > 0
                        ? Border(left: BorderSide(color: colorScheme.outline))
                        : null,
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        seg.icon,
                        size: 14,
                        color: isActive
                            ? colorScheme.primary
                            : colorScheme.onSurfaceVariant,
                      ),
                      const SizedBox(width: 6),
                      Text(
                        seg.label,
                        style: TextStyle(
                          fontSize: 13,
                          color: isActive
                              ? colorScheme.primary
                              : colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          );
        }),
      ),
    );
  }

  Widget _iconBtn({
    required BuildContext context,
    required IconData icon,
    required VoidCallback? onTap,
    required String tooltip,
  }) {
    final colorScheme = Theme.of(context).colorScheme;
    final radius = BorderRadius.circular(AutobutlerColors.radiusMd);
    return Tooltip(
      message: tooltip,
      child: MouseRegion(
        cursor: onTap != null
            ? SystemMouseCursors.click
            : SystemMouseCursors.basic,
        child: Material(
          color: colorScheme.surfaceContainerHighest,
          shape: RoundedRectangleBorder(
            side: BorderSide(color: colorScheme.outline),
            borderRadius: radius,
          ),
          clipBehavior: Clip.antiAlias,
          child: InkWell(
            onTap: onTap,
            borderRadius: radius,
            child: Padding(
              padding: const EdgeInsets.all(8),
              child: Icon(
                icon,
                size: 18,
                color: onTap != null
                    ? colorScheme.onSurfaceVariant
                    : colorScheme.onSurface.withValues(alpha: 0.3),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _chip({
    required BuildContext context,
    required IconData icon,
    required String label,
    VoidCallback? onTap,
    bool active = false,
  }) {
    final colorScheme = Theme.of(context).colorScheme;
    final radius = BorderRadius.circular(AutobutlerColors.radiusLg);
    return MouseRegion(
      cursor: onTap != null
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      child: Material(
        color: active
            ? colorScheme.primary.withValues(alpha: 0.12)
            : colorScheme.surfaceContainerHighest,
        shape: RoundedRectangleBorder(
          side: BorderSide(
            color: active
                ? colorScheme.primary.withValues(alpha: 0.3)
                : colorScheme.outline,
          ),
          borderRadius: radius,
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          borderRadius: radius,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  icon,
                  size: 14,
                  color: active
                      ? colorScheme.primary
                      : colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 6),
                Text(
                  label,
                  style: TextStyle(
                    fontSize: 13,
                    color: active
                        ? colorScheme.primary
                        : colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
