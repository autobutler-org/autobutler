import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/utils/auto_refresh_mixin.dart';
import 'package:autobutler/widgets/core/empty_state_widget.dart';
import 'package:autobutler/widgets/refresh_icon_button.dart';
import 'package:autobutler/pages/image_viewer_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:autobutler/widgets/layout/autobutler_app_bar.dart';
import 'package:autobutler/models/photo_album.dart';
import 'package:autobutler/pages/album_page.dart';
import 'package:autobutler/services/album_service.dart';
import 'package:autobutler/widgets/photos/album_sidebar.dart';
import 'package:autobutler/widgets/photos/photo_selection_bar.dart';
import 'package:flutter/foundation.dart';
import 'package:autobutler/router.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:photo_manager/photo_manager.dart';

class PhotosPage extends StatefulWidget {
  const PhotosPage({this.addingToAlbum, super.key});

  /// When set, the page opens in "adding to album" mode — selection mode is
  /// immediately active and the header shows the album name.
  final PhotoAlbum? addingToAlbum;

  @override
  State<PhotosPage> createState() => _PhotosPageState();
}

enum PhotoCategory { cirrus, mobile, all }

class PhotoItem {
  final CirrusFileNode? cirrus;
  final AssetEntity? asset;
  final bool isCirrus;

  PhotoItem._({this.cirrus, this.asset}) : isCirrus = cirrus != null;

  factory PhotoItem.fromCirrus(CirrusFileNode c) => PhotoItem._(cirrus: c);
  factory PhotoItem.fromAsset(AssetEntity a) => PhotoItem._(asset: a);

  /// A stable key for this item suitable for use in a selection set.
  String get selectionKey {
    if (isCirrus) {
      return '${cirrus!.deviceSerial}:${cirrus!.dirPath}';
    }
    return 'asset:${asset!.id}';
  }
}

