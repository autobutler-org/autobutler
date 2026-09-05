import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The photo grid at desktop widths, with the empty states that stand in for
/// it when there is nothing to show.
class PhotoGrid extends StatelessWidget {
  final List<PhotoItem> photos;
  final int crossAxisCount;
  final PhotoCategory selectedCategory;
  final bool isLoadingMoreQuark;
  final bool quarkUnreachable;

  /// Whether to append a spinner tile for the Quark page still loading.
  final bool showLoadingIndicator;

  final bool selectionMode;
  final ScrollController scrollController;
  final RefreshCallback onRefresh;
  final IndexedWidgetBuilder tileBuilder;

  const PhotoGrid({
    super.key,
    required this.photos,
    required this.crossAxisCount,
    required this.selectedCategory,
    required this.isLoadingMoreQuark,
    required this.quarkUnreachable,
    required this.showLoadingIndicator,
    required this.selectionMode,
    required this.scrollController,
    required this.onRefresh,
    required this.tileBuilder,
  });

  @override
  Widget build(BuildContext context) {
    final itemCount = photos.length + (showLoadingIndicator ? 1 : 0);

    return RefreshIndicator(
      onRefresh: onRefresh,
      child: photos.isEmpty && !isLoadingMoreQuark
          // "No photos yet" would be a lie when the library is merely out of
          // reach, so an unreachable Quark wins over every empty state (#1637).
          ? (quarkUnreachable
                ? QuarkDisconnectedView(
                    hostAddress: AppSettings.instance.activeHost,
                    onRetry: onRefresh,
                    onManageHosts: () => context.go(AppRoutes.settings),
                  )
                : selectedCategory == PhotoCategory.favorites
                ? const EmptyStateWidget(
                    icon: QuarkIcons.star_outline_rounded,
                    headline: 'No favorites yet',
                    subtext: 'Tap ★ on any photo to save it here.',
                  )
                : const EmptyStateWidget(
                    icon: QuarkIcons.photo_library_outlined,
                    headline: 'No photos yet',
                    subtext: 'Photos you upload to Quark will appear here.',
                  ))
          : GridView.builder(
              controller: scrollController,
              padding: EdgeInsets.fromLTRB(2, 2, 2, selectionMode ? 84 : 2),
              gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: crossAxisCount,
                crossAxisSpacing: 2,
                mainAxisSpacing: 2,
              ),
              itemCount: itemCount,
              itemBuilder: (context, idx) {
                if (idx >= photos.length) {
                  return const Center(
                    child: Padding(
                      padding: EdgeInsets.all(16),
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  );
                }
                return tileBuilder(context, idx);
              },
            ),
    );
  }
}
