import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:photo_manager/photo_manager.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/widgets/photos/star_overlay.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// One photo in the grid — a Quark thumbnail or a device asset — with its
/// selection, favorite and open gestures.
class PhotoTile extends StatelessWidget {
  final PhotoItem item;
  final bool isSelected;
  final bool isFavorite;
  final bool selectionMode;
  final VoidCallback onOpen;
  final VoidCallback onToggleSelection;
  final VoidCallback onEnterSelectionMode;
  final VoidCallback onToggleFavorite;

  const PhotoTile({
    super.key,
    required this.item,
    required this.isSelected,
    required this.isFavorite,
    required this.selectionMode,
    required this.onOpen,
    required this.onToggleSelection,
    required this.onEnterSelectionMode,
    required this.onToggleFavorite,
  });

  @override
  Widget build(BuildContext context) {
    final p = item;
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
              StarOverlay(isFavorite: isFavorite),
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
