import 'dart:math' as math;

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
            // Mobile: breadcrumb (with home icon + smart truncation) + Views icon.
            // Create actions move to a FAB at the page level.
            return Row(
              children: [
                Expanded(child: _buildBreadcrumb(context)),
                const SizedBox(width: 8),
                _buildViewsMenu(context),
              ],
            );
          }

          // Desktop: breadcrumb + optional device chips + create actions + view chips.
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
        iconOnly: true,
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

  // ── Smart breadcrumb (all viewports) ────────────────────────────────────
  //
  // Renders a pill container with the home icon pinned left and path segments
  // to the right. Two truncation cases are handled:
  //
  //  Case 1 — Long segment name: the name is middle-truncated to fit within a
  //            per-segment pixel cap (MyLong…Name), preserving both ends.
  //
  //  Case 2 — Too many segments: leading (ancestor) segments are dropped and a
  //            "⋯" indicator is prepended until the remainder fits the available
  //            width. The home icon is always visible.

  Widget _buildBreadcrumb(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    final trimmed = widget.currentPath.startsWith('/')
        ? widget.currentPath.substring(1)
        : widget.currentPath;
    final segments = trimmed.isEmpty ? <String>[] : trimmed.split('/');

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerHighest,
        border: Border.all(color: colorScheme.outline),
        borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
      ),
      // LayoutBuilder inside the Container so constraints.maxWidth already
      // reflects the width after the Container's padding is subtracted.
      child: LayoutBuilder(
        builder: (context, constraints) {
          return Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              // Home icon — always visible, never truncated.
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
              if (segments.isNotEmpty) ...[
                const SizedBox(width: 2),
                ..._buildSmartCrumbs(context, segments, constraints.maxWidth),
              ],
            ],
          );
        },
      ),
    );
  }

  /// Determines which segments to display given [availableWidth] (the inner
  /// width of the breadcrumb container after its padding is removed).
  ///
  /// Works by accumulating segment slots from the rightmost (current directory)
  /// towards the root. Stops as soon as adding the next segment would overflow,
  /// prepending a "⋯" indicator for any hidden ancestors.
  List<Widget> _buildSmartCrumbs(
    BuildContext context,
    List<String> segments,
    double availableWidth,
  ) {
    if (segments.isEmpty) return [];

    final colorScheme = Theme.of(context).colorScheme;

    // Space occupied by the home icon + small gap before the first crumb.
    const homeIconPx = 20.0; // Icon(16) + Padding(all(2)) = 16 + 4
    const homeGapPx = 2.0;
    // Each separator (chevron icon + horizontal padding).
    const separatorPx = 22.0; // Icon(14) + Padding(horizontal(4)) = 14 + 8
    // The "⋯" ellipsis prefix when ancestors are hidden (same visual budget).
    const ellipsisPx = 22.0;
    // Hard cap on a single segment's rendered text width.
    const maxSegmentPx = 140.0;
    // Text style used for segment labels.
    const segStyle = TextStyle(fontSize: 13);

    final budget = availableWidth - homeIconPx - homeGapPx;

    // Measure each segment, capped at maxSegmentPx.
    final segWidths = segments.map((s) {
      return math.min(_measureText(s, segStyle), maxSegmentPx);
    }).toList();

    // Slot cost for a segment = separator + text + small horizontal padding.
    List<double> slotCosts = segWidths.map((w) => separatorPx + w + 4).toList();

    // Greedily add segments from right to left until the budget is exhausted.
    double accumulated = 0;
    int visibleFrom = segments.length; // exclusive index; will shrink left

    for (int i = segments.length - 1; i >= 0; i--) {
      // If there are still ancestors to the left of i, we may need to show "⋯".
      final ellipsisNeeded = i > 0 ? ellipsisPx : 0.0;
      if (accumulated + slotCosts[i] + ellipsisNeeded <= budget) {
        accumulated += slotCosts[i];
        visibleFrom = i;
      } else {
        break;
      }
    }

    // Always show at least the current (last) directory even if it overflows.
    if (visibleFrom == segments.length) visibleFrom = segments.length - 1;

    final hasHidden = visibleFrom > 0;
    final result = <Widget>[];

    if (hasHidden) {
      result.add(
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4),
          child: Icon(
            Icons.more_horiz_rounded,
            size: 14,
            color: colorScheme.onSurface.withValues(alpha: 0.45),
          ),
        ),
      );
    }

    for (int i = visibleFrom; i < segments.length; i++) {
      // Chevron separator before each segment.
      result.add(
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
      // Middle-truncate the label if it exceeds the per-segment pixel cap.
      final label = _middleTruncate(segments[i], maxSegmentPx, segStyle);

      result.add(
        MouseRegion(
          cursor: (isLast || widget.onPathSelected == null)
              ? SystemMouseCursors.basic
              : SystemMouseCursors.click,
          child: InkWell(
            onTap: (isLast || widget.onPathSelected == null)
                ? null
                : () => widget.onPathSelected!(targetPath),
            borderRadius: BorderRadius.circular(4),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 1),
              child: Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  color: isLast ? colorScheme.onSurface : colorScheme.primary,
                ),
                maxLines: 1,
                overflow: TextOverflow.clip,
              ),
            ),
          ),
        ),
      );
    }

    return result;
  }

  /// Returns the pixel width of [text] rendered with [style].
  static double _measureText(String text, TextStyle style) {
    final painter = TextPainter(
      text: TextSpan(text: text, style: style),
      textDirection: TextDirection.ltr,
      maxLines: 1,
    )..layout(maxWidth: double.infinity);
    return painter.width;
  }

  /// Middle-truncates [text] so that its rendered width fits within [maxPx].
  ///
  /// Keeps [head] characters from the front and [tail] characters from the
  /// back, joined by the Unicode ellipsis "…". Uses binary search to find
  /// the maximum number of kept characters that still fits.
  static String _middleTruncate(String text, double maxPx, TextStyle style) {
    if (text.isEmpty || _measureText(text, style) <= maxPx) return text;

    var lo = 2; // minimum: 1 head char + ellipsis + 1 tail char
    var hi = text.length - 1;

    while (lo < hi) {
      final total = (lo + hi + 1) ~/ 2;
      final head = (total + 1) ~/ 2;
      final tail = total - head;
      final candidate =
          '${text.substring(0, head)}\u2026${text.substring(text.length - tail)}';
      if (_measureText(candidate, style) <= maxPx) {
        lo = total;
      } else {
        hi = total - 1;
      }
    }

    final head = (lo + 1) ~/ 2;
    final tail = lo - head;
    if (tail <= 0) return '${text[0]}\u2026';
    return '${text.substring(0, head)}\u2026${text.substring(text.length - tail)}';
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
    bool iconOnly = false,
  }) {
    final colorScheme = Theme.of(context).colorScheme;
    final radius = BorderRadius.circular(AutobutlerColors.radiusLg);
    final iconColor = active
        ? colorScheme.primary
        : colorScheme.onSurfaceVariant;
    return MouseRegion(
      cursor: onTap != null
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      child: Tooltip(
        message: iconOnly ? label : '',
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
              padding: iconOnly
                  ? const EdgeInsets.all(8)
                  : const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
              child: iconOnly
                  ? Icon(icon, size: 16, color: iconColor)
                  : Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(icon, size: 14, color: iconColor),
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
      ),
    );
  }
}
