import 'package:autobutler/controllers/file_browser_controller.dart';
import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/pages/image_viewer_page.dart';
import 'package:autobutler/pages/photos_page.dart';
import 'package:autobutler/pages/settings_page.dart';
import 'package:autobutler/pages/video_viewer_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_dialog_utils.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:autobutler/widgets/file_browser/file_actions_bar.dart';
import 'package:autobutler/widgets/file_browser/file_breadcrumb_bar.dart';
import 'package:autobutler/widgets/file_browser/file_browser_header.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class FileBrowserPage extends StatefulWidget {
  const FileBrowserPage({super.key});

  @override
  State<FileBrowserPage> createState() => _FileBrowserPageState();
}

class _FileBrowserPageState extends State<FileBrowserPage> {
  final _controller = const FileBrowserController();

  late Future<List<CirrusFileNode>> _filesFuture;
  String _currentPath = '';
  bool _isGridView = false;
  bool _isUploading = false;
  bool _isCreatingFolder = false;
  bool _noHostSelected = false;

  // Search state
  bool _isSearchMode = false;
  Future<List<CirrusFileNode>>? _searchFuture;
  String? _searchQuery;

  // Server/version state
  String? _serverVersionSemver;
  List<Map<String, dynamic>> _availableVersions = [];
  bool _isUpdatingVersion = false;

