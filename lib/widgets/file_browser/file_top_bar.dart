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
    required this.onPathSelected,
    required this.onToggleView,
    required this.onSearchPressed,
    required this.onRefresh,
    required this.onUploadPressed,
    required this.onCreateFolderPressed,
    required this.onOpenDrawer,
    required this.onOpenSettings,
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
  final ValueChanged<String> onPathSelected;
  final VoidCallback onToggleView;
  final VoidCallback onSearchPressed;
  final VoidCallback onRefresh;
  final VoidCallback onUploadPressed;
  final VoidCallback onCreateFolderPressed;
  final VoidCallback onOpenDrawer;
  final VoidCallback onOpenSettings;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: AutobutlerColors.sidebar,
        border: Border(bottom: BorderSide(color: AutobutlerColors.border)),
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
          icon: Icons.arrow_back_rounded,
          onTap: currentPath.isEmpty ? null : onGoUp,
          tooltip: 'Back',
        ),
        const SizedBox(width: 4),
        _iconBtn(
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
          icon: Icons.search_rounded,
          onTap: onSearchPressed,
          tooltip: 'Search',
        ),
        const SizedBox(width: 4),
        _iconBtn(
          icon: Icons.settings_outlined,
          onTap: onOpenSettings,
          tooltip: 'Settings',
        ),
      ],
    );
  }

  Widget _buildPathRow(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        border: Border(
          top: BorderSide(color: AutobutlerColors.border, width: 0.5),
        ),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
        children: [
          Expanded(child: _buildBreadcrumb(context)),
          const SizedBox(width: 12),
          _buildActions(context),
          const SizedBox(width: 8),
          _buildViewChips(context),
        ],
      ),
    );
  }

  Widget _buildBreadcrumb(BuildContext context) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        decoration: BoxDecoration(
          color: AutobutlerColors.input,
          border: Border.all(color: AutobutlerColors.border),
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
                child: const Padding(
                  padding: EdgeInsets.all(2),
                  child: Icon(
                    Icons.home_rounded,
                    size: 16,
                    color: AutobutlerColors.secondaryForeground,
                  ),
                ),
              ),
            ),
            const SizedBox(width: 6),
            const Text(
              '/',
              style: TextStyle(
                fontSize: 13,
                color: AutobutlerColors.secondaryForeground,
              ),
            ),
            ..._buildCrumbs(context),
          ],
        ),
      ),
    );
  }

  List<Widget> _buildCrumbs(BuildContext context) {
    if (currentPath.isEmpty) return [];
    final segments = currentPath.substring(1).split('/');
    final widgets = <Widget>[];

    for (var i = 0; i < segments.length; i++) {
      widgets.add(
        const Padding(
          padding: EdgeInsets.symmetric(horizontal: 4),
          child: Icon(
            Icons.chevron_right_rounded,
            size: 14,
            color: AutobutlerColors.mutedForeground,
          ),
        ),
      );

      final isLast = i == segments.length - 1;
      final targetPath = '/${segments.take(i + 1).join('/')}';

      widgets.add(
        MouseRegion(
          cursor: isLast ? SystemMouseCursors.basic : SystemMouseCursors.click,
          child: InkWell(
            onTap: isLast ? null : () => onPathSelected(targetPath),
            borderRadius: BorderRadius.circular(4),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 1),
              child: Text(
                segments[i],
                style: TextStyle(
                  fontSize: 13,
                  color: isLast
                      ? AutobutlerColors.foreground
                      : AutobutlerColors.primary,
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
          icon: Icons.create_new_folder_outlined,
          label: 'New',
          onTap: isCreatingFolder ? null : onCreateFolderPressed,
        ),
      ],
    );
  }

  Widget _buildViewChips(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _segmentedToggle(
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
    required List<({IconData icon, String label})> segments,
    required int selectedIndex,
    required ValueChanged<int> onSelected,
  }) {
    final radius = BorderRadius.circular(AutobutlerColors.radiusLg);
    return Material(
      color: AutobutlerColors.input,
      shape: RoundedRectangleBorder(
        side: const BorderSide(color: AutobutlerColors.border),
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
                        ? AutobutlerColors.primary.withValues(alpha: 0.12)
                        : Colors.transparent,
                    border: i > 0
                        ? const Border(
                            left: BorderSide(color: AutobutlerColors.border),
                          )
                        : null,
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        seg.icon,
                        size: 14,
                        color: isActive
                            ? AutobutlerColors.primary
                            : AutobutlerColors.secondaryForeground,
                      ),
                      const SizedBox(width: 6),
                      Text(
                        seg.label,
                        style: TextStyle(
                          fontSize: 13,
                          color: isActive
                              ? AutobutlerColors.primary
                              : AutobutlerColors.secondaryForeground,
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
    required IconData icon,
    required VoidCallback? onTap,
    required String tooltip,
  }) {
    final radius = BorderRadius.circular(AutobutlerColors.radiusMd);
    return Tooltip(
      message: tooltip,
      child: MouseRegion(
        cursor: onTap != null
            ? SystemMouseCursors.click
            : SystemMouseCursors.basic,
        child: Material(
          color: AutobutlerColors.input,
          shape: RoundedRectangleBorder(
            side: const BorderSide(color: AutobutlerColors.border),
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
                    ? AutobutlerColors.secondaryForeground
                    : AutobutlerColors.mutedForeground,
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _chip({
    required IconData icon,
    required String label,
    VoidCallback? onTap,
    bool active = false,
  }) {
    final radius = BorderRadius.circular(AutobutlerColors.radiusLg);
    return MouseRegion(
      cursor: onTap != null
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      child: Material(
        color: active
            ? AutobutlerColors.primary.withValues(alpha: 0.12)
            : AutobutlerColors.input,
        shape: RoundedRectangleBorder(
          side: BorderSide(
            color: active
                ? AutobutlerColors.primary.withValues(alpha: 0.3)
                : AutobutlerColors.border,
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
                      ? AutobutlerColors.primary
                      : AutobutlerColors.secondaryForeground,
                ),
                const SizedBox(width: 6),
                Text(
                  label,
                  style: TextStyle(
                    fontSize: 13,
                    color: active
                        ? AutobutlerColors.primary
                        : AutobutlerColors.secondaryForeground,
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
