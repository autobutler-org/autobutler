import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/pages/file_browser_page.dart';
import 'package:autobutler/pages/image_viewer_page.dart';
import 'package:autobutler/pages/settings_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:flutter/material.dart';

class PhotosPage extends StatefulWidget {
  const PhotosPage({super.key});

  @override
  State<PhotosPage> createState() => _PhotosPageState();
}

class _PhotosPageState extends State<PhotosPage> {
  static const int _defaultCrossAxisCount = 4;
  static const int _minPreviewColumns = 1;
  static const int _maxPreviewColumns = 8;
  static const double _minTileWidth = 80;

  late Future<List<CirrusFileNode>> _photosFuture;
  bool _noHostSelected = false;
  int _previewColumns = _defaultCrossAxisCount;

  @override
  void initState() {
    super.initState();
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      _photosFuture = Future.value(const <CirrusFileNode>[]);
    } else {
      _photosFuture = _loadPhotos();
    }
  }

  Future<List<CirrusFileNode>> _loadPhotos() async {
    final files = await CirrusService.getFiles('');
    return files
        .where((f) {
          final n = f.name.toLowerCase();
          return n.endsWith('.jpg') ||
              n.endsWith('.jpeg') ||
              n.endsWith('.png') ||
              n.endsWith('.gif') ||
              n.endsWith('.webp');
        })
        .toList(growable: false);
  }

  Future<void> _refresh() async {
    setState(() {
      _photosFuture = _loadPhotos();
    });
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

  double _previewScaleForColumns(int columns) {
    return _defaultCrossAxisCount / columns;
  }

  Widget _buildSidebar(
    BuildContext context,
    double availableWidth,
    int photoCount, {
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
    final selectedScale = _previewScaleForColumns(selectedColumns);
    final divisions = maxColumns - minColumns;

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
          Row(
            children: [
              const Icon(Icons.photo_library_outlined),
              const SizedBox(width: 8),
              Text('Cirrus: $photoCount', style: theme.textTheme.titleMedium),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildPhotoGrid(List<CirrusFileNode> photos, int crossAxisCount) {
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
                final url = CirrusService.constructThumbnailUrl(
                  p.apiPath,
                  serial: p.deviceSerial,
                );
                return MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: GestureDetector(
                    onTap: () async {
                      final bytes = await CirrusService.downloadFileBytes(
                        p.apiPath,
                        serial: p.deviceSerial,
                      );
                      if (bytes == null) return;
                      if (!mounted) return;
                      await Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) =>
                              ImageViewerPage(bytes: bytes, name: p.name),
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
                      errorBuilder: (context, error, stack) =>
                          Container(color: Colors.grey[300]),
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
          await Navigator.of(
            context,
          ).push(MaterialPageRoute(builder: (_) => const SettingsPage()));
          setState(() {
            _noHostSelected = AppSettings.instance.activeHost == null;
          });
        },
      ),
      body: _noHostSelected
          ? Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Text('No target host configured.'),
                  const SizedBox(height: 8),
                  ElevatedButton(
                    onPressed: () async {
                      await Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const SettingsPage()),
                      );
                      setState(() {
                        _noHostSelected =
                            AppSettings.instance.activeHost == null;
                        if (!_noHostSelected) _refresh();
                      });
                    },
                    child: const Text('Add target host'),
                  ),
                ],
              ),
            )
          : FutureBuilder<List<CirrusFileNode>>(
              future: _photosFuture,
              builder: (context, snapshot) {
                final photos = snapshot.data ?? const <CirrusFileNode>[];
                return LayoutBuilder(
                  builder: (context, constraints) {
                    final compact = constraints.maxWidth < 900;
                    final contentWidth = compact
                        ? constraints.maxWidth
                        : (constraints.maxWidth - 281)
                              .clamp(1.0, double.infinity)
                              .toDouble();
                    final crossAxisCount = _effectiveCrossAxisCount(
                      contentWidth,
                    );
                    final sidebar = _buildSidebar(
                      context,
                      contentWidth,
                      photos.length,
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

                    if (snapshot.connectionState == ConnectionState.waiting &&
                        photos.isEmpty) {
                      return buildShell(
                        const Center(child: CircularProgressIndicator()),
                      );
                    }

                    if (snapshot.hasError) {
                      return buildShell(
                        const Center(child: Text('Failed to load photos')),
                      );
                    }

                    return buildShell(_buildPhotoGrid(photos, crossAxisCount));
                  },
                );
              },
            ),
    );
  }
}
