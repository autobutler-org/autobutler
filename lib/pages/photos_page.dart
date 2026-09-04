import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:photo_manager/photo_manager.dart';
import 'package:quark/controllers/photo_bytes_cache.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/pages/album_page.dart';
import 'package:quark/pages/image_viewer_page.dart';
import 'package:quark/router.dart';
import 'package:quark/services/album_service.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/services/favorites_service.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/auto_refresh_mixin.dart';
import 'package:quark/utils/connection_error.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/device_upload_picker.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:quark/widgets/photos/album_sidebar.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';
import 'package:quark/widgets/photos/album_picker_sheet.dart';

class PhotosPage extends StatefulWidget {
  const PhotosPage({this.addingToAlbum, super.key});

  /// When set, the page opens in "adding to album" mode — selection mode is
  /// immediately active and the header shows the album name.
  final PhotoAlbum? addingToAlbum;

  @override
  State<PhotosPage> createState() => PhotosPageState();
}

enum PhotoCategory { quark, mobile, all, favorites }

class PhotoItem {
  final FileNode? quark;
  final AssetEntity? asset;
  final bool isFiles;
  final bool hasLiveVideo;

  PhotoItem._({this.quark, this.asset, this.hasLiveVideo = false})
    : isFiles = quark != null;

  factory PhotoItem.fromFiles(FileNode c, {bool hasLiveVideo = false}) =>
      PhotoItem._(quark: c, hasLiveVideo: hasLiveVideo);
  factory PhotoItem.fromAsset(AssetEntity a) => PhotoItem._(asset: a);

  /// Stable key for use in sets (favorites, selection).
  String get selectionKey {
    if (isFiles) {
      return '${quark!.deviceSerial}:${quark!.apiPath}';
    }
    return 'asset:${asset!.id}';
  }
}

