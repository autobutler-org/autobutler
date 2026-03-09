import 'dart:typed_data';
import 'dart:io';

import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/pages/file_browser_page.dart';
import 'package:autobutler/pages/image_viewer_page.dart';
import 'package:autobutler/pages/settings_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:flutter/material.dart';
import 'package:photo_manager/photo_manager.dart';

class PhotosPage extends StatefulWidget {
  const PhotosPage({super.key});

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
}

class _PhotosPageState extends State<PhotosPage> {
  static const int _defaultCrossAxisCount = 4;
  static const int _minPreviewColumns = 1;
  static const int _maxPreviewColumns = 8;
  static const double _minTileWidth = 80;

  late Future<List<PhotoItem>> _photosFuture;
  Future<List<PhotoItem>>? _cirrusFuture;
  Future<List<PhotoItem>>? _mobileFuture;

  bool _noHostSelected = false;
  int _previewColumns = _defaultCrossAxisCount;
  PhotoCategory _selectedCategory = PhotoCategory.cirrus;

  @override
  void initState() {
    super.initState();
    _noHostSelected = AppSettings.instance.activeHost == null;
    _cirrusFuture = _noHostSelected ? Future.value(const <PhotoItem>[]) : _loadCirrusPhotos();
    _photosFuture = _cirrusFuture!;
  }

  Future<List<PhotoItem>> _loadCirrusPhotos() async {
    final files = await CirrusService.getFiles('');
    final filtered = files
        .where((f) {
          final n = f.name.toLowerCase();
          return n.endsWith('.jpg') ||
              n.endsWith('.jpeg') ||
              n.endsWith('.png') ||
              n.endsWith('.gif') ||
              n.endsWith('.webp');
        })
        .map((f) => PhotoItem.fromCirrus(f))
        .toList(growable: false);
    return filtered;
  }

  Future<List<PhotoItem>> _loadMobilePhotos() async {
    // Only attempt mobile photo loading on Android or iOS devices.
    if (!(Platform.isAndroid || Platform.isIOS)) return [];

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
    final List<AssetEntity> assets = await all.getAssetListPaged(page: 0, size: 200);
    return assets.map((a) => PhotoItem.fromAsset(a)).toList(growable: false);
  }

  Future<void> _selectCategory(PhotoCategory cat) async {
    setState(() {
      _selectedCategory = cat;
    });

    if (cat == PhotoCategory.cirrus) {
      _cirrusFuture ??= _loadCirrusPhotos();
      setState(() {
        _photosFuture = _cirrusFuture!;
      });
    } else if (cat == PhotoCategory.mobile) {
      _mobileFuture ??= _loadMobilePhotos();
      setState(() {
        _photosFuture = _mobileFuture!;
      });
    } else {
      _cirrusFuture ??= _loadCirrusPhotos();
      _mobileFuture ??= _loadMobilePhotos();
      setState(() {
        _photosFuture = Future.wait([_cirrusFuture!, _mobileFuture!])
            .then((lists) => lists.expand((l) => l).toList(growable: false));
      });
    }
  }

