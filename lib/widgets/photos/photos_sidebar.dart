import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/widgets/photos/album_sidebar.dart';
import 'package:quark_icons/quark_icons.dart';

/// The photos view's sidebar: the tile-size slider, the category picker, and
/// the album tree.
///
/// Desktop shows it as a fixed 280px column beside the grid; compact shows it
/// above the grid inside the page's scroll view.
class PhotosSidebar extends StatelessWidget {
  const PhotosSidebar({
    required this.selectedCategory,
    required this.quarkCount,
    required this.mobileCount,
    required this.quarkTotal,
    required this.quarkInitialLoadDone,
    required this.favoriteCount,
    required this.categoriesExpanded,
    required this.previewColumns,
    required this.minColumns,
    required this.maxColumns,
    required this.onColumnsChanged,
    required this.onToggleCategories,
    required this.onSelectCategory,
    required this.onAlbumSelected,
    this.albumSidebarKey,
    this.compact = false,
    super.key,
  });

  /// The category whose photos the grid is currently showing.
  final PhotoCategory selectedCategory;

  /// How many Quark-stored photos have been fetched so far. Used only until
  /// the server's total is known.
  final int quarkCount;

  /// How many device photos are available.
  final int mobileCount;

  /// The server's total count of Quark-stored photos, including pages that
  /// have not been fetched.
  final int quarkTotal;

  /// Whether the first page of Quark photos has come back, which is when
  /// [quarkTotal] becomes meaningful.
  final bool quarkInitialLoadDone;

  /// How many photos are marked as favorites.
  final int favoriteCount;

  /// Whether the category list under "Showing" is expanded.
  final bool categoriesExpanded;

  /// The user's chosen number of grid columns, before clamping.
  final int previewColumns;

  /// The fewest columns the grid may show at this width.
  final int minColumns;

  /// The most columns the grid may show at this width.
  final int maxColumns;

  /// Called with the new column count when the user moves the slider or taps
  /// either of the size buttons. Already clamped to [minColumns]/[maxColumns].
  final ValueChanged<int> onColumnsChanged;

  /// Called when the user taps "Showing" to expand or collapse the category
  /// list.
  final VoidCallback onToggleCategories;

  /// Called with the category the user picked.
  final ValueChanged<PhotoCategory> onSelectCategory;

  /// Called with the album the user picked in the album tree.
  final ValueChanged<PhotoAlbum> onAlbumSelected;

  /// Key for the embedded [AlbumSidebar], so the page can reload the album
  /// tree after a change made elsewhere.
  final GlobalKey<AlbumSidebarState>? albumSidebarKey;

  /// Whether to render the compact (narrow) layout, which shrink-wraps rather
  /// than filling its parent's height.
  final bool compact;

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
        onTap: () => onSelectCategory(cat),
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

    // For Quark-stored photos, show total from server (includes un-fetched pages)
    final quarkDisplayCount = quarkInitialLoadDone ? quarkTotal : quarkCount;

    final selectedLabel = switch (selectedCategory) {
      PhotoCategory.all => 'All',
      PhotoCategory.quark => 'Quark',
      PhotoCategory.mobile => 'Mobile',
      PhotoCategory.favorites => 'Favorites',
    };

    // The compact layout puts this sidebar inside a SliverToBoxAdapter, which
    // hands its child unbounded height. `Expanded` there is a hard layout
    // error — the subtree fails to lay out and the whole view renders empty,
    // silently in release builds (#1599). So compact shrink-wraps instead, and
    // the album list scrolls with the page rather than inside itself.
    final albumSidebar = AlbumSidebar(
      key: albumSidebarKey,
      selectedAlbumId: null,
      shrinkWrap: compact,
      onAlbumSelected: (album) {
        if (album == null) return;
        onAlbumSelected(album);
      },
    );

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
                    PhotoCategory.favorites => favoriteCount,
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
                  favoriteCount,
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
