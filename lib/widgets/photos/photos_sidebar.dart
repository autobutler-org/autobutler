import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark_icons/quark_icons.dart';

/// The photos page's navigation panel: tile-size slider, category counts and
/// the album list.
///
/// Named `PhotosSidebar` rather than `Sidebar` because it sits next to
/// `AlbumSidebar`, which it embeds as [albumSidebar].
class PhotosSidebar extends StatelessWidget {
  /// Shrink-wraps instead of filling the parent, for the compact layout.
  final bool compact;

  final int minColumns;
  final int maxColumns;
  final int previewColumns;
  final ValueChanged<int> onColumnsChanged;
  final PhotoCategory selectedCategory;
  final ValueChanged<PhotoCategory> onCategorySelected;
  final int quarkDisplayCount;
  final int mobileCount;
  final int favoritesCount;
  final bool categoriesExpanded;
  final VoidCallback onToggleCategories;
  final Widget albumSidebar;

  const PhotosSidebar({
    super.key,
    required this.compact,
    required this.minColumns,
    required this.maxColumns,
    required this.previewColumns,
    required this.onColumnsChanged,
    required this.selectedCategory,
    required this.onCategorySelected,
    required this.quarkDisplayCount,
    required this.mobileCount,
    required this.favoritesCount,
    required this.categoriesExpanded,
    required this.onToggleCategories,
    required this.albumSidebar,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final selectedColumns = previewColumns.clamp(minColumns, maxColumns);
    final divisions = maxColumns - minColumns;

    Widget categoryButton(PhotoCategory cat, String label, int count) {
      final selected = selectedCategory == cat;
      return ListTile(
        dense: true,
        visualDensity: VisualDensity.compact,
        contentPadding: EdgeInsets.zero,
        onTap: () => onCategorySelected(cat),
        leading: Icon(switch (cat) {
          PhotoCategory.quark => QuarkIcons.cloud,
          PhotoCategory.mobile => QuarkIcons.smartphone,
          PhotoCategory.all => QuarkIcons.photo_library,
          PhotoCategory.favorites => QuarkIcons.star_rounded,
        }, color: selected ? theme.colorScheme.primary : null),
        title: Text('$label: $count', style: theme.textTheme.titleMedium),
        trailing: selected ? const Icon(QuarkIcons.check, size: 16) : null,
      );
    }

    final selectedLabel = switch (selectedCategory) {
      PhotoCategory.all => 'All',
      PhotoCategory.quark => 'Quark',
      PhotoCategory.mobile => 'Mobile',
      PhotoCategory.favorites => 'Favorites',
    };

    // Material, not a colored Container: the category ListTiles below paint
    // their background and ink on the nearest Material ancestor, and a plain
    // ColoredBox in between would hide both (Flutter 3.47 asserts on it).
    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      child: Container(
        width: compact ? double.infinity : 280,
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: compact ? MainAxisSize.min : MainAxisSize.max,
          children: [
            Row(
              children: [
                IconButton(
                  onPressed: selectedColumns > minColumns
                      ? () => onColumnsChanged(
                          (selectedColumns - 1).clamp(minColumns, maxColumns),
                        )
                      : null,
                  icon: const Icon(QuarkIcons.crop_square_outlined),
                  tooltip: 'Larger photos',
                ),
                Expanded(
                  child: Slider(
                    min: minColumns.toDouble(),
                    max: maxColumns.toDouble(),
                    divisions: divisions > 0 ? divisions : null,
                    value: selectedColumns.toDouble(),
                    onChanged: (value) => onColumnsChanged(value.round()),
                  ),
                ),
                IconButton(
                  onPressed: selectedColumns < maxColumns
                      ? () => onColumnsChanged(
                          (selectedColumns + 1).clamp(minColumns, maxColumns),
                        )
                      : null,
                  icon: const Icon(QuarkIcons.grid_view_outlined),
                  tooltip: 'Smaller photos',
                ),
              ],
            ),
            if (!kIsWeb) ...[
              const SizedBox(height: 8),
              ListTile(
                dense: true,
                contentPadding: EdgeInsets.zero,
                title: const Text('Showing'),
                subtitle: Text(
                  '$selectedLabel: ${switch (selectedCategory) {
                    PhotoCategory.all => quarkDisplayCount + mobileCount,
                    PhotoCategory.quark => quarkDisplayCount,
                    PhotoCategory.mobile => mobileCount,
                    PhotoCategory.favorites => favoritesCount,
                  }}',
                ),
                trailing: Icon(
                  categoriesExpanded
                      ? QuarkIcons.expand_less
                      : QuarkIcons.expand_more,
                ),
                onTap: onToggleCategories,
              ),
              if (categoriesExpanded) ...[
                categoryButton(
                  PhotoCategory.all,
                  'All',
                  quarkDisplayCount + mobileCount,
                ),
                categoryButton(PhotoCategory.quark, 'Quark', quarkDisplayCount),
                categoryButton(PhotoCategory.mobile, 'Mobile', mobileCount),
                categoryButton(
                  PhotoCategory.favorites,
                  'Favorites',
                  favoritesCount,
                ),
              ],
            ],
            const SizedBox(height: 16),
            if (compact) albumSidebar else Expanded(child: albumSidebar),
          ],
        ),
      ),
    );
  }
}
