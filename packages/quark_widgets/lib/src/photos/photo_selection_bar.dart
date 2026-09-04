import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

import '../theme/quark_tokens.dart';

/// The bar that sits at the bottom of the photo grid while photos are being
/// selected: cancel, a count, and the add-to-album action.
///
/// Adding is disabled at a count of zero, so the button never opens a picker
/// that would do nothing.
///
/// Key prefixes: `photo_selection_cancel` and `photo_selection_add_to_album`.
///
/// ```dart
/// PhotoSelectionBar(
///   selectedCount: controller.selectedKeys.length,
///   onAddToAlbum: () => showAlbumPicker(context),
///   onCancel: controller.exitSelectionMode,
/// );
/// ```
class PhotoSelectionBar extends StatelessWidget {
  /// Creates the selection bar for [selectedCount] photos.
  const PhotoSelectionBar({
    required this.selectedCount,
    required this.onAddToAlbum,
    required this.onCancel,
    super.key,
  });

  /// How many photos are selected, shown in the middle of the bar.
  final int selectedCount;

  /// Opens the album picker. Not called while [selectedCount] is zero.
  final VoidCallback onAddToAlbum;

  /// Leaves selection mode.
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final tokens = QuarkTokens.of(context);

    return SafeArea(
      top: false,
      child: Container(
        decoration: BoxDecoration(
          color: colorScheme.surface,
          border: Border(top: BorderSide(color: colorScheme.outline)),
          boxShadow: [
            BoxShadow(
              color: colorScheme.shadow.withValues(alpha: 0.2),
              blurRadius: 8,
              offset: const Offset(0, -2),
            ),
          ],
        ),
        padding: EdgeInsets.symmetric(
          horizontal: tokens.spacingMd,
          vertical: tokens.spacingSm + tokens.spacingXs,
        ),
        // `spaceBetween` rather than a pair of `Spacer`s: it keeps the action
        // flush right while leaving the button the only flexible child, which
        // is what lets it give up label width instead of overflowing the row
        // on a narrow phone (#1599).
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            TextButton(
              key: const ValueKey('photo_selection_cancel'),
              onPressed: onCancel,
              child: const Text('Cancel'),
            ),
            Flexible(
              child: Text(
                '$selectedCount ${selectedCount == 1 ? 'photo' : 'photos'} '
                'selected',
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 13,
                  color: colorScheme.onSurface.withValues(alpha: 0.6),
                ),
              ),
            ),
            Flexible(
              flex: 2,
              child: FilledButton(
                key: const ValueKey('photo_selection_add_to_album'),
                onPressed: selectedCount > 0 ? onAddToAlbum : null,
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(QuarkIcons.photo_album_outlined, size: 16),
                    SizedBox(width: tokens.spacingSm),
                    const Flexible(
                      child: Text(
                        'Add to Album',
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