  @override
  void initState() {
    super.initState();

    // Don't attempt to load files if no host is configured; show a prompt instead.
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      _filesFuture = Future.value(const <CirrusFileNode>[]);
    } else {
      _reloadFiles();
      _loadServerVersion();
    }
  }

  void _reloadFiles() {
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      _filesFuture = Future.value(const <CirrusFileNode>[]);
      return;
    }

    _filesFuture = _controller.fetchFiles(_currentPath);
  }

  Future<void> _loadServerVersion() async {
    if (AppSettings.instance.activeHost == null) return;
    try {
      final v = await CirrusService.getInstalledVersion();
      final versions = await CirrusService.listAvailableVersions();
      if (!mounted) return;
      setState(() {
        _serverVersionSemver =
            v['semver'] as String? ?? v['version'] as String?;
        _availableVersions = versions;
      });
    } catch (_) {
      // ignore failures silently
    } finally {
      if (mounted) {}
    }
  }

  Future<void> _performUpdate(String version) async {
    if (_isUpdatingVersion) return;
    setState(() {
      _isUpdatingVersion = true;
    });
    try {
      await CirrusService.updateToVersion(version);
      if (!mounted) return;
      _showMessage('Update started for $version');
    } catch (e) {
      if (!mounted) return;
      _showMessage('Update failed: ${e.toString()}');
    } finally {
      if (mounted) {
        setState(() {
          _isUpdatingVersion = false;
        });
      }
    }
  }

  void _refreshFileState() {
    setState(() {
      _reloadFiles();
    });
  }

  Future<void> _handleUploadPressed() async {
    if (_isUploading) {
      return;
    }

    final selectedFile = await _controller.pickUploadFile();
    if (selectedFile == null) {
      return;
    }

    setState(() {
      _isUploading = true;
    });

    try {
      await _controller.uploadFile(
        currentPath: _currentPath,
        selectedFile: selectedFile,
      );

      if (!mounted) {
        return;
      }

      _refreshFileState();

      _showMessage('Uploaded ${selectedFile.filename ?? 'file'}');
    } on MissingPluginException {
      if (!mounted) {
        return;
      }

      _showMessage('File picker plugin not available. Fully restart the app.');
    } catch (_) {
      if (!mounted) {
        return;
      }

      _showMessage('Upload failed');
    } finally {
      if (mounted) {
        setState(() {
          _isUploading = false;
        });
      }
    }
  }

  Future<void> _handleCreateFolderPressed() async {
    if (_isCreatingFolder) {
      return;
    }

    final folderName = await _controller.promptFolderName(context);
    if (folderName == null) {
      return;
    }

    setState(() {
      _isCreatingFolder = true;
    });

    try {
      await _controller.createFolder(
        currentPath: _currentPath,
        folderName: folderName,
      );

      if (!mounted) {
        return;
      }

      _refreshFileState();

      _showMessage('Created folder $folderName');
    } catch (_) {
      if (!mounted) {
        return;
      }

      _showMessage('Failed to create folder');
    } finally {
      if (mounted) {
        setState(() {
          _isCreatingFolder = false;
        });
      }
    }
  }

  Future<void> _handleFileMenuAction(
    CirrusFileNode node,
    FileMenuAction action,
  ) async {
    try {
      final outcome = await _controller.handleFileAction(
        currentPath: _currentPath,
        node: node,
        action: action,
        context: context,
      );

      if (!mounted || outcome == null) {
        return;
      }

      if (action == FileMenuAction.moveRename) {
        if (outcome.shouldRefresh) {
          _refreshFileState();
        }
        return;
      }

      _applyOutcome(outcome);
    } catch (_) {
      if (!mounted) {
        return;
      }

      if (action == FileMenuAction.moveRename) {
        return;
      }

      _showMessage(_controller.failureMessage(action));
    }
  }

  void _applyOutcome(FileMenuActionOutcome outcome) {
    if (!mounted) {
      return;
    }

    if (outcome.shouldRefresh) {
      _refreshFileState();
    }

    final messenger = ScaffoldMessenger.maybeOf(context);
    if (messenger != null) {
      messenger.hideCurrentSnackBar();
      messenger.showSnackBar(SnackBar(content: Text(outcome.message)));
    }
  }

  Future<void> _handleOpenNode(CirrusFileNode node) async {
    if (node.isDir) {
      _openDirectory(node);
      return;
    }

    final lowerName = node.name.toLowerCase();
    final viewable =
        lowerName.endsWith('.jpg') ||
        lowerName.endsWith('.jpeg') ||
        lowerName.endsWith('.png') ||
        lowerName.endsWith('.gif') ||
        lowerName.endsWith('.webp') ||
        lowerName.endsWith('.mp4') ||
        lowerName.endsWith('.mov') ||
        lowerName.endsWith('.mkv') ||
        lowerName.endsWith('.webm') ||
        lowerName.endsWith('.avi') ||
        lowerName.endsWith('.mp3') ||
        lowerName.endsWith('.wav') ||
        lowerName.endsWith('.m4a') ||
        lowerName.endsWith('.aac');
    if (!viewable) {
      return;
    }

    try {
      final filePath = toRootDir(
        joinPath(_currentPath, trimTrailingSlashes(node.name)),
      );
      // Open images in-app using ImageViewer; fallback to platform handlers for other types.
      final lower = lowerName;
      if (lower.endsWith('.jpg') ||
          lower.endsWith('.jpeg') ||
          lower.endsWith('.png') ||
          lower.endsWith('.gif') ||
          lower.endsWith('.webp')) {
        final bytes = await CirrusService.downloadFileBytes(
          filePath,
          serial: serialOrNull(node.deviceSerial),
          fileName: trimTrailingSlashes(node.name),
        );
        if (bytes == null || !mounted) {
          return;
        }
        await Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => ImageViewerPage(bytes: bytes, name: node.name),
          ),
        );
        return;
      }
      if (lower.endsWith('.mp4') ||
          lower.endsWith('.mov') ||
          lower.endsWith('.mkv') ||
          lower.endsWith('.webm') ||
          lower.endsWith('.avi') ||
          lower.endsWith('.mp3') ||
          lower.endsWith('.wav') ||
          lower.endsWith('.m4a') ||
          lower.endsWith('.aac')) {
        await Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => VideoViewerPage(
              url: CirrusService.constructMediaUrl(filePath),
              name: node.name,
            ),
          ),
        );
        return;
      }
    } catch (_) {
      if (!mounted) {
        return;
      }
      _showMessage('Unable to open file');
    }
  }

  void _openDirectory(CirrusFileNode node) {
    if (!node.isDir) {
      return;
    }

    _setPath(
      _controller.nextPathForOpenDirectory(
        currentPath: _currentPath,
        node: node,
      ),
    );
  }

  Future<void> _handleSearchPressed() async {
    final query = await promptForSearchQuery(context);
    if (query == null) {
      return;
    }

    setState(() {
      _isSearchMode = true;
      _searchFuture = CirrusService.searchFiles(query);
      _searchQuery = query;
    });
  }

  void _navigateToFolder(CirrusFileNode node) {
    // Use the node's fullPath to determine the containing folder and switch to it
    final parent = parentPath(node.dirPath);
    _setPath(parent);
    setState(() {
      _isSearchMode = false;
      _searchFuture = null;
      _searchQuery = null;
    });
  }

  void _goUpOneLevel() {
    if (_currentPath.isEmpty) {
      return;
    }

    _setPath(_controller.nextPathForGoUp(_currentPath));
  }

  void _setPath(String path) {
    final normalized = normalizePath(path);
    if (normalized == _currentPath) {
      return;
    }

    setState(() {
      _currentPath = normalized;
      _reloadFiles();
    });
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: Builder(
          builder: (context) => IconButton(
            icon: const Icon(Icons.menu),
            onPressed: () => Scaffold.of(context).openDrawer(),
          ),
        ),
        title: Row(
          children: [
            const Text('Cirrus'),
            if (_serverVersionSemver != null) ...[
              const SizedBox(width: 8),
              Text(
                'v$_serverVersionSemver',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          ],
        ),
        actions: [
          IconButton(
            onPressed: _handleSearchPressed,
            icon: const Icon(Icons.search),
          ),
          IconButton(
            onPressed: () {
              setState(() => _isGridView = !_isGridView);
            },
            icon: Icon(_isGridView ? Icons.view_list : Icons.grid_view_rounded),
          ),
          PopupMenuButton<String>(
            icon: const Icon(Icons.update),
            tooltip: 'Update Autobutler',
            onSelected: (val) async {
              if (val.isEmpty) return;
              await _performUpdate(val);
            },
            itemBuilder: (ctx) {
              if (_availableVersions.isEmpty) {
                return [
                  const PopupMenuItem<String>(
                    value: '',
                    enabled: false,
                    child: Text('No updates available'),
                  ),
                ];
              }
              return _availableVersions.map((m) {
                final ver = (m['version'] as String?) ?? '';
                return PopupMenuItem<String>(value: ver, child: Text(ver));
              }).toList();
            },
          ),
        ],
      ),
      drawer: AutobutlerDrawer(
        activeSection: AutobutlerDrawerSection.cirrus,
        onTapCirrus: () {
          Navigator.of(context).pop();
        },
        onTapPhotos: () {
          Navigator.of(context).pushReplacement(
            MaterialPageRoute(builder: (_) => const PhotosPage()),
          );
        },
        onTapSettings: () async {
          await Navigator.of(
            context,
          ).push(MaterialPageRoute(builder: (_) => const SettingsPage()));
          setState(() {
            _noHostSelected = AppSettings.instance.activeHost == null;
            if (!_noHostSelected) {
              _reloadFiles();
              _loadServerVersion();
            }
          });
        },
      ),
      body: Column(
        children: [
          FileActionsBar(
            isUploading: _isUploading,
            isCreatingFolder: _isCreatingFolder,
            onUploadPressed: _handleUploadPressed,
            onCreateFolderPressed: _handleCreateFolderPressed,
            onRefreshPressed: _refreshFileState,
            isSearchMode: _isSearchMode,
          ),
          FileBreadcrumbBar(
            currentPath: _currentPath,
            onGoHome: () => _setPath(''),
            onGoUp: _goUpOneLevel,
            onPathSelected: _setPath,
            isSearchMode: _isSearchMode,
          ),
          FileBrowserHeader(
            isGridView: _isGridView,
            isSearchMode: _isSearchMode,
            filesFuture: _isSearchMode
                ? (_searchFuture ?? Future.value(const <CirrusFileNode>[]))
                : _filesFuture,
            searchQuery: _searchQuery,
            onClose: () {
              setState(() {
                _isSearchMode = false;
                _searchFuture = null;
                _searchQuery = null;
                _reloadFiles();
              });
            },
          ),
          Expanded(
            child: _noHostSelected
                ? Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Text('No target host configured.'),
                        const SizedBox(height: 8),
                        ElevatedButton(
                          onPressed: () async {
                            await Navigator.of(context).push(
                              MaterialPageRoute(
                                builder: (_) => const SettingsPage(),
                              ),
                            );
                            // After returning from settings, attempt to reload files if a host was added.
                            setState(() {
                              _noHostSelected =
                                  AppSettings.instance.activeHost == null;
                              if (!_noHostSelected) {
                                _reloadFiles();
                                _loadServerVersion();
                              }
                            });
                          },
                          child: const Text('Add target host'),
                        ),
                      ],
                    ),
                  )
                : FileBrowserView(
                    filesFuture: _isSearchMode
                        ? (_searchFuture ??
                              Future.value(const <CirrusFileNode>[]))
                        : _filesFuture,
                    onFileMenuAction: _handleFileMenuAction,
                    onOpenDirectory: _isSearchMode ? (_) {} : _handleOpenNode,
                    isGridView: _isGridView,
                    isSearchMode: _isSearchMode,
                    onNavigateToFolder: _navigateToFolder,
                  ),
          ),
        ],
      ),
    );
  }
}
