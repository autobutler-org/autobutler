import 'package:flutter/material.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark_icons/quark_icons.dart';

/// One row of the photos sidebar's category picker: a glyph, a label, a count,
/// and a check when it is the category being shown.
class PhotoCategoryTile extends StatelessWidget {
  /// Creates the row for [category].
  const PhotoCategoryTile({
    required this.category,
    required this.label,
    required this.count,
    required this.isSelected,
    required this.onTap,
    super.key,
  });

  /// The category this row picks.
  final PhotoCategory category;

  /// What the row calls the category.
  final String label;

  /// How many photos the category holds.
  final int count;

  /// Whether this is the category the grid is showing.
  final bool isSelected;

  /// Fires when the row is tapped.
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return ListTile(
      dense: true,
      visualDensity: VisualDensity.compact,
      contentPadding: EdgeInsets.zero,
      onTap: onTap,
      leading: Icon(switch (category) {
        PhotoCategory.quark => QuarkIcons.cloud,
        PhotoCategory.mobile => QuarkIcons.smartphone,
        PhotoCategory.all => QuarkIcons.photo_library,
        PhotoCategory.favorites => QuarkIcons.star_rounded,
      }, color: isSelected ? theme.colorScheme.primary : null),
      title: Text('$label: $count', style: theme.textTheme.titleMedium),
      trailing: isSelected ? const Icon(QuarkIcons.check, size: 16) : null,
    );
  }
}
