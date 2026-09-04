import 'package:flutter/material.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';

/// The app bar the photos page wears while photos are being selected.
///
/// Two modes in one bar: plain selection, which counts what is selected and
/// offers a way out, and adding to an album, which names the album and offers
/// a confirm button as well.
///
/// ```dart
/// PhotosSelectionAppBar(
///   selectedCount: selectedKeys.length,
///   albumName: addingToAlbum?.name,
///   onConfirm: selectedKeys.isEmpty ? null : confirmAddToAlbum,
///   onCancel: exitSelectionMode,
/// );
/// ```
class PhotosSelectionAppBar extends StatelessWidget
    implements PreferredSizeWidget {
  /// Creates the selection bar for [selectedCount] photos.
  const PhotosSelectionAppBar({
    required this.selectedCount,
    required this.onCancel,
    this.albumName,
    this.onConfirm,
    super.key,
  });

  /// How many photos are selected.
  final int selectedCount;

  /// Leaves selection mode.
  final VoidCallback onCancel;

  /// The album the selection is being added to, or null in plain selection
  /// mode.
  final String? albumName;

  /// Adds the selection to [albumName]. Null disables the confirm button, and
  /// renders none at all outside adding mode.
  final VoidCallback? onConfirm;

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return AppBar(
      backgroundColor: colorScheme.secondary,
      automaticallyImplyLeading: false,
      title: albumName != null
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  'Adding to $albumName',
                  style: TextStyle(
                    fontSize: 14,
                    color: colorScheme.primary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                if (selectedCount > 0)
                  Text(
                    '$selectedCount selected',
                    style: TextStyle(
                      fontSize: 11,
                      color: colorScheme.onSurface.withValues(alpha: 0.5),
                    ),
                  ),
              ],
            )
          : Text('$selectedCount selected'),
      actions: [
        if (albumName != null)
          TextButton(
            key: const ValueKey('photos_selection_done'),
            onPressed: onConfirm,
            child: Text('Done ($selectedCount)'),
          ),
        TextButton(
          key: const ValueKey('photos_selection_cancel'),
          onPressed: onCancel,
          child: const Text('Cancel'),
        ),
        const AppThemeToggle(),
      ],
    );
  }
}
