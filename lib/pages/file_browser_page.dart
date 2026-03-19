import 'dart:async';

import 'package:autobutler/controllers/file_browser_controller.dart';
import 'package:autobutler/utils/auto_refresh_mixin.dart';
import 'package:autobutler/widgets/refresh_icon_button.dart';
import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/pages/image_viewer_page.dart';
import 'package:autobutler/pages/settings_page.dart';
import 'package:autobutler/pages/video_viewer_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_dialog_utils.dart';
import 'package:autobutler/utils/file_browser_drag_config.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:autobutler/utils/safe_set_state_mixin.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:autobutler/widgets/file_browser/file_actions_bar.dart';
import 'package:autobutler/widgets/file_browser/file_breadcrumb_bar.dart';
import 'package:autobutler/widgets/file_browser/file_browser_header.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:flutter/foundation.dart';
import 'package:autobutler/router.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;

class FileBrowserPage extends StatefulWidget {
  const FileBrowserPage({super.key});

  @override
  State<FileBrowserPage> createState() => _FileBrowserPageState();
}

class _FileBrowserPageState extends State<FileBrowserPage>
    with SafeSetStateMixin, WidgetsBindingObserver, AutoRefreshMixin {
  final _controller = const FileBrowserController();
  final _dropRegionKey = GlobalKey();
  final _fileBrowserScrollController = ScrollController();

  late Future<List<CirrusFileNode>> _filesFuture;
  List<CirrusFileNode>?
  _cachedFiles; // last successful result, shown during refresh
  int _generation = 0; // incremented on each reload to discard stale fetches
  // Optimistic overlay: when non-null, shown instead of _cachedFiles until
  // the next real fetch completes. Allows instant UI feedback for mutations.
  List<CirrusFileNode>? _optimisticFiles;
  String _currentPath = '';
  bool _isGridView = false;
  bool _isUploading = false;
  bool _isCreatingFolder = false;
  bool _isWebDragging = false;
  bool _isHoveringFolderDropTarget = false;
  bool _noHostSelected = false;
  Timer? _folderDragExitTimer;

  // Search state
  bool _isSearchMode = false;
  Future<List<CirrusFileNode>>? _searchFuture;
  String? _searchQuery;

  @override
  Future<void> refresh() async {
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      setState(() {
        _filesFuture = Future.value(const <CirrusFileNode>[]);
      });
      return;
    }
    setState(() => _reloadFiles());
    await _filesFuture;
  }

  @override
  void dispose() {
    _folderDragExitTimer?.cancel();
    _fileBrowserScrollController.dispose();
    super.dispose();
  }

  void _reloadFiles() {
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      _filesFuture = Future.value(const <CirrusFileNode>[]);
      return;
    }

    final generation = ++_generation;
    _filesFuture = _controller.fetchFiles(_currentPath).then((files) {
      if (mounted && _generation == generation) {
        setState(() {
          _cachedFiles = files;
          _optimisticFiles =
              null; // ground truth arrived — clear optimistic overlay
        });
      }
      return files;
    });
  }

  Future<void> _refreshFileState() => manualRefresh();

  /// Returns the current display list: optimistic overlay if set, otherwise
  /// the last cached result.
  List<CirrusFileNode>? get _displayFiles => _optimisticFiles ?? _cachedFiles;

  /// Applies an optimistic mutation to the current display list and triggers
  /// a real refresh in the background. If the real fetch fails, the optimistic
  /// state remains until the next successful fetch.
  void _applyOptimisticUpdate(
    List<CirrusFileNode> Function(List<CirrusFileNode>) mutate,
  ) {
    final current = _displayFiles ?? const [];
    setState(() => _optimisticFiles = mutate(current));
    // Background refresh — updates _cachedFiles and clears _optimisticFiles.
    _refreshFileState();
  }

  /// Optimistically adds a file placeholder after an upload.
  void _optimisticAddFile(String fileName, String uploadPath) {
    _applyOptimisticUpdate((files) {
      final newNode = CirrusFileNode(
        name: fileName,
        size: 0,
        isDir: false,
        deviceName: files.isNotEmpty ? files.first.deviceName : '',
        devicePath: files.isNotEmpty ? files.first.devicePath : '',
        deviceSerial: files.isNotEmpty ? files.first.deviceSerial : '',
        dirPath: uploadPath.isEmpty ? fileName : '$uploadPath/$fileName',
      );
      return [...files, newNode];
    });
  }

  /// Optimistically removes a node after delete.
  void _optimisticRemoveNode(CirrusFileNode node) {
    _applyOptimisticUpdate(
      (files) => files.where((f) => f.apiPath != node.apiPath).toList(),
    );
  }

  /// Optimistically adds a folder placeholder after creation.
  void _optimisticAddFolder(String folderName, String currentPath) {
    _applyOptimisticUpdate((files) {
      final newNode = CirrusFileNode(
        name: folderName,
        size: 0,
        isDir: true,
        deviceName: files.isNotEmpty ? files.first.deviceName : '',
        devicePath: files.isNotEmpty ? files.first.devicePath : '',
        deviceSerial: files.isNotEmpty ? files.first.deviceSerial : '',
        dirPath: currentPath.isEmpty ? folderName : '$currentPath/$folderName',
      );
      return [...files, newNode];
    });
  }

  Future<void> _uploadSelectedFiles(
    List<http.MultipartFile> selectedFiles,
    String uploadPath,
  ) async {
    if (_isUploading || selectedFiles.isEmpty) {
      return;
    }

    setState(() {
      _isUploading = true;
    });

    // Optimistically add placeholders immediately so the UI reflects the
    // pending upload without waiting for the backend round-trip.
    for (final file in selectedFiles) {
      if (file.filename != null) {
        _optimisticAddFile(file.filename!, uploadPath);
      }
    }

    try {
      await _controller.uploadFiles(
        currentPath: uploadPath,
        selectedFiles: selectedFiles,
      );

      if (!mounted) {
        return;
      }

      final uploadedLabel = selectedFiles.length == 1
          ? selectedFiles.first.filename ?? 'file'
          : '${selectedFiles.length} files';
      _showMessage('Uploaded $uploadedLabel');
      // _applyOptimisticUpdate already triggered a background refresh;
      // no explicit call needed here.
    } catch (e) {
      debugPrint('[file_browser_page.dart] Upload error: $e');
      if (!mounted) {
        return;
      }
      // Rollback: clear the optimistic overlay so the stale cache is shown
      // instead of the phantom files.
      setState(() => _optimisticFiles = null);
      _showMessage('Upload failed');
    } finally {
      if (mounted) {
        setState(() {
          _isUploading = false;
        });
      }
    }
  }

  Future<void> _handleUploadPressed() async {
    if (_isUploading) {
      return;
    }

    try {
      final selectedFile = await _controller.pickUploadFile();
      if (selectedFile == null) {
        return;
      }

      await _uploadSelectedFiles([selectedFile], _currentPath);
    } on MissingPluginException {
      if (!mounted) {
        return;
      }

      _showMessage('File picker plugin not available. Fully restart the app.');
    }
  }

  Future<void> _handleDroppedItems({
    required List<DropItem> droppedItems,
    required String uploadPath,
  }) async {
    // Drag-and-drop upload is currently web-only. The desktop_drop package
    // supports native desktop platforms too — native support can be enabled
    // here in a follow-up once it's been validated on macOS/Linux/Windows.
    if (!kIsWeb || droppedItems.isEmpty || _isUploading) {
      return;
    }

    try {
      final selectedFiles = <http.MultipartFile>[];
      for (final droppedItem in droppedItems) {
        if (droppedItem is! DropItemFile) {
          continue;
        }

        final bytes = await _readDroppedFileBytes(droppedItem);
        if (bytes == null || bytes.isEmpty) {
          continue;
        }

        selectedFiles.add(
          _controller.multipartFileFromBytes(
            bytes: bytes,
            filename: droppedItem.name,
          ),
        );
      }

      if (selectedFiles.isEmpty) {
        _showMessage('No files to upload');
        return;
      }

      await _uploadSelectedFiles(selectedFiles, uploadPath);
    } catch (_) {
      debugPrint('[file_browser_page.dart] Error in catch block');
      _showMessage('Unable to read dropped files');
    }
  }

  Future<Uint8List?> _readDroppedFileBytes(DropItemFile droppedItem) async {
    try {
      return await droppedItem.readAsBytes();
    } catch (_) {
      // Some browser drag sources (e.g. dragging from another browser tab or
      // certain file managers) expose an HTTP/HTTPS URL via droppedItem.path
      // rather than providing raw bytes directly. Blob URLs (blob:...) are
      // not fetchable this way — this fallback only applies to http/https paths.
      if (!kIsWeb) {
        rethrow;
      }

      final path = droppedItem.path;
      if (path.isEmpty) {
        return null;
      }

      final fallbackResponse = await http.get(Uri.parse(path));
      if (fallbackResponse.statusCode >= 200 &&
          fallbackResponse.statusCode < 300) {
        return fallbackResponse.bodyBytes;
      }

      throw Exception(
        'Dropped file read failed (${fallbackResponse.statusCode})',
      );
    }
  }

  Future<void> _handleDropToCurrentFolder(DropDoneDetails details) {
    return _handleDroppedItems(
      droppedItems: details.files,
      uploadPath: _currentPath,
    );
  }

  Future<void> _handleDropToFolder(
    List<DropItem> droppedItems,
    String folderPath,
  ) {
    return _handleDroppedItems(
      droppedItems: droppedItems,
      uploadPath: folderPath,
    );
  }

  void _handleFolderDragEnter() {
    _folderDragExitTimer?.cancel();

    if (!mounted || _isHoveringFolderDropTarget) {
      return;
    }
    setStateSafely(() {
      _isHoveringFolderDropTarget = true;
      _isWebDragging = false;
    });
  }

  void _handleFolderDragExit() {
    _folderDragExitTimer?.cancel();
    _folderDragExitTimer = Timer(
      const Duration(
        milliseconds: FileBrowserDragConfig.folderHoverExitDebounceMs,
      ),
      () {
        if (!mounted || !_isHoveringFolderDropTarget) {
          return;
        }
        setStateSafely(() {
          _isHoveringFolderDropTarget = false;
        });
      },
    );
  }

  void _maybeAutoScrollDuringDrag(double localDy) {
    if (!_fileBrowserScrollController.hasClients) {
      return;
    }

    final viewportHeight = _dropRegionKey.currentContext?.size?.height;
    if (viewportHeight == null || viewportHeight <= 0) {
      return;
    }

    const edgeActivation = FileBrowserDragConfig.autoScrollEdgeActivationPx;
    const baseDelta = FileBrowserDragConfig.autoScrollBaseDeltaPx;
    const maxExtraDelta = FileBrowserDragConfig.autoScrollMaxExtraDeltaPx;

    double delta = 0;
    if (localDy < edgeActivation) {
      final strength = ((edgeActivation - localDy) / edgeActivation).clamp(
        0.0,
        1.0,
      );
      delta = -(baseDelta + maxExtraDelta * strength);
    } else if (localDy > viewportHeight - edgeActivation) {
      final strength =
          ((localDy - (viewportHeight - edgeActivation)) / edgeActivation)
              .clamp(0.0, 1.0);
      delta = baseDelta + maxExtraDelta * strength;
    }

    if (delta == 0) {
      return;
    }

    final position = _fileBrowserScrollController.position;
    final targetOffset = (position.pixels + delta).clamp(
      position.minScrollExtent,
      position.maxScrollExtent,
    );
    if (targetOffset == position.pixels) {
      return;
    }

    _fileBrowserScrollController.jumpTo(targetOffset);
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

    // Optimistic: add folder placeholder immediately.
    _optimisticAddFolder(folderName, _currentPath);

    try {
      await _controller.createFolder(
        currentPath: _currentPath,
        folderName: folderName,
      );

      if (!mounted) {
        return;
      }

      _showMessage('Created folder $folderName');
    } catch (e) {
      debugPrint('[file_browser_page.dart] Create folder error: $e');
      if (!mounted) {
        return;
      }
      setState(() => _optimisticFiles = null);
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
        node: node,
        action: action,
        context: context,
      );

      if (!mounted || outcome == null) {
        return;
      }

      // Apply optimistic updates immediately after a confirmed successful
      // action — the UI reflects the change before the background refresh
      // returns ground truth from the backend.
      if (outcome.shouldRefresh) {
        switch (action) {
          case FileMenuAction.delete:
            _optimisticRemoveNode(node);
          case FileMenuAction.moveRename:
            // The outcome message carries the new path; fall back to refresh.
            _refreshFileState();
          default:
            _refreshFileState();
        }
      }

      _showOutcomeMessage(outcome);
    } catch (e) {
      debugPrint('[file_browser_page.dart] File action error: $e');
      if (!mounted) {
        return;
      }

      if (action == FileMenuAction.moveRename) {
        return;
      }

      // Rollback any optimistic state that may have been applied before the
      // error (e.g. if the action partially executed).
      setState(() => _optimisticFiles = null);
      _showMessage(_controller.failureMessage(action));
    }
  }

  void _showOutcomeMessage(FileMenuActionOutcome outcome) {
    if (!mounted) {
      return;
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
      final filePath = node.apiPath;
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
      debugPrint('[file_browser_page.dart] Error in catch block');
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
    // Use the node's API path to determine the containing folder and switch to it.
    final parent = parentPath(node.apiPath);
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
        title: const Text('Cirrus'),
        centerTitle: true,
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
          RefreshIconButton(
            isRefreshing: isRefreshing,
            onPressed: _refreshFileState,
            tooltip: 'Refresh files',
          ),
        ],
      ),
      drawer: AutobutlerDrawer(
        activeSection: AutobutlerDrawerSection.cirrus,
        onTapCirrus: () {
          Navigator.of(context).pop();
        },
        onTapPhotos: () {
          context.go(AppRoutes.photos);
        },
        onTapHealth: () {
          context.go(AppRoutes.health);
        },
        onTapSettings: () {
          context.go(AppRoutes.settings);
        },
      ),
      body: Column(
        children: [
          FileActionsBar(
            isUploading: _isUploading,
            isCreatingFolder: _isCreatingFolder,
            onUploadPressed: _handleUploadPressed,
            onCreateFolderPressed: _handleCreateFolderPressed,
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
                              }
                            });
                          },
                          child: const Text('Add target host'),
                        ),
                      ],
                    ),
                  )
                : DropTarget(
                    key: _dropRegionKey,
                    enable: kIsWeb && !_isSearchMode && !_isUploading,
                    onDragEntered: (_) {
                      if (!mounted) {
                        return;
                      }
                      setStateSafely(() {
                        _isWebDragging = true;
                      });
                    },
                    onDragExited: (_) {
                      if (!mounted) {
                        return;
                      }
                      setStateSafely(() {
                        _isWebDragging = false;
                      });
                    },
                    onDragUpdated: (details) {
                      _maybeAutoScrollDuringDrag(details.localPosition.dy);
                    },
                    onDragDone: (details) async {
                      _folderDragExitTimer?.cancel();
                      if (mounted) {
                        setStateSafely(() {
                          _isWebDragging = false;
                        });
                      }

                      if (_isHoveringFolderDropTarget) {
                        return;
                      }

                      await _handleDropToCurrentFolder(details);
                    },
                    child: Stack(
                      fit: StackFit.expand,
                      children: [
                        FileBrowserView(
                          filesFuture: _isSearchMode
                              ? (_searchFuture ??
                                    Future.value(const <CirrusFileNode>[]))
                              : _filesFuture,
                          initialData: _isSearchMode ? null : _displayFiles,
                          onFileMenuAction: _handleFileMenuAction,
                          onOpenDirectory: _isSearchMode
                              ? (_) {}
                              : _handleOpenNode,
                          isGridView: _isGridView,
                          isSearchMode: _isSearchMode,
                          onNavigateToFolder: _navigateToFolder,
                          currentPath: _currentPath,
                          onDropToFolder: _handleDropToFolder,
                          onFolderDragEnter: _handleFolderDragEnter,
                          onFolderDragExit: _handleFolderDragExit,
                          scrollController: _fileBrowserScrollController,
                        ),
                        if (_isWebDragging && !_isHoveringFolderDropTarget)
                          IgnorePointer(
                            child: Container(
                              decoration: BoxDecoration(
                                border: Border.all(
                                  color: Theme.of(context).colorScheme.primary,
                                  width: 1.5,
                                ),
                                color: Theme.of(context)
                                    .colorScheme
                                    .primaryContainer
                                    .withValues(alpha: 0.20),
                              ),
                              alignment: Alignment.topCenter,
                              padding: const EdgeInsets.only(top: 10),
                            ),
                          ),
                      ],
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}
