import 'package:autobutler/services/storage_service.dart';
import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:autobutler/widgets/autobutler_brand_button.dart';
import 'package:autobutler/widgets/refresh_icon_button.dart';
import 'package:flutter/material.dart';

class FileTopBar extends StatelessWidget {
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
            if (!isSearchMode) _buildPathRow(context),
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
    return AutobutlerBrandButton(label: 'Files', onTap: onOpenDrawer);
  }

  Widget _buildNavButtons(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _iconBtn(
          context: context,
          icon: Icons.arrow_back_rounded,
          onTap: currentPath.isEmpty ? null : onGoUp,
          tooltip: 'Back',
        ),
        const SizedBox(width: 4),
        _iconBtn(
          context: context,
          icon: Icons.arrow_upward_rounded,
          onTap: currentPath.isEmpty ? null : onGoUp,
          tooltip: 'Up one level',
        ),
        const SizedBox(width: 4),
        RefreshIconButton(
          isRefreshing: isRefreshing,
          onPressed: onRefresh,
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
          onTap: onSearchPressed,
          tooltip: 'Search',
        ),
        const SizedBox(width: 4),
        _iconBtn(
          context: context,
          icon: Icons.settings_outlined,
          onTap: onOpenSettings,
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
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                _buildBreadcrumb(context),
                const SizedBox(height: 8),
                SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      if (devices != null && devices!.length > 1) ...[
                        _buildDeviceChips(context),
                        const SizedBox(width: 8),
                      ],
                      _buildActions(context),
                      const SizedBox(width: 8),
                      _buildViewChips(context),
                    ],
                  ),
                ),
              ],
            );
          }

          return Row(
            children: [
              Expanded(child: _buildBreadcrumb(context)),
              if (devices != null && devices!.length > 1) ...[
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

  Widget _buildDeviceChips(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: devices!.map((device) {
        final isSelected =
            activeDevicePaths?.contains(device.devicePath) ?? true;
        return Padding(
          padding: const EdgeInsets.only(right: 6),
          child: _chip(
            icon: isSelected
                ? Icons.check_circle_outline_rounded
                : Icons.circle_outlined,
            label: device.name.isNotEmpty ? device.name : device.mountPoint,
            onTap: onDeviceToggled != null
                ? () => onDeviceToggled!(device.devicePath)
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
                onTap: onGoHome,
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
    if (currentPath.isEmpty) return [];
    final trimmed = currentPath.startsWith('/')
        ? currentPath.substring(1)
        : currentPath;
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
            onTap: (isLast || onPathSelected == null)
                ? null
                : () => onPathSelected!(targetPath),
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

  Widget _buildDeviceChips(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: devices!.map((device) {
        final isSelected =
            activeDevicePaths?.contains(device.devicePath) ?? true;
        return Padding(
          padding: const EdgeInsets.only(right: 4),
          child: _chip(
            context: context,
            icon: Icons.devices_rounded,
            label: device.deviceName ?? device.devicePath,
            onTap: onDeviceToggled != null
                ? () => onDeviceToggled!(device.devicePath)
                : null,
            active: isSelected,
          ),
        );
      }).toList(),
    );
  }

  Widget _buildActions(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _chip(
          context: context,
          icon: Icons.upload_rounded,
          label: isUploading
              ? (uploadTotal > 0
                    ? '$uploadCompleted/$uploadTotal'
                    : 'Uploading...')
              : 'Upload',
          onTap: isUploading ? null : onUploadPressed,
        ),
        const SizedBox(width: 6),
        _chip(
          context: context,
          icon: Icons.create_new_folder_outlined,
          label: 'New folder',
          onTap: isCreatingFolder ? null : onCreateFolderPressed,
        ),
        const SizedBox(width: 6),
        _chip(
          context: context,
          icon: Icons.edit_document,
          label: 'New file',
          onTap: onNewFilePressed,
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
          selectedIndex: isGridView ? 1 : 0,
          onSelected: (index) {
            final wantGrid = index == 1;
            if (wantGrid != isGridView) {
              onToggleView();
            }
          },
        ),
        const SizedBox(width: 4),
        _chip(
          context: context,
          icon: isUnifiedView
              ? Icons.folder_copy_outlined
              : Icons.device_hub_outlined,
          label: isUnifiedView ? 'Unified' : 'Per-device',
          onTap: onToggleUnifiedView,
          active: isUnifiedView,
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
