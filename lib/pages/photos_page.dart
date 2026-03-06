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
  late Future<List<CirrusFileNode>> _photosFuture;
  bool _noHostSelected = false;

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
                if (photos.isEmpty) {
                  return const Center(child: Text('No photos found'));
                }
                return RefreshIndicator(
                  onRefresh: () async => _refresh(),
                  child: GridView.builder(
                    padding: const EdgeInsets.all(2),
                    gridDelegate:
                        const SliverGridDelegateWithFixedCrossAxisCount(
                          crossAxisCount: 4,
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
                                builder: (_) => ImageViewerPage(
                                  bytes: bytes,
                                  name: p.name,
                                ),
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
              },
            ),
    );
  }
}