class _PhotosPageState extends State<PhotosPage>
    with WidgetsBindingObserver, AutoRefreshMixin {
  static const int _defaultCrossAxisCount = 4;
  static const int _minPreviewColumns = 1;
  static const int _maxPreviewColumns = 8;
  static const double _minTileWidth = 80;
  static const int _pageSize = 50;

  Future<List<PhotoItem>> _photosFuture = Future.value(const <PhotoItem>[]);

  // Cirrus pagination state
  List<PhotoItem> _cirrusPhotos = <PhotoItem>[];
  int _cirrusTotal = 0;
  int _cirrusOffset = 0;
  bool _isLoadingMoreCirrus = false;
  bool _cirrusInitialLoadDone = false;

  List<PhotoItem> _mobilePhotos = const <PhotoItem>[];

  bool _noHostSelected = false;
  bool _categoriesExpanded = false;
  int _previewColumns = _defaultCrossAxisCount;
  PhotoCategory _selectedCategory = PhotoCategory.cirrus;

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
  bool _navScrollInitialized = false;
  bool _showScrollHint = true;

  // Selection mode
  bool _selectionMode = false;
  final Set<String> _selectedKeys = {};
  // When non-null, we're in "adding to album" mode (Flow 2)
  PhotoAlbum? _addingToAlbum;

  ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    WidgetsBinding.instance.addPostFrameCallback((_) => _measureAndJumpNav());
    // If launched in adding-to-album mode, enter selection mode immediately
    if (widget.addingToAlbum != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _enterSelectionMode(addingToAlbum: widget.addingToAlbum);
      });
    }
  }

  void _measureAndJumpNav() {
    if (_navScrollInitialized) return;
    final ctx = _navPanelKey.currentContext;
    if (ctx == null) {
      // Nav not yet in tree — retry next frame
      WidgetsBinding.instance.addPostFrameCallback((_) => _measureAndJumpNav());
      return;
    }
    final box = ctx.findRenderObject() as RenderBox?;
    if (box == null || !box.hasSize) {
      WidgetsBinding.instance.addPostFrameCallback((_) => _measureAndJumpNav());
      return;
    }
    final navHeight = box.size.height;
    if (navHeight <= 0) {
      WidgetsBinding.instance.addPostFrameCallback((_) => _measureAndJumpNav());
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
      _loadMoreCirrusPhotos();
    }
  }

  /// Load the next page of cirrus photos via the paginated endpoint.
  Future<void> _loadMoreCirrusPhotos() async {
    if (_isLoadingMoreCirrus) return;
    if (_cirrusOffset >= _cirrusTotal && _cirrusInitialLoadDone) return;

    setState(() {
      _isLoadingMoreCirrus = true;
    });

    try {
      final response = await CirrusService.getPhotos(
        offset: _cirrusOffset,
        limit: _pageSize,
      );

      final newPhotos = response.photos
          .map(
            (p) => PhotoItem.fromCirrus(
              CirrusFileNode(
                name: p.fileName,
                size: p.size,
                isDir: false,
                deviceName: '',
                devicePath: '',
                deviceSerial: p.serial,
                dirPath: p.relPath,
              ),
            ),
          )
          .toList(growable: false);

      setState(() {
        _cirrusPhotos = [..._cirrusPhotos, ...newPhotos];
        _cirrusTotal = response.total;
        _cirrusOffset += newPhotos.length;
        _cirrusInitialLoadDone = true;
        _isLoadingMoreCirrus = false;
        // Rebuild the future so FutureBuilder picks up the new list
        _photosFuture = _photosForCategory(_selectedCategory);
      });
    } catch (_) {
      debugPrint('[photos_page.dart] Error loading more cirrus photos');
      setState(() {
        _isLoadingMoreCirrus = false;
        _cirrusInitialLoadDone = true;
      });
    }
  }

  /// Initial load of cirrus photos (first page).
  Future<List<PhotoItem>> _loadCirrusPhotos() async {
    if (_noHostSelected) return const <PhotoItem>[];

    _cirrusPhotos = <PhotoItem>[];
    _cirrusOffset = 0;
    _cirrusTotal = 0;
    _cirrusInitialLoadDone = false;

    try {
      final response = await CirrusService.getPhotos(
        offset: 0,
        limit: _pageSize,
      );

      final items = response.photos
          .map(
            (p) => PhotoItem.fromCirrus(
              CirrusFileNode(
                name: p.fileName,
                size: p.size,
                isDir: false,
                deviceName: '',
                devicePath: '',
                deviceSerial: p.serial,
                dirPath: p.relPath,
              ),
            ),
          )
          .toList(growable: false);

      _cirrusPhotos = items;
      _cirrusTotal = response.total;
      _cirrusOffset = items.length;
      _cirrusInitialLoadDone = true;
      return items;
    } catch (_) {
      debugPrint('[photos_page.dart] Error loading initial cirrus photos');
      _cirrusInitialLoadDone = true;
      return const <PhotoItem>[];
    }
  }

  Future<void> _primeSources() async {
    final cirrusFuture = _safeLoadPhotos(_loadCirrusPhotos);
    if (kIsWeb) {
      _cirrusPhotos = await cirrusFuture;
      _mobilePhotos = const <PhotoItem>[];
      return;
    }

    final mobileFuture = _safeLoadPhotos(_loadMobilePhotos);
    final lists = await Future.wait([cirrusFuture, mobileFuture]);
    _cirrusPhotos = lists[0];
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
      return _cirrusPhotos;
    }

    switch (category) {
      case PhotoCategory.cirrus:
        return _cirrusPhotos;
      case PhotoCategory.mobile:
        return _mobilePhotos;
      case PhotoCategory.all:
        return [..._cirrusPhotos, ..._mobilePhotos];
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
    await _primeSources();
    setState(() {
      _photosFuture = _photosForCategory(_selectedCategory);
    });
    await _photosFuture;
  }

  bool get _hasMoreCirrus =>
      _cirrusInitialLoadDone && _cirrusOffset < _cirrusTotal;

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
    int cirrusCount,
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
        leading: Icon(
          cat == PhotoCategory.cirrus
              ? Icons.cloud
              : (cat == PhotoCategory.mobile
                    ? Icons.smartphone
                    : Icons.photo_library),
          color: selected ? theme.colorScheme.primary : null,
        ),
        title: Text('$label: $count', style: theme.textTheme.titleMedium),
        trailing: selected ? const Icon(Icons.check, size: 16) : null,
      );
    }

    // For cirrus, show total from server (includes un-fetched pages)
    final cirrusDisplayCount = _cirrusInitialLoadDone
        ? _cirrusTotal
        : cirrusCount;

    final selectedLabel = switch (_selectedCategory) {
      PhotoCategory.all => 'All',
      PhotoCategory.cirrus => 'Cirrus',
      PhotoCategory.mobile => 'Mobile',
    };

    return Container(
      width: compact ? double.infinity : 280,
      padding: const EdgeInsets.all(16),
      color: theme.colorScheme.surfaceContainerLowest,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
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
                icon: const Icon(Icons.crop_square_outlined),
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
                icon: const Icon(Icons.grid_view_outlined),
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
                  PhotoCategory.all => cirrusDisplayCount + mobileCount,
                  PhotoCategory.cirrus => cirrusDisplayCount,
                  PhotoCategory.mobile => mobileCount,
                }}',
              ),
              trailing: Icon(
                _categoriesExpanded ? Icons.expand_less : Icons.expand_more,
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
                cirrusDisplayCount + mobileCount,
              ),
              categoryButton(
                PhotoCategory.cirrus,
                'Cirrus',
                cirrusDisplayCount,
              ),
              categoryButton(PhotoCategory.mobile, 'Mobile', mobileCount),
            ],
          ],
          const SizedBox(height: 16),
          AlbumSidebar(
            selectedAlbumId: null,
            onAlbumSelected: (album) {
              if (album == null) return;
              Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => AlbumPage(album: album)),
              );
            },
          ),
        ],
      ),
    );
  }

  // Builds a single photo tile. Extracted so both the desktop GridView and
  // the mobile SliverGrid can share the same tile logic.
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
                      ? const Icon(Icons.check, size: 14, color: Colors.white)
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

    if (p.isCirrus) {
      final c = p.cirrus!;
      final url = CirrusService.constructThumbnailUrl(
        c.apiPath,
        serial: c.deviceSerial,
      );
      final thumbnail = Image.network(
        url.toString(),
        fit: BoxFit.cover,
        loadingBuilder: (context, child, progress) {
          if (progress == null) return child;
          return Container(color: Colors.grey[300]);
        },
        errorBuilder: (context, error, stack) =>
            Container(color: Colors.grey[300]),
      );
      if (_selectionMode) return wrapWithSelection(thumbnail);
      return MouseRegion(
        cursor: SystemMouseCursors.click,
        child: GestureDetector(
          onLongPress: () {
            _enterSelectionMode();
            _toggleSelection(p);
          },
          onTap: () async {
            final navigator = Navigator.of(context);
            final bytes = await CirrusService.downloadFileBytes(
              c.apiPath,
              serial: c.deviceSerial,
            );
            if (bytes == null) return;
            if (!mounted) return;
            await navigator.push(
              MaterialPageRoute(
                builder: (_) => ImageViewerPage(
                  bytes: bytes,
                  name: c.name,
                  initialIndex: idx,
                  imageCount: photos.length,
                  getImageCount: () async =>
                      (await _photosForCategory(_selectedCategory)).length,
                  onLoadImage: (newIdx) async {
                    final live = await _photosForCategory(_selectedCategory);
                    if (newIdx >= live.length) return (null, '');
                    final item = live[newIdx];
                    if (item.isCirrus) {
                      final nc = item.cirrus!;
                      var b = await CirrusService.downloadFileBytes(
                        nc.apiPath,
                        serial: nc.deviceSerial,
                      );
                      if (b == null) await manualRefresh();
                      return (b, nc.name);
                    } else {
                      final na = item.asset!;
                      final b = await na.originBytes;
                      if (b == null) await manualRefresh();
                      return (b, na.id);
                    }
                  },
                ),
              ),
            );
          },
          child: thumbnail,
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
          final navigator = Navigator.of(context);
          final bytes = await a.originBytes;
          if (bytes == null) return;
          if (!mounted) return;
          await navigator.push(
            MaterialPageRoute(
              builder: (_) => ImageViewerPage(
                bytes: bytes,
                name: a.id,
                initialIndex: idx,
                imageCount: photos.length,
                getImageCount: () async =>
                    (await _photosForCategory(_selectedCategory)).length,
                onLoadImage: (newIdx) async {
                  final live = await _photosForCategory(_selectedCategory);
                  if (newIdx >= live.length) return (null, '');
                  final item = live[newIdx];
                  if (item.isCirrus) {
                    final nc = item.cirrus!;
                    var b = await CirrusService.downloadFileBytes(
                      nc.apiPath,
                      serial: nc.deviceSerial,
                    );
                    if (b == null) await manualRefresh();
                    return (b, nc.name);
                  } else {
                    final na = item.asset!;
                    final b = await na.originBytes;
                    if (b == null) await manualRefresh();
                    return (b, na.id);
                  }
                },
              ),
            ),
          );
        },
        child: assetThumb,
      ),
    );
  }

  Widget _buildPhotoGrid(List<PhotoItem> photos, int crossAxisCount) {
    // When viewing cirrus (or all), we may have more pages to load.
    // Show an extra item as a loading indicator if there are more.
    final showLoadingIndicator =
        _hasMoreCirrus &&
        (_selectedCategory == PhotoCategory.cirrus ||
            _selectedCategory == PhotoCategory.all);
    final itemCount = photos.length + (showLoadingIndicator ? 1 : 0);

    return RefreshIndicator(
      onRefresh: manualRefresh,
      child: photos.isEmpty && !_isLoadingMoreCirrus
          ? const EmptyStateWidget(
              icon: Icons.photo_library_outlined,
              headline: 'No photos yet',
              subtext: 'Photos you upload to AutoButler will appear here.',
            )
          : GridView.builder(
              controller: _scrollController,
              padding: const EdgeInsets.all(2),
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
    // Flow 2: user tapped Done while in adding-to-album mode
    // We need the current photo list — pull from state
    final photos = _selectedCategory == PhotoCategory.mobile
        ? _mobilePhotos
        : _cirrusPhotos;
    await _addSelectedToAlbum(_addingToAlbum!, photos);
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
      if (!item.isCirrus) {
        skipped++;
        continue;
      }
      final c = item.cirrus!;
      try {
        await AlbumService.addPhotoToAlbum(
          album.id,
          deviceSerial: c.deviceSerial,
          relPath: c.dirPath,
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
      message = 'Failed to add photos to "${album.name}"';
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
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
                              color: Theme.of(
                                context,
                              ).colorScheme.onSurface.withValues(alpha: 0.5),
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
                  onPressed: _exitSelectionMode,
                  child: const Text('Cancel'),
                ),
              ],
            )
          : AutobutlerAppBar(
              label: 'Photos',
              icon: Icons.photo_library_outlined,
              actions: [
                TextButton(
                  onPressed: _enterSelectionMode,
                  child: const Text('Select'),
                ),
                RefreshIconButton(
                  isRefreshing: isRefreshing,
                  onPressed: manualRefresh,
                  tooltip: 'Reload photos',
                ),
              ],
            ),
      drawer: AutobutlerDrawer(
        activeSection: AutobutlerDrawerSection.photos,
        onTapCirrus: () {
          context.go(AppRoutes.cirrus);
        },
        onTapPhotos: () {
          Navigator.of(context).pop();
        },
        onTapDevices: () {
          context.go(AppRoutes.devices);
        },
        onTapHealth: () {
          context.go(AppRoutes.health);
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
              final contentWidth = compact
                  ? constraints.maxWidth
                  : (constraints.maxWidth - 281)
                        .clamp(1.0, double.infinity)
                        .toDouble();
              final crossAxisCount = _effectiveCrossAxisCount(contentWidth);

              final sidebar = _buildSidebar(
                context,
                contentWidth,
                _cirrusPhotos.length,
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
                    const Center(child: Text('Failed to load photos')),
                  );
                }
                return buildDesktop(_buildPhotoGrid(photos, crossAxisCount));
              }

              // Mobile: nav panel lives *above* the photo grid in the scroll
              // stack. On mount we jump to navHeight so only the grid is
              // visible. Scroll up to reveal the nav.
              if (snapshot.connectionState == ConnectionState.waiting &&
                  photos.isEmpty) {
                return const Center(child: CircularProgressIndicator());
              }
              if (snapshot.hasError) {
                return const Center(child: Text('Failed to load photos'));
              }

              final showLoadingIndicator =
                  _hasMoreCirrus &&
                  (_selectedCategory == PhotoCategory.cirrus ||
                      _selectedCategory == PhotoCategory.all);
              final itemCount = photos.length + (showLoadingIndicator ? 1 : 0);

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
                                  color: Theme.of(context).colorScheme.outline,
                                ),
                              ],
                            ),
                          ),
                        ),
                        // Photo grid
                        photos.isEmpty
                            ? const SliverFillRemaining(
                                child: EmptyStateWidget(
                                  icon: Icons.photo_library_outlined,
                                  headline: 'No photos yet',
                                  subtext:
                                      'Photos you upload to AutoButler will appear here.',
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
                                Theme.of(
                                  context,
                                ).colorScheme.surface.withValues(alpha: 0.7),
                                Colors.transparent,
                              ],
                            ),
                          ),
                          child: Center(
                            child: Icon(
                              Icons.keyboard_arrow_up_rounded,
                              size: 20,
                              color: Theme.of(
                                context,
                              ).colorScheme.onSurface.withValues(alpha: 0.4),
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
    );
  }
}
