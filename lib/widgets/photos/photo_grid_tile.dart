import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:photo_manager/photo_manager.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/widgets/photos/photo_star_overlay.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// A single photo tile. Shared so both the desktop GridView and the mobile
/// SliverGrid render the same tile.
///
/// Quark-stored photos load their thumbnail over the network; device assets
/// come off disk through photo_manager. Selection mode swaps the tap gestures
/// for selection and paints the dim/border/checkbox overlay on top.
class PhotoGridTile extends StatelessWidget {
  const PhotoGridTile({
    required this.photo,
    required this.isSelected,
    required this.isFavorite,
    required this.selectionMode,
    required this.onOpen,
    required this.onToggleFavorite,
    required this.onToggleSelection,
    required this.onEnterSelectionMode,
    super.key,
  });

  /// The photo this tile renders — either a Quark-stored file or a device
  /// asset.
  final PhotoItem photo;

  /// Whether [photo] is part of the current selection. Drives the teal border
  /// and the filled checkbox.
  final bool isSelected;

  /// Whether [photo] is a favorite. Drives the star overlay.
  final bool isFavorite;

  /// Whether the grid is in selection mode. When true, tapping selects rather
  /// than opening the photo, and every tile shows a checkbox.
  final bool selectionMode;

  /// Called when the user taps the tile outside selection mode, to open the
  /// photo in the viewer.
  final VoidCallback onOpen;

  /// Called on a double tap of a Quark-stored photo, to toggle its favorite
  /// state.
  final VoidCallback onToggleFavorite;

  /// Called to add or remove [photo] from the selection.
  final VoidCallback onToggleSelection;

  /// Called on a long press outside selection mode, to enter selection mode.
  final VoidCallback onEnterSelectionMode;

  @override
  Widget build(BuildContext context) {
    final p = photo;
    final colorScheme = Theme.of(context).colorScheme;

    // In selection mode wrap everything with selection overlay
    Widget wrapWithSelection(Widget child) {
      return GestureDetector(
        onTap: onToggleSelection,
        onLongPress: () {
          if (!selectionMode) onEnterSelectionMode();
          onToggleSelection();
        },
        child: Stack(
          fit: StackFit.expand,
          children: [
            child,
            // Dim overlay for unselected
            if (selectionMode && !isSelected)
              Container(color: Colors.black.withValues(alpha: 0.3)),
            // Teal border for selected
            if (isSelected)
              DecoratedBox(
                decoration: BoxDecoration(
                  border: Border.all(color: colorScheme.primary, width: 3),
                ),
              ),
            // Checkbox in top-left
            if (selectionMode)
              Positioned(
                top: 6,
                left: 6,
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  width: 22,
                  height: 22,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: isSelected
                        ? colorScheme.primary
                        : Colors.transparent,
                    border: Border.all(
                      color: isSelected
                          ? colorScheme.primary
                          : Colors.white.withValues(alpha: 0.8),
                      width: 2,
                    ),
                  ),
                  child: isSelected
                      ? const Icon(
                          QuarkIcons.check,
                          size: 14,
                          color: Colors.white,
                        )
                      : null,
                ),
              ),
          ],
        ),
      );
    }

    if (p.isFiles) {
      final c = p.quark!;
      final url = FilesService.constructThumbnailUrl(
        c.apiPath,
        serial: c.deviceSerial,
      );
      Widget thumbnail = Image.network(
        url.toString(),
        fit: BoxFit.cover,
        loadingBuilder: (context, child, progress) {
          if (progress == null) return child;
          return Container(color: Colors.grey[300]);
        },
        errorBuilder: (context, error, stack) =>
            Container(color: Colors.grey[300]),
      );
      if (p.hasLiveVideo) {
        thumbnail = Stack(
          fit: StackFit.expand,
          children: [
            thumbnail,
            const Positioned(top: 4, left: 4, child: LiveBadge()),
          ],
        );
      }
      if (selectionMode) return wrapWithSelection(thumbnail);
      return MouseRegion(
        cursor: SystemMouseCursors.click,
        child: GestureDetector(
          onLongPress: () {
            onEnterSelectionMode();
            onToggleSelection();
          },
          onDoubleTap: onToggleFavorite,
          onTap: onOpen,
          child: Stack(
            fit: StackFit.expand,
            children: [
              thumbnail,
              PhotoStarOverlay(isFavorite: isFavorite),
            ],
          ),
        ),
      );
    }

    // Mobile asset
    final a = p.asset!;
    final assetThumb = FutureBuilder<Uint8List?>(
      future: a.thumbnailDataWithSize(ThumbnailSize(200, 200)),
      builder: (context, snap) {
        final thumb = snap.data;
        if (thumb == null) return Container(color: Colors.grey[300]);
        return Image.memory(thumb, fit: BoxFit.cover);
      },
    );
    if (selectionMode) return wrapWithSelection(assetThumb);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onLongPress: () {
          onEnterSelectionMode();
          onToggleSelection();
        },
        onTap: onOpen,
        child: assetThumb,
      ),
    );
  }
}