  Future<void> _refresh() async {
    _cirrusFuture = null;
    _mobileFuture = null;
    await _selectCategory(_selectedCategory);
  }

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
      return TextButton(
        onPressed: () => _selectCategory(cat),
        style: TextButton.styleFrom(
          padding: const EdgeInsets.symmetric(vertical: 8),
          alignment: Alignment.centerLeft,
        ),
        child: Row(
          children: [
            Icon(
              cat == PhotoCategory.cirrus ? Icons.cloud : (cat == PhotoCategory.mobile ? Icons.smartphone : Icons.photo_library),
              color: selected ? theme.colorScheme.primary : null,
            ),
            const SizedBox(width: 8),
            Expanded(child: Text('$label: $count', style: theme.textTheme.titleMedium)),
            if (selected) const Icon(Icons.check, size: 16),
          ],
        ),
      );
    }

    return Container(
      width: compact ? double.infinity : 280,
      padding: const EdgeInsets.all(16),
      color: theme.colorScheme.surfaceContainerLowest,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Photos', style: theme.textTheme.titleLarge),
          const SizedBox(height: 16),
          Text('Photos per row', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          Slider(
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
          const SizedBox(height: 20),
          categoryButton(PhotoCategory.cirrus, 'Cirrus', cirrusCount),
          categoryButton(PhotoCategory.mobile, 'Mobile', mobileCount),
          categoryButton(PhotoCategory.all, 'All', cirrusCount + mobileCount),
        ],
      ),
    );
  }

  Widget _buildPhotoGrid(List<PhotoItem> photos, int crossAxisCount) {
    return RefreshIndicator(
      onRefresh: () async => _refresh(),
      child: photos.isEmpty
          ? ListView(
              children: const [
                SizedBox(height: 120),
                Center(child: Text('No photos found')),
              ],
            )
          : GridView.builder(
              padding: const EdgeInsets.all(2),
              gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: crossAxisCount,
                crossAxisSpacing: 2,
                mainAxisSpacing: 2,
              ),
              itemCount: photos.length,
              itemBuilder: (context, idx) {
                final p = photos[idx];

                if (p.isCirrus) {
                  final c = p.cirrus!;
                  final url = CirrusService.constructThumbnailUrl(
                    c.apiPath,
                    serial: c.deviceSerial,
                  );
                  return MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: GestureDetector(
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
                            builder: (_) => ImageViewerPage(bytes: bytes, name: c.name),
                          ),
                        );
                      },
                      child: Image.network(
                        url.toString(),
                        fit: BoxFit.cover,
                        loadingBuilder: (context, child, progress) {
                          if (progress == null) return child;
                          return Container(color: Colors.grey[300]);
                        },
                        errorBuilder: (context, error, stack) => Container(color: Colors.grey[300]),
                      ),
                    ),
                  );
                }

                // Mobile asset
                final a = p.asset!;
                return MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: GestureDetector(
                    onTap: () async {
                      final navigator = Navigator.of(context);
                      final bytes = await a.originBytes;
                      if (bytes == null) return;
                      if (!mounted) return;
                      await navigator.push(
                        MaterialPageRoute(
                          builder: (_) => ImageViewerPage(bytes: bytes, name: a.id),
                        ),
                      );
                    },
                    child: FutureBuilder<Uint8List?>(
                      future: a.thumbnailDataWithSize(ThumbnailSize(200, 200)),
                      builder: (context, snap) {
                        final thumb = snap.data;
                        if (thumb == null) return Container(color: Colors.grey[300]);
                        return Image.memory(thumb, fit: BoxFit.cover);
                      },
                    ),
                  ),
                );
              },
            ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Photos')),
      drawer: AutobutlerDrawer(
        activeSection: AutobutlerDrawerSection.photos,
        onTapCirrus: () {
          Navigator.of(context).pushReplacement(
            MaterialPageRoute(builder: (_) => const FileBrowserPage()),
          );
        },
        onTapPhotos: () {
          Navigator.of(context).pop();
        },
        onTapSettings: () async {
          await Navigator.of(context).push(MaterialPageRoute(builder: (_) => const SettingsPage()));
          setState(() {
            _noHostSelected = AppSettings.instance.activeHost == null;
          });
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
                  : (constraints.maxWidth - 281).clamp(1.0, double.infinity).toDouble();
              final crossAxisCount = _effectiveCrossAxisCount(contentWidth);

              final sidebar = _buildSidebar(
                context,
                contentWidth,
                // provide counts if futures completed, otherwise 0
                photos.where((p) => p.isCirrus).length,
                photos.where((p) => !p.isCirrus).length,
                compact: compact,
              );

              Widget buildShell(Widget content) {
                if (compact) {
                  return Column(
                    children: [
                      sidebar,
                      const Divider(height: 1),
                      Expanded(child: content),
                    ],
                  );
                }

                return Row(
                  children: [
                    sidebar,
                    const VerticalDivider(width: 1),
                    Expanded(child: content),
                  ],
                );
              }

              if (snapshot.connectionState == ConnectionState.waiting && photos.isEmpty) {
                return buildShell(const Center(child: CircularProgressIndicator()));
              }

              if (snapshot.hasError) {
                return buildShell(const Center(child: Text('Failed to load photos')));
              }

              return buildShell(_buildPhotoGrid(photos, crossAxisCount));
            },
          );
        },
      ),
    );
  }
}