class PhotosPageState extends State<PhotosPage>
    with WidgetsBindingObserver, AutoRefreshMixin {
  static const int _defaultCrossAxisCount = 4;
  static const int _minPreviewColumns = 1;
  static const int _maxPreviewColumns = 8;
  static const double _minTileWidth = 80;
  static const int _pageSize = 50;

  Future<List<PhotoItem>> _photosFuture = Future.value(const <PhotoItem>[]);

  // Quark-device pagination state
  List<PhotoItem> _quarkPhotos = <PhotoItem>[];
  int _quarkTotal = 0;
  int _quarkOffset = 0;
  bool _isLoadingMoreQuark = false;
  bool _quarkInitialLoadDone = false;

  List<PhotoItem> _mobilePhotos = const <PhotoItem>[];

  bool _noHostSelected = false;

  /// Whether the last attempt to list Quark-stored photos never reached the
  /// Quark (#1637).
  ///
  /// Without this the page swallows the failure and renders "No photos yet",
  /// which tells the user their library is empty when in fact it is simply out
  /// of reach — the most misleading state in the app.
  bool _quarkUnreachable = false;

  bool _categoriesExpanded = false;
  bool _isUploading = false;
  int _previewColumns = _defaultCrossAxisCount;
  PhotoCategory _selectedCategory = PhotoCategory.quark;

  // Favorites: set of selectionKeys for photos that are favorited.
  // Fetched eagerly from the server on every refresh via FavoritesService.listFavoriteKeys().
  final Set<String> _favoriteKeys = {};

  // Above-viewport nav: the hidden nav panel is measured once on first layout,
  // then the scroll controller's initial offset is set so the photo grid is
  // flush with the top of the viewport. Scroll up to reveal the nav.
  //
  // We can't know the nav height before layout, so we use a two-pass approach:
  //  1. First render: nav is visible briefly at offset 0
  //  2. After layout: measure nav height, recreate the scroll controller with
  //     that initialScrollOffset, setState to rebuild — nav is now above viewport
  //
  // This is the only reliable way: initialScrollOffset is set before the first
  // frame the user sees (WidgetsBinding post-frame), so there's no visible flash.
  final GlobalKey _navPanelKey = GlobalKey();
  final GlobalKey<AlbumSidebarState> _albumSidebarKey =
      GlobalKey<AlbumSidebarState>();
  bool _navScrollInitialized = false;
  bool _showScrollHint = true;

  // Selection mode
  bool _selectionMode = false;
  bool _isOpeningPhoto = false;
  final Set<String> _selectedKeys = {};
  // When non-null, we're in "adding to album" mode (Flow 2)
  PhotoAlbum? _addingToAlbum;

  ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _scheduleNavMeasure();
    // If launched in adding-to-album mode, enter selection mode immediately
    if (widget.addingToAlbum != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _enterSelectionMode(addingToAlbum: widget.addingToAlbum);
      });
    }
  }

  /// Frames to wait for the nav panel to report a size before giving up.
  ///
  /// Every failure path here used to re-post itself unconditionally, so the
  /// three retries below were an unbounded frame loop. It ran forever on
  /// desktop, where the nav panel is never built at all, and forever in compact
  /// mode too while the sidebar's layout error stopped it ever getting a size —
  /// a permanent frame loop behind an already-blank screen. Nothing checked
  /// `mounted` either, so the chain outlived the State it belonged to (#1599).
  static const int _navMeasureMaxFrames = 20;

  /// Whether the most recent build used the compact layout. Only that layout
  /// builds a nav panel, so it is the only one with anything to measure.
  bool _compactLayout = false;
  int _navMeasureAttempts = 0;
  bool _navMeasureScheduled = false;

  /// True once the nav measurement has finished — either it jumped the scroll
  /// view past the nav panel, or it gave up. Either way nothing is still
  /// scheduling frames.
  @visibleForTesting
  bool get navScrollSettled => _navScrollInitialized;

  /// Frames spent waiting for the nav panel to report a size.
  @visibleForTesting
  int get navMeasureAttempts => _navMeasureAttempts;

  /// Schedules a measurement attempt, at most one outstanding at a time.
  void _scheduleNavMeasure() {
    if (_navScrollInitialized || _navMeasureScheduled) return;
    _navMeasureScheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _navMeasureScheduled = false;
      _measureAndJumpNav();
    });
  }

  void _measureAndJumpNav() {
    if (_navScrollInitialized || !mounted) return;
    // Desktop lays the sidebar out in a bounded Row with no nav panel, so
    // there is nothing to measure and nothing to wait for.
    if (!_compactLayout) return;

    void retry() {
      _navMeasureAttempts++;
      if (_navMeasureAttempts >= _navMeasureMaxFrames) {
        // Give up rather than spin. The page simply stays scrolled to the top
        // with the nav panel visible — cosmetic, and the user can scroll.
        _navScrollInitialized = true;
        return;
      }
      _scheduleNavMeasure();
    }

    final ctx = _navPanelKey.currentContext;
    if (ctx == null) {
      // Nav not yet in tree — retry next frame
      retry();
      return;
    }
    final box = ctx.findRenderObject() as RenderBox?;
    if (box == null || !box.hasSize) {
      retry();
      return;
    }
    final navHeight = box.size.height;
    if (navHeight <= 0) {
      retry();
      return;
    }
    // Recreate the scroll controller with the nav height as initial offset.
    // This ensures the scroll position starts at the right place even before
    // content has loaded (no dependency on maxScrollExtent).
    final oldController = _scrollController;
    final newController = ScrollController(initialScrollOffset: navHeight);
    newController.addListener(_onScroll);
    setState(() {
      _scrollController = newController;
      _navScrollInitialized = true;
    });
    oldController.removeListener(_onScroll);
    oldController.dispose();
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (!_scrollController.hasClients) return;

    final currentScroll = _scrollController.position.pixels;
    final maxScroll = _scrollController.position.maxScrollExtent;

    // Try to initialize the nav scroll if it hasn't happened yet (photos may
    // have loaded after the first frame callback fired).
    if (!_navScrollInitialized) _measureAndJumpNav();

    // Hide the scroll hint once the user starts scrolling down.
    if (_showScrollHint && currentScroll > 0) {
      setState(() => _showScrollHint = false);
    }

    // Trigger fetch when scrolled past 80%
    if (currentScroll >= maxScroll * 0.8) {
      _loadMoreQuarkPhotos();
    }
  }

  /// Load the next page of Quark-stored photos via the paginated endpoint.
  Future<void> _loadMoreQuarkPhotos() async {
    if (_isLoadingMoreQuark) return;
    if (_quarkOffset >= _quarkTotal && _quarkInitialLoadDone) return;

    setState(() {
      _isLoadingMoreQuark = true;
    });

    try {
      final response = await FilesService.getPhotos(
        offset: _quarkOffset,
        limit: _pageSize,
      );

      final newPhotos = response.photos
          .map(
            (p) => PhotoItem.fromFiles(
              FileNode(
                name: p.fileName,
                size: p.size,
                isDir: false,
                deviceName: '',
                devicePath: '',
                deviceSerial: p.serial,
                dirPath: p.relPath,
              ),
              hasLiveVideo: p.hasLiveVideo,
            ),
          )
          .toList(growable: false);

      setState(() {
        _quarkPhotos = [..._quarkPhotos, ...newPhotos];
        _quarkTotal = response.total;
        _quarkOffset += newPhotos.length;
        _quarkInitialLoadDone = true;
        _isLoadingMoreQuark = false;
        // Rebuild the future so FutureBuilder picks up the new list
        _photosFuture = _photosForCategory(_selectedCategory);
      });
    } catch (_) {
      debugPrint('[photos_page.dart] Error loading more Quark-stored photos');
      setState(() {
        _isLoadingMoreQuark = false;
        _quarkInitialLoadDone = true;
      });
    }
  }

  /// Initial load of Quark-stored photos (first page).
  Future<List<PhotoItem>> _loadQuarkPhotos() async {
    if (_noHostSelected) {
      // No host is a different state with its own UI, so a flag left over
      // from the host that was just removed must not outlive it.
      _quarkUnreachable = false;
      return const <PhotoItem>[];
    }

    _quarkPhotos = <PhotoItem>[];
    _quarkOffset = 0;
    _quarkTotal = 0;
    _quarkInitialLoadDone = false;

    try {
      final response = await FilesService.getPhotos(
        offset: 0,
        limit: _pageSize,
      );

      final items = response.photos
          .map(
            (p) => PhotoItem.fromFiles(
              FileNode(
                name: p.fileName,
                size: p.size,
                isDir: false,
                deviceName: '',
                devicePath: '',
                deviceSerial: p.serial,
                dirPath: p.relPath,
              ),
              hasLiveVideo: p.hasLiveVideo,
            ),
          )
          .toList(growable: false);

      _quarkPhotos = items;
      _quarkTotal = response.total;
      _quarkOffset = items.length;
      _quarkInitialLoadDone = true;
      _quarkUnreachable = false;
      return items;
    } catch (e) {
      debugPrint(
        '[photos_page.dart] Error loading initial Quark-stored photos: $e',
      );
      _quarkInitialLoadDone = true;
      _quarkUnreachable = isQuarkUnreachableError(e);
      return const <PhotoItem>[];
    }
  }

  Future<void> _primeSources() async {
    final quarkFuture = _safeLoadPhotos(_loadQuarkPhotos);
    if (kIsWeb) {
      _quarkPhotos = await quarkFuture;
      _mobilePhotos = const <PhotoItem>[];
      return;
    }

    final mobileFuture = _safeLoadPhotos(_loadMobilePhotos);
    final lists = await Future.wait([quarkFuture, mobileFuture]);
    _quarkPhotos = lists[0];
    _mobilePhotos = lists[1];
  }

  Future<List<PhotoItem>> _safeLoadPhotos(
    Future<List<PhotoItem>> Function() loader,
  ) async {
    try {
      return await loader();
    } catch (_) {
      debugPrint('[photos_page.dart] Error in catch block');
      return const <PhotoItem>[];
    }
  }

  Future<List<PhotoItem>> _photosForCategory(PhotoCategory category) async {
    if (kIsWeb) {
      return _quarkPhotos;
    }

    switch (category) {
      case PhotoCategory.quark:
        return _quarkPhotos;
      case PhotoCategory.mobile:
        return _mobilePhotos;
      case PhotoCategory.all:
        return [..._quarkPhotos, ..._mobilePhotos];
      case PhotoCategory.favorites:
        return [
          ..._quarkPhotos,
          ..._mobilePhotos,
        ].where((p) => _favoriteKeys.contains(p.selectionKey)).toList();
    }
  }

  Future<List<PhotoItem>> _loadMobilePhotos() async {
    // Only attempt mobile photo loading on Android or iOS devices.
    if (kIsWeb ||
        (defaultTargetPlatform != TargetPlatform.android &&
            defaultTargetPlatform != TargetPlatform.iOS)) {
      return [];
    }

    final permission = await PhotoManager.requestPermissionExtend();
    if (!permission.isAuth) return [];

    // Load from the device 'All' album.
    final List<AssetPathEntity> paths = await PhotoManager.getAssetPathList(
      onlyAll: true,
      type: RequestType.image,
    );
    if (paths.isEmpty) return [];

    final AssetPathEntity all = paths.first;
    // Lazy: fetch an initial safe page size; UI can be extended to load more on scroll.
    final List<AssetEntity> assets = await all.getAssetListPaged(
      page: 0,
      size: 200,
    );
    return assets.map((a) => PhotoItem.fromAsset(a)).toList(growable: false);
  }

  Future<void> _selectCategory(PhotoCategory cat) async {
    setState(() {
      _selectedCategory = cat;
      _categoriesExpanded = false;
      _photosFuture = _photosForCategory(cat);
    });
  }

  @override
  Future<void> refresh() async {
    _noHostSelected = AppSettings.instance.activeHost == null;
    final futures = <Future<void>>[_primeSources()];
    if (!_noHostSelected) {
      futures.add(
        FavoritesService.listFavoriteKeys()
            .then((keys) {
              if (mounted) {
                setState(() {
                  _favoriteKeys
                    ..clear()
                    ..addAll(keys);
                });
              }
            })
            .catchError((_) {}),
      );
    }
    await Future.wait(futures);
    setState(() {
      _photosFuture = _photosForCategory(_selectedCategory);
    });
    await _photosFuture;
  }

  bool get _hasMoreQuark => _quarkInitialLoadDone && _quarkOffset < _quarkTotal;

  int _minColumnsByScale() {
    return _minPreviewColumns;
  }

  int _maxColumnsByScale() {
    return _maxPreviewColumns;
  }

  int _maxColumnsByWidth(double availableWidth) {
    return (availableWidth / _minTileWidth).floor().clamp(1, 100);
  }

  int _effectiveCrossAxisCount(double availableWidth) {
    final maxByWidth = _maxColumnsByWidth(availableWidth);
    var minColumns = _minColumnsByScale();
    var maxColumns = _maxColumnsByScale();

    if (minColumns > maxByWidth) {
      minColumns = maxByWidth;
    }
    if (maxColumns > maxByWidth) {
      maxColumns = maxByWidth;
    }
    if (minColumns > maxColumns) {
      minColumns = maxColumns;
    }

    return _previewColumns.clamp(minColumns, maxColumns);
  }

  Widget _buildSidebar(
    BuildContext context,
    double availableWidth,
    int quarkCount,
    int mobileCount, {
    bool compact = false,
  }) {
    final theme = Theme.of(context);
    final maxByWidth = _maxColumnsByWidth(availableWidth);
    var minColumns = _minColumnsByScale();
    var maxColumns = _maxColumnsByScale();
    if (minColumns > maxByWidth) {
      minColumns = maxByWidth;
    }
    if (maxColumns > maxByWidth) {
      maxColumns = maxByWidth;
    }
    if (minColumns > maxColumns) {
      minColumns = maxColumns;
    }

    final selectedColumns = _previewColumns.clamp(minColumns, maxColumns);
    final divisions = maxColumns - minColumns;

    Widget categoryButton(PhotoCategory cat, String label, int count) {
      final selected = _selectedCategory == cat;
      return ListTile(
        dense: true,
        visualDensity: VisualDensity.compact,
        contentPadding: EdgeInsets.zero,
        onTap: () => _selectCategory(cat),
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
    final quarkDisplayCount = _quarkInitialLoadDone ? _quarkTotal : quarkCount;

    final selectedLabel = switch (_selectedCategory) {
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
      key: _albumSidebarKey,
      selectedAlbumId: null,
      shrinkWrap: compact,
      onAlbumSelected: (album) {
        if (album == null) return;
        Navigator.of(
          context,
        ).push(MaterialPageRoute(builder: (_) => AlbumPage(album: album)));
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
                      ? () {
                          setState(() {
                            _previewColumns = (selectedColumns - 1).clamp(
                              minColumns,
                              maxColumns,
                            );
                          });
                        }
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
                    onChanged: (value) {
                      setState(() {
                        _previewColumns = value.round();
                      });
                    },
                  ),
                ),
                IconButton(
                  onPressed: selectedColumns < maxColumns
                      ? () {
                          setState(() {
                            _previewColumns = (selectedColumns + 1).clamp(
                              minColumns,
                              maxColumns,
                            );
                          });
                        }
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
                  '$selectedLabel: ${switch (_selectedCategory) {
                    PhotoCategory.all => quarkDisplayCount + mobileCount,
                    PhotoCategory.quark => quarkDisplayCount,
                    PhotoCategory.mobile => mobileCount,
                    PhotoCategory.favorites => _favoriteKeys.length,
                  }}',
                ),
                trailing: Icon(
                  _categoriesExpanded
                      ? QuarkIcons.expand_less
                      : QuarkIcons.expand_more,
                ),
                onTap: () {
                  setState(() {
                    _categoriesExpanded = !_categoriesExpanded;
                  });
                },
              ),
              if (_categoriesExpanded) ...[
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
                  _favoriteKeys.length,
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

  // Builds a single photo tile. Extracted so both the desktop GridView and
  // the mobile SliverGrid can share the same tile logic.
  Future<void> _toggleFavorite(PhotoItem item) async {
    if (!item.isFiles) return;
    final c = item.quark!;
    try {
      final nowFav = await FavoritesService.toggle(
        relPath: c.apiPath,
        serial: c.deviceSerial.isNotEmpty ? c.deviceSerial : null,
      );
      if (!mounted) return;
      setState(() {
        if (nowFav) {
          _favoriteKeys.add(item.selectionKey);
        } else {
          _favoriteKeys.remove(item.selectionKey);
        }
        // Refresh the grid immediately if the favorites tab is active.
        if (_selectedCategory == PhotoCategory.favorites) {
          _photosFuture = _photosForCategory(PhotoCategory.favorites);
        }
      });
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'update the favorite'))),
      );
    }
  }

  Widget _buildStarOverlay(BuildContext context, PhotoItem item) {
    final isFav = _favoriteKeys.contains(item.selectionKey);
    if (!isFav) return const SizedBox.shrink();
    return Positioned(
      bottom: 4,
      right: 4,
      child: Icon(
        QuarkIcons.star_rounded,
        size: 16,
        color: Colors.white,
        shadows: const [Shadow(blurRadius: 4, color: Colors.black54)],
      ),
    );
  }

  Widget _buildPhotoTile(
    BuildContext context,
    List<PhotoItem> photos,
    int idx,
    int crossAxisCount,
  ) {
    final p = photos[idx];
    final isSelected = _selectedKeys.contains(p.selectionKey);
    final colorScheme = Theme.of(context).colorScheme;

    // In selection mode wrap everything with selection overlay
    Widget wrapWithSelection(Widget child) {
      return GestureDetector(
        onTap: () => _toggleSelection(p),
        onLongPress: () {
          if (!_selectionMode) _enterSelectionMode();
          _toggleSelection(p);
        },
        child: Stack(
          fit: StackFit.expand,
          children: [
            child,
            // Dim overlay for unselected
            if (_selectionMode && !isSelected)
              Container(color: Colors.black.withValues(alpha: 0.3)),
            // Teal border for selected
            if (isSelected)
              DecoratedBox(
                decoration: BoxDecoration(
                  border: Border.all(color: colorScheme.primary, width: 3),
                ),
              ),
            // Checkbox in top-left
            if (_selectionMode)
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

    if (!_selectionMode) {
      // Normal mode — use original long-press to enter selection
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
      if (_selectionMode) return wrapWithSelection(thumbnail);
      return MouseRegion(
        cursor: SystemMouseCursors.click,
        child: GestureDetector(
          onLongPress: () {
            _enterSelectionMode();
            _toggleSelection(p);
          },
          onDoubleTap: () => _toggleFavorite(p),
          onTap: () async {
            if (_isOpeningPhoto) return;
            _isOpeningPhoto = true;
            try {
              final navigator = Navigator.of(context);
              final bytes = await FilesService.downloadFileBytes(
                c.apiPath,
                serial: c.deviceSerial,
              );
              if (bytes == null) return;
              if (!mounted) return;
              final changed = await navigator.push<bool>(
                MaterialPageRoute(
                  builder: (_) => ImageViewerPage(
                    bytes: bytes,
                    name: c.name,
                    initialIndex: idx,
                    imageCount: photos.length,
                    relPath: c.apiPath,
                    serial: c.deviceSerial,
                    getImageCount: () async =>
                        (await _photosForCategory(_selectedCategory)).length,
                    onLoadImage: _loadPhotoAt,
                    onPrefetchImage: _prefetchPhotoAt,
                  ),
                ),
              );
              if (changed == true) {
                await manualRefresh();
                _albumSidebarKey.currentState?.reload();
              }
            } finally {
              if (mounted) setState(() => _isOpeningPhoto = false);
            }
          },
          child: Stack(
            fit: StackFit.expand,
            children: [thumbnail, _buildStarOverlay(context, p)],
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
    if (_selectionMode) return wrapWithSelection(assetThumb);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onLongPress: () {
          _enterSelectionMode();
          _toggleSelection(p);
        },
        onTap: () async {
          if (_isOpeningPhoto) return;
          _isOpeningPhoto = true;
          try {
            final navigator = Navigator.of(context);
            final bytes = await a.originBytes;
            if (bytes == null) return;
            if (!mounted) return;
            final changed = await navigator.push<bool>(
              MaterialPageRoute(
                builder: (_) => ImageViewerPage(
                  bytes: bytes,
                  name: a.id,
                  initialIndex: idx,
                  imageCount: photos.length,
                  getImageCount: () async =>
                      (await _photosForCategory(_selectedCategory)).length,
                  onLoadImage: _loadPhotoAt,
                  onPrefetchImage: _prefetchPhotoAt,
                ),
              ),
            );
            if (changed == true) await manualRefresh();
          } finally {
            if (mounted) setState(() => _isOpeningPhoto = false);
          }
        },
        child: assetThumb,
      ),
    );
  }

  /// Loads the photo at [newIdx] for the open viewer.
  ///
  /// A 404 is the only answer that means the photo is really gone, and it is
  /// the only one worth a full grid refetch — the grid behind the viewer is
  /// stale. Every other failure propagates so the viewer can offer a retry,
  /// and leaves the grid alone rather than paying for a refetch on one flaky
  /// request (#1708).
  Future<(Uint8List?, String, String?, String?)> _loadPhotoAt(
    int newIdx,
  ) async {
    final live = await _photosForCategory(_selectedCategory);
    if (newIdx >= live.length) return (null, '', null, null);
    final item = live[newIdx];
    if (!item.isFiles) {
      final na = item.asset!;
      return (await na.originBytes, na.id, null, null);
    }
    final nc = item.quark!;
    try {
      final bytes = await _quarkPhotoBytes(nc);
      return (bytes, nc.name, nc.apiPath, nc.deviceSerial);
    } on ApiException catch (e) {
      if (e.statusCode == 404) await manualRefresh();
      rethrow;
    }
  }

  /// Downloads the photo at [idx] into the cache so the viewer's next step is
  /// instant (#1710).
  ///
  /// Unlike [_loadPhotoAt] this reacts to nothing: the user did not ask for
  /// this photo, so a 404 on it is not a reason to refetch the grid and no
  /// failure here is worth telling them about. The viewer swallows what this
  /// throws.
  Future<void> _prefetchPhotoAt(int idx) async {
    final live = await _photosForCategory(_selectedCategory);
    if (idx < 0 || idx >= live.length) return;
    final item = live[idx];
    // Local device assets come off disk through photo_manager's own cache;
    // only the network round trip is worth pre-paying.
    if (!item.isFiles) return;
    await _quarkPhotoBytes(item.quark!);
  }

  Future<Uint8List?> _quarkPhotoBytes(FileNode photo) =>
      PhotoBytesCache.instance.fetch(
        PhotoBytesCache.key(photo.apiPath, photo.deviceSerial),
        () => FilesService.downloadFileBytes(
          photo.apiPath,
          serial: photo.deviceSerial,
        ),
      );

  Widget _buildPhotoGrid(List<PhotoItem> photos, int crossAxisCount) {
    // When viewing Quark-stored photos (or all), we may have more pages to load.
    // Show an extra item as a loading indicator if there are more.
    final showLoadingIndicator =
        _hasMoreQuark &&
        (_selectedCategory == PhotoCategory.quark ||
            _selectedCategory == PhotoCategory.all);
    final itemCount = photos.length + (showLoadingIndicator ? 1 : 0);

    return RefreshIndicator(
      onRefresh: manualRefresh,
      child: photos.isEmpty && !_isLoadingMoreQuark
          // "No photos yet" would be a lie when the library is merely out of
          // reach, so an unreachable Quark wins over every empty state (#1637).
          ? (_quarkUnreachable
                ? QuarkDisconnectedView(
                    hostAddress: AppSettings.instance.activeHost,
                    onRetry: manualRefresh,
                    onManageHosts: () => context.go(AppRoutes.settings),
                  )
                : _selectedCategory == PhotoCategory.favorites
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
              controller: _scrollController,
              padding: EdgeInsets.fromLTRB(2, 2, 2, _selectionMode ? 84 : 2),
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
                return _buildPhotoTile(context, photos, idx, crossAxisCount);
              },
            ),
    );
  }

  void _enterSelectionMode({PhotoAlbum? addingToAlbum}) {
    setState(() {
      _selectionMode = true;
      _addingToAlbum = addingToAlbum;
      _selectedKeys.clear();
    });
  }

  void _exitSelectionMode() {
    setState(() {
      _selectionMode = false;
      _addingToAlbum = null;
      _selectedKeys.clear();
    });
  }

  void _toggleSelection(PhotoItem item) {
    setState(() {
      final key = item.selectionKey;
      if (_selectedKeys.contains(key)) {
        _selectedKeys.remove(key);
      } else {
        _selectedKeys.add(key);
      }
    });
  }

  Future<void> _handleAddToAlbum(List<PhotoItem> allPhotos) async {
    final album = await AlbumPickerSheet.show(
      context,
      selectedCount: _selectedKeys.length,
    );
    if (album == null || !mounted) return;
    await _addSelectedToAlbum(album, allPhotos);
  }

  Future<void> _confirmAddToAlbum() async {
    // Flow 2: user tapped Done while in adding-to-album mode.
    // Pass the full set of loaded photos so that any selected device assets
    // are counted as skipped (and the user gets feedback) rather than silently
    // ignored. _addSelectedToAlbum already rejects photos not stored on Quark.
    await _addSelectedToAlbum(_addingToAlbum!, [
      ..._quarkPhotos,
      ..._mobilePhotos,
    ]);
  }

  Future<void> _addSelectedToAlbum(
    PhotoAlbum album,
    List<PhotoItem> allPhotos,
  ) async {
    final selected = allPhotos
        .where((p) => _selectedKeys.contains(p.selectionKey))
        .toList();

    int added = 0;
    int skipped = 0;
    int failed = 0;
    for (final item in selected) {
      if (!item.isFiles) {
        skipped++;
        continue;
      }
      final c = item.quark!;
      try {
        await AlbumService.addPhotoToAlbum(
          album.id,
          deviceSerial: c.deviceSerial,
          relPath: c.apiPath,
        );
        added++;
      } catch (_) {
        failed++;
      }
    }

    if (!mounted) return;

    final wasAddingMode = _addingToAlbum != null;
    _exitSelectionMode();

    String message;
    if (added == 0 && skipped > 0) {
      message = 'No photos added — device photos cannot be added to albums yet';
    } else if (added > 0) {
      message =
          '$added ${added == 1 ? 'photo' : 'photos'} added to "${album.name}"';
      if (failed > 0) message += ' ($failed failed)';
    } else {
      message = Errors.couldNot('add photos to "${album.name}"');
    }

    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));

    if (wasAddingMode && added > 0) {
      Navigator.of(context).pop();
    } else if (wasAddingMode && added == 0) {
      // Stay in adding mode so user can try again or pick different photos
    }
  }

  Future<void> _handleUploadPhotos() async {
    if (_isUploading) return;

    try {
      final result = await FilePicker.pickFiles(
        type: FileType.image,
        allowMultiple: true,
        withData: true,
      );
      if (result == null || result.files.isEmpty || !mounted) return;

      final multipartFiles = <http.MultipartFile>[];
      for (final f in result.files) {
        if (kIsWeb) {
          if (f.bytes == null) continue;
          multipartFiles.add(
            http.MultipartFile.fromBytes('files', f.bytes!, filename: f.name),
          );
        } else if (f.path != null && f.path!.isNotEmpty) {
          multipartFiles.add(
            await http.MultipartFile.fromPath(
              'files',
              f.path!,
              filename: f.name,
            ),
          );
        } else if (f.bytes != null) {
          multipartFiles.add(
            http.MultipartFile.fromBytes('files', f.bytes!, filename: f.name),
          );
        }
      }
      if (multipartFiles.isEmpty || !mounted) return;

      String? targetSerial;
      try {
        final devices = (await StorageService.listDevices())
            .where((d) => d.isEnabled)
            .toList();
        if (devices.length > 1) {
          if (!mounted) return;
          final picked = await showDeviceUploadPicker(context, devices);
          if (picked == null) return;
          targetSerial = picked.serial.isNotEmpty ? picked.serial : null;
        } else if (devices.length == 1) {
          targetSerial = devices.first.serial.isNotEmpty
              ? devices.first.serial
              : null;
        }
      } catch (_) {}

      setState(() => _isUploading = true);

      try {
        await FilesService.uploadFilesFromFormData(
          '',
          multipartFiles,
          serial: targetSerial,
        );
        if (!mounted) return;
        final label = multipartFiles.length == 1
            ? multipartFiles.first.filename ?? 'photo'
            : '${multipartFiles.length} photos';
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Uploaded $label')));
        await manualRefresh();
      } catch (e) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(Errors.message(e, 'upload your photos'))),
        );
      } finally {
        if (mounted) setState(() => _isUploading = false);
      }
    } on MissingPluginException {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('File picker not available. Fully restart the app.'),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return CallbackShortcuts(
      bindings: {
        const SingleActivator(LogicalKeyboardKey.escape): () {
          if (_selectionMode) {
            final wasAdding = _addingToAlbum != null;
            _exitSelectionMode();
            if (wasAdding) Navigator.of(context).pop();
          }
        },
      },
      child: Focus(
        autofocus: true,
        child: Scaffold(
          appBar: _selectionMode
              ? AppBar(
                  backgroundColor: Theme.of(context).colorScheme.secondary,
                  automaticallyImplyLeading: false,
                  title: _addingToAlbum != null
                      ? Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              'Adding to ${_addingToAlbum!.name}',
                              style: TextStyle(
                                fontSize: 14,
                                color: Theme.of(context).colorScheme.primary,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            if (_selectedKeys.isNotEmpty)
                              Text(
                                '${_selectedKeys.length} selected',
                                style: TextStyle(
                                  fontSize: 11,
                                  color: Theme.of(context).colorScheme.onSurface
                                      .withValues(alpha: 0.5),
                                ),
                              ),
                          ],
                        )
                      : Text('${_selectedKeys.length} selected'),
                  actions: [
                    if (_addingToAlbum != null)
                      TextButton(
                        onPressed: _selectedKeys.isNotEmpty
                            ? _confirmAddToAlbum
                            : null,
                        child: Text('Done (${_selectedKeys.length})'),
                      ),
                    TextButton(
                      onPressed: () {
                        final wasAdding = _addingToAlbum != null;
                        _exitSelectionMode();
                        if (wasAdding) Navigator.of(context).pop();
                      },
                      child: const Text('Cancel'),
                    ),
                    const AppThemeToggle(),
                  ],
                )
              : QuarkAppBar(
                  label: 'Photos',
                  icon: QuarkIcons.photo_library_outlined,
                  actions: [
                    IconButton(
                      icon: _isUploading
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.add),
                      tooltip: 'Upload photos',
                      onPressed: _isUploading ? null : _handleUploadPhotos,
                    ),
                    TextButton(
                      onPressed: _enterSelectionMode,
                      child: const Text('Select'),
                    ),
                    RefreshIconButton(
                      isRefreshing: isRefreshing,
                      onPressed: manualRefresh,
                      tooltip: 'Reload photos',
                    ),
                    const AppThemeToggle(),
                  ],
                ),
          drawer: QuarkDrawer(
            activeSection: QuarkDrawerSection.photos,
            onTapFiles: () {
              context.go(AppRoutes.files);
            },
            onTapPhotos: () {
              Navigator.of(context).pop();
            },
            onTapDocs: () {
              context.go(AppRoutes.docs);
            },
            onTapSheets: () {
              context.go(AppRoutes.sheets);
            },
            onTapDevices: () {
              context.go(AppRoutes.devices);
            },
            onTapHealth: () {
              context.go(AppRoutes.health);
            },
            onTapVault: () {
              context.go(AppRoutes.vault);
            },
            onTapSettings: () {
              context.go(AppRoutes.settings);
            },
          ),
          body: FutureBuilder<List<PhotoItem>>(
            future: _photosFuture,
            builder: (context, snapshot) {
              final photos = snapshot.data ?? const <PhotoItem>[];

              return LayoutBuilder(
                builder: (context, constraints) {
                  final compact = constraints.maxWidth < 900;
                  // Recorded for _measureAndJumpNav, which runs after this
                  // build and only has something to measure in compact mode.
                  _compactLayout = compact;
                  if (compact && !_navScrollInitialized) {
                    _scheduleNavMeasure();
                  }
                  final contentWidth = compact
                      ? constraints.maxWidth
                      : (constraints.maxWidth - 281)
                            .clamp(1.0, double.infinity)
                            .toDouble();
                  final crossAxisCount = _effectiveCrossAxisCount(contentWidth);

                  final sidebar = _buildSidebar(
                    context,
                    contentWidth,
                    _quarkPhotos.length,
                    _mobilePhotos.length,
                    compact: compact,
                  );

                  // Desktop: sidebar + grid side-by-side (unchanged behavior)
                  if (!compact) {
                    Widget buildDesktop(Widget content) => Row(
                      children: [
                        sidebar,
                        const VerticalDivider(width: 1),
                        Expanded(child: content),
                      ],
                    );
                    if (snapshot.connectionState == ConnectionState.waiting &&
                        photos.isEmpty) {
                      return buildDesktop(
                        const Center(child: CircularProgressIndicator()),
                      );
                    }
                    if (snapshot.hasError) {
                      return buildDesktop(
                        Center(
                          child: Text(
                            Errors.message(snapshot.error!, 'load your photos'),
                          ),
                        ),
                      );
                    }
                    return buildDesktop(
                      _buildPhotoGrid(photos, crossAxisCount),
                    );
                  }

                  // Mobile: nav panel lives *above* the photo grid in the scroll
                  // stack. On mount we jump to navHeight so only the grid is
                  // visible. Scroll up to reveal the nav.
                  if (snapshot.connectionState == ConnectionState.waiting &&
                      photos.isEmpty) {
                    return const Center(child: CircularProgressIndicator());
                  }
                  if (snapshot.hasError) {
                    return Center(
                      child: Text(
                        Errors.message(snapshot.error!, 'load your photos'),
                      ),
                    );
                  }

                  final showLoadingIndicator =
                      _hasMoreQuark &&
                      (_selectedCategory == PhotoCategory.quark ||
                          _selectedCategory == PhotoCategory.all);
                  final itemCount =
                      photos.length + (showLoadingIndicator ? 1 : 0);

                  return Stack(
                    children: [
                      RefreshIndicator(
                        onRefresh: manualRefresh,
                        child: CustomScrollView(
                          controller: _scrollController,
                          physics: const AlwaysScrollableScrollPhysics(),
                          slivers: [
                            // Nav panel — hidden above viewport on load
                            SliverToBoxAdapter(
                              child: Container(
                                key: _navPanelKey,
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    sidebar,
                                    Divider(
                                      height: 1,
                                      color: Theme.of(
                                        context,
                                      ).colorScheme.outline,
                                    ),
                                  ],
                                ),
                              ),
                            ),
                            // Photo grid
                            photos.isEmpty
                                ? const SliverFillRemaining(
                                    child: EmptyStateWidget(
                                      icon: QuarkIcons.photo_library_outlined,
                                      headline: 'No photos yet',
                                      subtext:
                                          'Photos you upload to Quark will appear here.',
                                    ),
                                  )
                                : SliverGrid(
                                    delegate: SliverChildBuilderDelegate((
                                      context,
                                      idx,
                                    ) {
                                      if (idx >= photos.length) {
                                        return const Center(
                                          child: Padding(
                                            padding: EdgeInsets.all(16),
                                            child: CircularProgressIndicator(
                                              strokeWidth: 2,
                                            ),
                                          ),
                                        );
                                      }
                                      return _buildPhotoTile(
                                        context,
                                        photos,
                                        idx,
                                        crossAxisCount,
                                      );
                                    }, childCount: itemCount),
                                    gridDelegate:
                                        SliverGridDelegateWithFixedCrossAxisCount(
                                          crossAxisCount: crossAxisCount,
                                          crossAxisSpacing: 2,
                                          mainAxisSpacing: 2,
                                        ),
                                  ),
                          ],
                        ),
                      ),
                      // Selection action bar — shown at bottom when in selection mode
                      if (_selectionMode && _addingToAlbum == null)
                        Positioned(
                          bottom: 0,
                          left: 0,
                          right: 0,
                          child: PhotoSelectionBar(
                            selectedCount: _selectedKeys.length,
                            onAddToAlbum: () => _handleAddToAlbum(photos),
                            onCancel: _exitSelectionMode,
                          ),
                        ),
                      // Scroll-up hint — subtle chevron at top, fades when user scrolls
                      if (_showScrollHint)
                        Positioned(
                          top: 0,
                          left: 0,
                          right: 0,
                          child: IgnorePointer(
                            child: Container(
                              height: 32,
                              decoration: BoxDecoration(
                                gradient: LinearGradient(
                                  begin: Alignment.topCenter,
                                  end: Alignment.bottomCenter,
                                  colors: [
                                    Theme.of(context).colorScheme.surface
                                        .withValues(alpha: 0.7),
                                    Colors.transparent,
                                  ],
                                ),
                              ),
                              child: Center(
                                child: Icon(
                                  QuarkIcons.keyboard_arrow_up_rounded,
                                  size: 20,
                                  color: Theme.of(context).colorScheme.onSurface
                                      .withValues(alpha: 0.4),
                                ),
                              ),
                            ),
                          ),
                        ),
                    ],
                  );
                },
              );
            },
          ),
        ),
      ),
    );
  }
}
