import 'dart:async';
import 'dart:convert';

import 'package:data_table/data_sheet.dart';
import 'package:data_table/data_table.dart' as dt;
import 'package:desktop_drop/desktop_drop.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:quark/controllers/file_browser_controller.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/pages/document_editor_page.dart';
import 'package:quark/pages/generic_file_viewer_page.dart';
import 'package:quark/pages/image_viewer_page.dart';
import 'package:quark/pages/spreadsheet_editor_page.dart';
import 'package:quark/pages/video_viewer_page.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/services/events_service.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/auto_refresh_mixin.dart';
import 'package:quark/utils/files_route_path_utils.dart';
import 'package:quark/utils/file_browser_dialog_utils.dart';
import 'package:quark/utils/file_browser_drag_config.dart';
import 'package:quark/utils/file_browser_path_utils.dart';
import 'package:quark/utils/safe_set_state_mixin.dart';
import 'package:quark/widgets/core/empty_state_widget.dart';
import 'package:quark/widgets/device_upload_picker.dart';
import 'package:quark/widgets/file_browser/file_browser_header.dart';
import 'package:quark/widgets/file_browser/file_browser_view.dart';
import 'package:quark/widgets/file_browser/file_storage_footer.dart';
import 'package:quark/widgets/file_browser/file_top_bar.dart';
import 'package:quark/widgets/file_browser/new_file_dialog.dart';
import 'package:quark/widgets/file_browser/recent_files_section.dart';
import 'package:quark/widgets/quark_drawer.dart';
import 'package:quark_icons/quark_icons.dart';

class FileBrowserPage extends StatefulWidget {
  /// Optional path to navigate to on load, e.g. 'photos/2024'.
  /// When non-null and non-empty, the browser opens at this path instead of root.
  final String? initialPath;

  const FileBrowserPage({super.key, this.initialPath});

  @override
  State<FileBrowserPage> createState() => _FileBrowserPageState();
}

class _FileBrowserPageState extends State<FileBrowserPage>
    with SafeSetStateMixin, WidgetsBindingObserver, AutoRefreshMixin {
  final _controller = const FileBrowserController();
  final _dropRegionKey = GlobalKey();
  final _fileBrowserScrollController = ScrollController();

  // FAB visibility — hidden when the user scrolls down, restored on scroll up.
  bool _fabVisible = true;
  double _lastScrollOffset = 0.0;

  Future<List<FileNode>> _filesFuture = Future.value(const <FileNode>[]);
  List<FileNode>? _cachedFiles; // last successful result, shown during refresh
  int _generation = 0; // incremented on each reload to discard stale fetches
  String _currentPath = '';

  /// If the deep-link URL pointed directly to a file, open its editor once
  /// the page has mounted. Only consumed once.
  String? _pendingFileOpen;

  bool _handlingPendingFile = false;
  _FilesRouteFailure? _routeFailure;
  bool _isGridView = false;

  /// When true, files from all devices are shown merged (unified).
  /// When false, they are grouped by device with section headers.
  bool _isUnifiedView = true;

  // Device filter state (#801)
  List<StorageDevice> _allDevices = [];

  /// Tracks selected devices by devicePath (unique per device, unlike serial).
  Set<String> _activeDevicePaths = {};
  bool _isUploading = false;
  int _uploadTotal = 0;
  int _uploadCompleted = 0;
  int _recentFilesSectionKey = 0;
  bool _isCreatingFolder = false;
  bool _isWebDragging = false;
  bool _isHoveringFolderDropTarget = false;
  bool _noHostSelected = false;
  Timer? _folderDragExitTimer;

  // WebSocket event subscription for real-time file updates
  StreamSubscription<FileEvent>? _eventSub;

  // Search state
  bool _isSearchMode = false;
  Future<List<FileNode>>? _searchFuture;
  String? _searchQuery;

  // Multi-select / batch delete state (#986)
  bool _selectionMode = false;
  final Set<String> _selectedPaths = {};
  List<FileNode> _allCurrentFiles = [];

  // Archive browser state — non-null when navigating inside an archive.
  _ArchiveContext? _archiveContext;

  void _applyIncomingRoutePath(String? initialPath) {
    final normalized = initialPath == null ? '' : normalizePath(initialPath);

    _archiveContext = null;
    _routeFailure = null;
    _isSearchMode = false;
    _searchFuture = null;
    _searchQuery = null;
    _currentPath = normalized;
    _pendingFileOpen = normalized.isEmpty ? null : normalized;
    _cachedFiles = FileBrowserCache.instance.get(normalized);
  }

  @override
  void initState() {
    // Apply deep-link initial path before AutoRefreshMixin triggers the first load.
    _applyIncomingRoutePath(widget.initialPath);
    super
        .initState(); // AutoRefreshMixin.initState handles timer + initial load
    _fileBrowserScrollController.addListener(_onScroll);
    EventsService.instance.start();
    // If the deep-link URL pointed at a file, open its editor after the first
    // frame so the folder content is loaded beneath it.
    if (_pendingFileOpen != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        final pending = _pendingFileOpen;
        _pendingFileOpen = null;
        if (pending != null && mounted) {
          _openPendingFile(pending);
        }
      });
    }
    _eventSub = EventsService.instance.events.listen((evt) {
      // Any file mutation on the server triggers a refresh
      if ({'upload', 'delete', 'move', 'new_folder'}.contains(evt.kind)) {
        manualRefresh();
      }
    });
  }

  @override
  void didUpdateWidget(covariant FileBrowserPage oldWidget) {
    super.didUpdateWidget(oldWidget);

    final oldPath = normalizePath(oldWidget.initialPath ?? '');
    final newPath = normalizePath(widget.initialPath ?? '');
    if (oldPath == newPath) {
      return;
    }

    if (newPath.isNotEmpty && FileBrowserCache.instance.isFileOpen(newPath)) {
      return;
    }

    setState(() {
      _applyIncomingRoutePath(widget.initialPath);
      _reloadFiles();
    });

    if (_pendingFileOpen != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        final pending = _pendingFileOpen;
        _pendingFileOpen = null;
        if (pending != null && mounted) {
          _openPendingFile(pending);
        }
      });
    }
  }

  @override
  Future<void> refresh() async {
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      setState(() {
        _filesFuture = Future.value(const <FileNode>[]);
      });
      return;
    }
    await _loadDevices();
    if (!mounted) return;
    setState(() => _reloadFiles());
    await _filesFuture;
  }

  Future<void> _loadDevices() async {
    if (AppSettings.instance.activeHost == null) return;
    try {
      final devices = await StorageService.listDevices();
      if (!mounted) return;
      setState(() {
        _allDevices = devices;
        // Preserve existing selection if devices haven't changed;
        // otherwise select all.
        final newPaths = devices.map((d) => d.devicePath).toSet();
        if (!newPaths.containsAll(_activeDevicePaths) ||
            _activeDevicePaths.isEmpty) {
          _activeDevicePaths = newPaths;
        }
      });
    } catch (e) {
      debugPrint('[file_browser_page.dart] Failed to load devices: $e');
    }
  }

  /// Converts the selected devicePaths into the serial values the backend expects.
  /// Returns an empty list when all devices are selected (backend interprets
  /// empty serials as "no filter — show all").
  List<String> _serialsForActiveDevices() {
    final allPaths = _allDevices.map((d) => d.devicePath).toSet();
    // If every known device is selected, pass no filter rather than an explicit
    // serial list. This avoids the edge case where count matches but content
    // differs (e.g. a device was added/removed between loads), and also avoids
    // sending serial='' for the internal device which confuses the backend.
    if (_activeDevicePaths.containsAll(allPaths) &&
        allPaths.containsAll(_activeDevicePaths)) {
      return const [];
    }
    return _allDevices
        .where((d) => _activeDevicePaths.contains(d.devicePath))
        .map((d) => d.serial)
        .where((s) => s.isNotEmpty) // never send blank serial as a filter
        .toList();
  }

  @override
  void dispose() {
    _eventSub?.cancel();
    _folderDragExitTimer?.cancel();
    _fileBrowserScrollController.dispose();
    super.dispose();
  }

  void _reloadFiles() {
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      _filesFuture = Future.value(const <FileNode>[]);
      return;
    }

    final generation = ++_generation;
    final serials = _serialsForActiveDevices();
    final archive = _archiveContext;

    Future<List<FileNode>> fetchFuture;
    if (archive != null) {
      fetchFuture = FilesService.listArchiveEntries(
        archive.archivePath,
        subPath: archive.subPath,
        serial: serials.isNotEmpty ? serials.first : null,
      );
    } else {
      fetchFuture = _controller.fetchFiles(
        _currentPath,
        serials: serials.isEmpty ? null : serials,
      );
    }

    _filesFuture = fetchFuture.then((files) {
      if (mounted && _generation == generation) {
        setState(() {
          _cachedFiles = files;
          _allCurrentFiles = files;
        });
        // Cache the listing so rebuilt pages (from context.go navigation) can
        // display it instantly while a fresh fetch is in flight.
        if (archive == null) {
          FileBrowserCache.instance.put(_currentPath, files);
        }
      }
      return files;
    });
  }

  Future<void> _refreshFileState() => manualRefresh();

  // ── Multi-select / batch delete (#986) ──────────────────────────────────

  void _onSelectionChanged(FileNode node, {required bool enterSelectionMode}) {
    setState(() {
      if (enterSelectionMode && !_selectionMode) {
        _selectionMode = true;
        _selectedPaths.clear();
      }
      if (_selectedPaths.contains(node.apiPath)) {
        _selectedPaths.remove(node.apiPath);
      } else {
        _selectedPaths.add(node.apiPath);
      }
    });
  }

  void _exitSelectionMode() {
    setState(() {
      _selectionMode = false;
      _selectedPaths.clear();
    });
  }

  void _selectAll() {
    setState(() {
      _selectedPaths
        ..clear()
        ..addAll(_allCurrentFiles.map((n) => n.apiPath));
    });
  }

  Future<void> _deleteSelected() async {
    if (_selectedPaths.isEmpty) return;
    final nodes = _allCurrentFiles
        .where((n) => _selectedPaths.contains(n.apiPath))
        .toList();
    if (nodes.isEmpty) return;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete selected'),
        content: Text(
          'Delete ${nodes.length} item${nodes.length == 1 ? '' : 's'}? This cannot be undone.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    // Optimistic removal.
    final snapshot = _cachedFiles;
    setState(() {
      if (_cachedFiles != null) {
        _cachedFiles = _cachedFiles!
            .where((n) => !_selectedPaths.contains(n.apiPath))
            .toList();
        FileBrowserCache.instance.put(_currentPath, _cachedFiles!);
      }
    });
    _exitSelectionMode();

    try {
      await _controller.deleteNodes(nodes: nodes);
      if (mounted) {
        _showMessage(
          'Deleted ${nodes.length} item${nodes.length == 1 ? "" : "s"}',
        );
      }
    } catch (_) {
      if (!mounted) return;
      if (snapshot != null) setState(() => _cachedFiles = snapshot);
      _showMessage('Delete failed');
    }
  }

  // ── Optimistic updates ───────────────────────────────────────────────────

  /// Immediately remove a node from the displayed list.
  /// If the server call fails, [_refreshFileState] will reconcile.
  void _optimisticRemove(FileNode node) {
    final current = _cachedFiles;
    if (current == null) return;
    final updated = current.where((n) => n.apiPath != node.apiPath).toList();
    setState(() {
      _cachedFiles = updated;
    });
    FileBrowserCache.instance.put(_currentPath, updated);
  }

  /// Immediately add a placeholder folder to the displayed list.
  void _optimisticAddFolder(String folderName) {
    final current = _cachedFiles;
    if (current == null) return;
    final placeholder = FileNode(
      name: folderName,
      size: 0,
      isDir: true,
      deviceName: current.isNotEmpty ? current.first.deviceName : '',
      devicePath: current.isNotEmpty ? current.first.devicePath : '',
      deviceSerial: current.isNotEmpty ? current.first.deviceSerial : '',
      dirPath: _currentPath.isEmpty
          ? '$folderName/'
          : '$_currentPath/$folderName/',
    );
    setState(() {
      _cachedFiles = [...current, placeholder];
    });
  }

  Future<void> _uploadSelectedFiles(
    List<http.MultipartFile> selectedFiles,
    String uploadPath,
  ) async {
    if (_isUploading || selectedFiles.isEmpty) {
      return;
    }

    // Resolve target device serial before starting upload.
    // Prefer the already-loaded _allDevices list to avoid a redundant network
    // call; fall back to StorageService.listDevices() only if the list is
    // empty for some reason (#1022).
    String? targetSerial;
    try {
      final devices =
          (_allDevices.isNotEmpty
                  ? _allDevices
                  : await StorageService.listDevices())
              .where((d) => d.isEnabled)
              .toList();
      if (devices.length > 1) {
        if (!mounted) return;
        final picked = await showDeviceUploadPicker(context, devices);
        if (picked == null) return; // user cancelled
        targetSerial = picked.serial.isNotEmpty ? picked.serial : null;
      } else if (devices.length == 1) {
        targetSerial = devices.first.serial.isNotEmpty
            ? devices.first.serial
            : null;
      }
    } catch (e) {
      debugPrint('[file_browser_page.dart] Failed to list devices: $e');
      // Fall through with null serial (default device)
    }

    setState(() {
      _isUploading = true;
      _uploadTotal = selectedFiles.length;
      _uploadCompleted = 0;
    });

    int failed = 0;
    try {
      for (final file in selectedFiles) {
        try {
          await _controller.uploadFiles(
            currentPath: uploadPath,
            selectedFiles: [file],
            serial: targetSerial,
          );
        } catch (_) {
          failed++;
          debugPrint(
            '[file_browser_page.dart] Failed to upload ${file.filename}',
          );
        }
        if (mounted) setState(() => _uploadCompleted++);
      }

      if (!mounted) return;

      _refreshFileState();

      final succeeded = selectedFiles.length - failed;
      if (failed == 0) {
        final label = selectedFiles.length == 1
            ? selectedFiles.first.filename ?? 'file'
            : '${selectedFiles.length} files';
        _showMessage('Uploaded $label');
      } else {
        _showMessage(
          'Uploaded $succeeded of ${selectedFiles.length} ($failed failed)',
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _isUploading = false;
          _uploadTotal = 0;
          _uploadCompleted = 0;
          _recentFilesSectionKey++;
        });
      }
    }
  }

  Future<void> _handleUploadPressed() async {
    if (_isUploading) {
      return;
    }

    try {
      final selectedFiles = await _controller.pickUploadFiles();
      if (selectedFiles.isEmpty) {
        return;
      }

      await _uploadSelectedFiles(selectedFiles, _currentPath);
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

    final snapshot = _cachedFiles;
    setState(() {
      _isCreatingFolder = true;
    });

    // Optimistically show the new folder immediately
    _optimisticAddFolder(folderName);

    try {
      await _controller.createFolder(
        currentPath: _currentPath,
        folderName: folderName,
      );

      if (!mounted) {
        return;
      }

      _showMessage('Created folder $folderName');
      // Belt-and-suspenders: refresh after a short delay in case the WebSocket
      // event is missed (dropped connection, buffering, etc.).
      Future.delayed(const Duration(seconds: 3), () {
        if (mounted) _refreshFileState();
      });
    } catch (_) {
      debugPrint('[file_browser_page.dart] Error in catch block');
      if (!mounted) {
        return;
      }

      // Roll back optimistic folder
      if (snapshot != null) setState(() => _cachedFiles = snapshot);

      _showMessage('Failed to create folder');
    } finally {
      if (mounted) {
        setState(() {
          _isCreatingFolder = false;
        });
      }
    }
  }

  /// Text-like extensions that the plaintext editor can handle.
  static const _kTextExtensions = {
    '.txt',
    '.md',
    '.json',
    '.yaml',
    '.yml',
    '.xml',
    '.html',
    '.css',
    '.js',
    '.go',
    '.py',
    '.sh',
    '.env',
    '.toml',
    '.ini',
    '.cfg',
    '.conf',
    '.log',
  };

  Future<void> _handleNewFilePressed() async {
    final fileName = await showNewFileDialog(context);
    if (fileName == null || !mounted) return;

    try {
      // Create empty content based on file type.
      final String emptyContent;
      if (fileName.endsWith('.absheet')) {
        emptyContent =
            '{"tabs":[{"name":"Sheet 1","data":{"columns":[],"rows":[]}}]}';
      } else if (fileName.endsWith('.abdoc')) {
        emptyContent = '{"ops":[{"insert":"\\n"}]}';
      } else {
        emptyContent = '';
      }
      final bytes = utf8.encode(emptyContent);

      final file = http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: fileName,
      );

      await FilesService.uploadFilesFromFormData(_currentPath, [file]);

      if (!mounted) return;

      _showMessage('Created $fileName');

      final filePath = _currentPath.isEmpty
          ? fileName
          : '$_currentPath/$fileName';

      // For generic files, only open the plaintext editor for text-like
      // extensions; otherwise just refresh the listing.
      final isKnownType =
          fileName.endsWith('.abdoc') || fileName.endsWith('.absheet');
      if (isKnownType) {
        _refreshFileState();
        _openFileViaRoute(filePath);
      } else {
        final ext = filePath.contains('.')
            ? '.${filePath.split('.').last.toLowerCase()}'
            : '';
        if (_kTextExtensions.contains(ext)) {
          _refreshFileState();
          context.push(AppRoutes.plaintextEditorPath(filePath));
        } else {
          _refreshFileState();
        }
      }
    } catch (e) {
      if (!mounted) return;
      _showMessage('Failed to create file: $e');
    }
  }

  Future<void> _handleFileMenuAction(
    FileNode node,
    FileMenuAction action,
  ) async {
    // When inside an archive, handle download specially via the archive endpoint.
    if (_archiveContext != null && action == FileMenuAction.download) {
      try {
        final archive = _archiveContext!;
        final entryPath = archive.subPath.isEmpty
            ? node.name
            : '${archive.subPath}/${node.name}';
        final bytes = await FilesService.downloadArchiveFileBytes(
          archive.archivePath,
          entryPath,
        );
        if (bytes != null && mounted) {
          await FilesService.saveBytesToFile(bytes, node.name);
          _showMessage('Downloaded ${node.name}');
        }
      } catch (_) {
        if (mounted) _showMessage('Download failed');
      }
      return;
    }

    // Snapshot the pre-mutation cache for rollback on failure
    final snapshot = _cachedFiles;
    try {
      // Delete: confirm → optimistic remove → network call
      if (action == FileMenuAction.delete) {
        final shouldDelete = await confirmDelete(
          context,
          node.name.replaceAll(RegExp(r'/+$'), ''),
        );
        if (!mounted || shouldDelete != true) return;
        _optimisticRemove(node);
        await _controller.deleteNode(node: node);
        if (mounted) _showMessage('Deleted');
        return;
      }

      final outcome = await _controller.handleFileAction(
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
      debugPrint('[file_browser_page.dart] Error in catch block');
      if (!mounted) {
        return;
      }

      // Roll back the optimistic update on failure
      if (snapshot != null) {
        setState(() => _cachedFiles = snapshot);
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

  // ── CSV → .absheet conversion (#1019) ──────────────────────────────────

  Future<void> _handleCsvOpen(FileNode node) async {
    if (!mounted) return;

    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Convert to .absheet?'),
        content: Text(
          'Would you like to convert "${node.name}" to an Quark '
          'spreadsheet (.absheet)?\n\nThe original CSV file will not be '
          'modified or deleted.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Convert'),
          ),
        ],
      ),
    );

    if (confirm != true || !mounted) return;

    try {
      // Download the CSV.
      final bytes = await FilesService.downloadFileBytes(
        node.apiPath,
        serial: serialOrNull(node.deviceSerial),
        fileName: node.name,
      );
      if (!mounted) return;
      if (bytes == null || bytes.isEmpty) {
        _showMessage('Failed to read ${node.name}');
        return;
      }

      // Parse CSV into a DataTable via DataSheetController.
      final csvText = utf8.decode(bytes);
      final table = dt.DataTable([]);
      final controller = DataSheetController.fromTable(table);
      controller.loadFromCsv(csvText);

      // Serialize to .absheet JSON envelope.
      final absheetJson = jsonEncode({
        'tabs': [
          {
            'name': node.name.replaceAll(
              RegExp(r'\.csv$', caseSensitive: false),
              '',
            ),
            'data': table.toJson(),
          },
        ],
      });
      controller.dispose();

      // Derive the new file name and upload it alongside the original.
      final baseName = node.name.replaceAll(
        RegExp(r'\.csv$', caseSensitive: false),
        '',
      );
      final absheetName = '$baseName.absheet';
      final folder = parentPath(node.apiPath);
      final uploadFile = http.MultipartFile.fromBytes(
        'files',
        utf8.encode(absheetJson),
        filename: absheetName,
      );
      await FilesService.uploadFilesFromFormData(
        folder,
        [uploadFile],
        serial: serialOrNull(node.deviceSerial),
        overwrite: true,
      );
      if (!mounted) return;

      // Refresh the file list so the new .absheet appears.
      _refreshFileState();

      // Open the new .absheet through the canonical files route.
      final absheetPath = folder.isEmpty ? absheetName : '$folder/$absheetName';
      _openFileViaRoute(absheetPath);
    } catch (e) {
      if (!mounted) return;
      _showMessage('Conversion failed: $e');
    }
  }

  Future<void> _handleOpenNode(FileNode node) async {
    if (node.isDir) {
      _openDirectory(node);
      return;
    }

    // When inside an archive, preview files inline where possible.
    if (_archiveContext != null) {
      await _openArchiveFile(node);
      return;
    }

    // Navigate into archives as virtual directories.
    if (node.fileType == 'archive') {
      _openArchive(node);
      return;
    }

    final lowerName = node.name.toLowerCase();

    // Quark native document format — open in the rich text editor.
    if (lowerName.endsWith('.abdoc')) {
      _openFileViaRoute(node.apiPath);
      return;
    }

    // Quark native spreadsheet format.
    if (lowerName.endsWith('.absheet')) {
      _openFileViaRoute(node.apiPath);
      return;
    }

    // CSV — offer to convert to .absheet (#1019).
    if (lowerName.endsWith('.csv')) {
      await _handleCsvOpen(node);
      return;
    }

    // Text files — open in the plaintext editor via push so back works.
    final ext = node.name.contains('.')
        ? '.${node.name.split('.').last.toLowerCase()}'
        : '';
    if (_kTextExtensions.contains(ext)) {
      context.push(AppRoutes.plaintextEditorPath(node.apiPath));
      return;
    }

    // Types with no in-app viewer yet — show a detail view with download and
    // "Open with" actions instead of silently failing. Named document types
    // land here too: without them a .pdf ends up worse off than an unclassified
    // file, which reaches this branch as 'generic' (#1184).
    if (usesGenericFileViewer(node.fileType)) {
      Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => GenericFileViewerPage(node: node),
        ),
      );
      return;
    }

    // All other file types — navigate to /files/<path> which resolves the
    // file type via FileViewerPage and opens the correct viewer. This updates
    // the URL bar so the link is always shareable.
    _openFileViaRoute(node.apiPath);
  }

  static const _kImageExtensions = {
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.bmp',
    '.webp',
    '.tiff',
    '.tif',
  };

  Future<void> _openArchiveFile(FileNode node) async {
    final archive = _archiveContext!;
    final entryPath = archive.subPath.isEmpty
        ? node.name
        : '${archive.subPath}/${node.name}';
    final ext = node.name.contains('.')
        ? '.${node.name.split('.').last.toLowerCase()}'
        : '';

    try {
      final bytes = await FilesService.downloadArchiveFileBytes(
        archive.archivePath,
        entryPath,
      );
      if (bytes == null || !mounted) return;

      if (_kImageExtensions.contains(ext)) {
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => ImageViewerPage(bytes: bytes, name: node.name),
          ),
        );
        return;
      }

      if (_kTextExtensions.contains(ext)) {
        final text = utf8.decode(bytes, allowMalformed: true);
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => _ArchiveTextPreview(name: node.name, text: text),
          ),
        );
        return;
      }

      // Fallback: download the file.
      await FilesService.saveBytesToFile(bytes, node.name);
      if (mounted) _showMessage('Downloaded ${node.name}');
    } catch (e) {
      if (mounted) _showMessage('Failed to open file: $e');
    }
  }

  void _openDirectory(FileNode node) {
    if (!node.isDir) {
      return;
    }

    // If we're inside an archive, directory taps descend further into it.
    if (_archiveContext != null) {
      _descendIntoArchiveDir(node);
      return;
    }

    _setPath(
      _controller.nextPathForOpenDirectory(
        currentPath: _currentPath,
        node: node,
      ),
    );
  }

  /// Enter an archive file as a virtual directory.
  void _openArchive(FileNode node) {
    setState(() {
      _archiveContext = _ArchiveContext(
        archivePath: node.apiPath,
        subPath: '',
        archiveSerial: node.deviceSerial,
      );
      _isSearchMode = false;
      _searchFuture = null;
      _searchQuery = null;
      _reloadFiles();
    });
  }

  /// Descend into a subdirectory inside the current archive.
  void _descendIntoArchiveDir(FileNode node) {
    final ctx = _archiveContext!;
    final newSubPath = ctx.subPath.isEmpty
        ? node.name
        : '${ctx.subPath}/${node.name}';
    setState(() {
      _archiveContext = _ArchiveContext(
        archivePath: ctx.archivePath,
        subPath: newSubPath,
        archiveSerial: ctx.archiveSerial,
      );
      _reloadFiles();
    });
  }

  /// Exit the current archive, returning to the real filesystem.
  void _exitArchive() {
    setState(() {
      _archiveContext = null;
      _reloadFiles();
    });
  }

  void _handleSearchChanged(String query) {
    setState(() {
      _isSearchMode = true;
      _searchFuture = FilesService.searchFiles(query);
      _searchQuery = query;
    });
  }

  void _handleSearchClosed() {
    setState(() {
      _isSearchMode = false;
      _searchFuture = null;
      _searchQuery = null;
    });
  }

  void _navigateToFolder(FileNode node) {
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
    final archive = _archiveContext;
    if (archive != null) {
      if (archive.subPath.isEmpty) {
        // At archive root — exit back to the real filesystem.
        _exitArchive();
      } else {
        // Ascend one level inside the archive.
        final parent = archive.subPath.contains('/')
            ? archive.subPath.substring(0, archive.subPath.lastIndexOf('/'))
            : '';
        setState(() {
          _archiveContext = _ArchiveContext(
            archivePath: archive.archivePath,
            subPath: parent,
            archiveSerial: archive.archiveSerial,
          );
          _reloadFiles();
        });
      }
      return;
    }

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
    context.go(AppRoutes.filesPath(normalized));
  }

  void _showMessage(String message) {
    final messenger = ScaffoldMessenger.of(context);
    messenger.clearSnackBars();
    messenger.showSnackBar(SnackBar(content: Text(message)));
  }

  void _goHome() {
    if (_archiveContext != null) {
      _exitArchive();
      return;
    }

    if (_fileBrowserScrollController.hasClients) {
      _fileBrowserScrollController.jumpTo(0);
    }

    if (_currentPath.isEmpty) {
      setState(() {
        _isSearchMode = false;
        _searchFuture = null;
        _searchQuery = null;
        _activeDevicePaths = _allDevices.map((d) => d.devicePath).toSet();
        _reloadFiles();
      });
      return;
    }

    _setPath('');
  }

  Widget _buildRouteResolutionLoadingShell(BuildContext context) {
    final routeLabel = filesRouteDisplayPath(_currentPath);
    final isFileRoute = isLikelyFilePath(_currentPath);

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 420),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(QuarkIcons.folder_open, size: 48),
              const SizedBox(height: 16),
              Text(
                isFileRoute ? 'Opening file' : 'Opening folder',
                style: Theme.of(context).textTheme.titleMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              Text(
                routeLabel,
                style: Theme.of(context).textTheme.bodyMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 20),
              const LinearProgressIndicator(),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _retryRouteFailure(_FilesRouteFailure failure) async {
    if (!mounted) {
      return;
    }

    setState(() {
      _routeFailure = null;
    });

    if (failure.isFileRoute) {
      await _openPendingFile(failure.requestedPath);
      return;
    }

    await _refreshFileState();
  }

  Widget _buildFolderRouteErrorState(BuildContext context, Object error) {
    final requestError = error is FilesRequestException ? error : null;
    final isMissingFolder = requestError?.statusCode == 404;
    final isUnauthorized =
        requestError?.statusCode == 401 || requestError?.statusCode == 403;
    final normalizedPath = normalizePath(_currentPath);
    final routeLabel = normalizedPath.isEmpty
        ? AppRoutes.files
        : AppRoutes.filesPath(normalizedPath);
    final parent = parentPath(normalizedPath);

    return EmptyStateWidget(
      icon: isUnauthorized
          ? QuarkIcons.lock_outline
          : isMissingFolder
          ? QuarkIcons.folder_off_outlined
          : QuarkIcons.error_outline,
      headline: isUnauthorized
          ? 'Access denied'
          : isMissingFolder
          ? 'Folder not found'
          : 'Unable to open folder',
      subtext: isUnauthorized
          ? 'You do not have access to $routeLabel. Retry, move up a level, or return to /files.'
          : isMissingFolder
          ? 'The folder at $routeLabel is unavailable. Retry, move up a level, or return to /files.'
          : 'Files could not load $routeLabel. Retry, move up a level, or return to /files.',
      action: Wrap(
        alignment: WrapAlignment.center,
        spacing: 12,
        runSpacing: 12,
        children: [
          FilledButton(
            onPressed: _refreshFileState,
            child: const Text('Retry'),
          ),
          if (parent.isNotEmpty)
            OutlinedButton(
              onPressed: () => _setPath(parent),
              child: const Text('Go to parent'),
            ),
          OutlinedButton(onPressed: _goHome, child: const Text('Go to /files')),
        ],
      ),
    );
  }

  Widget _buildFileRouteErrorState(
    BuildContext context,
    _FilesRouteFailure failure,
  ) {
    final routeLabel = filesRouteDisplayPath(failure.requestedPath);
    final parent = parentPath(failure.requestedPath);

    return EmptyStateWidget(
      icon: failure.isUnsupported
          ? QuarkIcons.description_outlined
          : QuarkIcons.error_outline,
      headline: failure.isUnsupported
          ? 'No supported editor'
          : failure.isUnauthorized
          ? 'File access denied'
          : 'File not found',
      subtext: failure.isUnsupported
          ? 'No supported editor is available for $routeLabel. Retry, open the containing folder, or return to /files.'
          : failure.isUnauthorized
          ? 'You do not have access to $routeLabel. Retry, open the containing folder, or return to /files.'
          : 'The file at $routeLabel is unavailable. Retry, open the containing folder, or return to /files.',
      action: Wrap(
        alignment: WrapAlignment.center,
        spacing: 12,
        runSpacing: 12,
        children: [
          FilledButton(
            onPressed: () => _retryRouteFailure(failure),
            child: const Text('Retry'),
          ),
          if (parent.isNotEmpty)
            OutlinedButton(
              onPressed: () => _setPath(parent),
              child: const Text('Open containing folder'),
            ),
          OutlinedButton(onPressed: _goHome, child: const Text('Go to /files')),
        ],
      ),
    );
  }

  /// Push a file editor overlay and sync the canonical file route when needed.
  void _openFileViaRoute(String filePath) {
    if (!mounted) {
      return;
    }
    context.go(AppRoutes.filesPath(filePath));
  }

  Future<void> _openEditorWithUrl({
    required String filePath,
    required Widget Function(String targetRoute, String closeRoute) builder,
  }) async {
    FileBrowserCache.instance.markFileOpen(filePath);

    final navigator = Navigator.of(context);
    final targetRoute = AppRoutes.filesPath(filePath);
    // The live location is always percent-encoded, so both sides go through
    // canonicalRoute before comparing. Comparing the raw strings made every
    // name containing a space look like a different route, which fired a
    // spurious mid-push context.go and popped the viewer straight back (#1604).
    final routeBeforeOpen = AppRoutes.canonicalRoute(
      GoRouter.of(context).routeInformationProvider.value.uri.toString(),
    );
    final isAlreadyOnTarget =
        routeBeforeOpen == AppRoutes.canonicalRoute(targetRoute);
    final shouldSyncRoute = !isAlreadyOnTarget;
    final closeRoute = routeBeforeOpen.isEmpty || isAlreadyOnTarget
        ? AppRoutes.filesPath(parentPath(filePath))
        : routeBeforeOpen;
    var routeSyncFailed = false;
    var routeSynced = false;

    void syncRouteOnce() {
      if (!mounted || routeSynced || !shouldSyncRoute) {
        return;
      }
      routeSynced = true;
      try {
        context.go(targetRoute);
      } catch (_) {
        routeSyncFailed = true;
        if (navigator.canPop()) {
          navigator.pop();
        }
        _showMessage('Unable to update the file route');
      }
    }

    try {
      final route = MaterialPageRoute(
        builder: (_) => builder(targetRoute, closeRoute),
      );
      late final AnimationStatusListener statusListener;
      statusListener = (status) {
        if (status == AnimationStatus.completed) {
          route.animation?.removeStatusListener(statusListener);
          syncRouteOnce();
        }
      };

      final pushFuture = navigator.push(route);
      final animation = route.animation;
      if (animation != null) {
        animation.addStatusListener(statusListener);
      } else {
        WidgetsBinding.instance.addPostFrameCallback((_) => syncRouteOnce());
      }

      await pushFuture;
    } finally {
      FileBrowserCache.instance.markFileClosed(filePath);
    }

    if (!mounted) {
      return;
    }

    if (routeSyncFailed) {
      context.go(AppRoutes.filesPath(parentPath(filePath)));
    }
  }

  /// Opens a deep-linked path in the appropriate viewer after mount.
  /// Asks the backend what the path actually is (file vs. directory, and file
  /// type) so that e.g. a folder named "things.abdoc" is opened as a folder
  /// rather than being launched in the document editor.
  Future<void> _openPendingFile(String filePath) async {
    if (!mounted) return;
    _handlingPendingFile = true;
    try {
      await _openPendingFileInner(filePath);
    } finally {
      _handlingPendingFile = false;
    }
  }

  Future<void> _openPendingFileInner(String filePath) async {
    if (!mounted) return;

    // A URL update (SystemNavigator) causes go_router to rebuild this page in
    // the background while the viewer is already on top. Guard against opening
    // a second viewer by checking whether this file is already being shown.
    if (FileBrowserCache.instance.isFileOpen(filePath)) {
      // Navigate to the parent folder so the correct listing is shown when
      // the viewer eventually closes.
      final parent = filePath.contains('/')
          ? filePath.substring(0, filePath.lastIndexOf('/'))
          : '';
      setState(() {
        _currentPath = parent;
        _cachedFiles = FileBrowserCache.instance.get(parent);
        _reloadFiles();
      });
      return;
    }

    // Stat the backend to resolve the real type.
    late final bool isDir;
    late final String fileType;
    late final String fileName;
    try {
      final stat = await FilesService.statFile(filePath);
      isDir = stat.isDir;
      fileType = stat.fileType;
      fileName = stat.name.isEmpty ? filePath.split('/').last : stat.name;
    } on FilesRequestException catch (error) {
      if (!mounted) return;
      if (isLikelyFilePath(filePath)) {
        setState(() {
          _routeFailure = _FilesRouteFailure(
            requestedPath: filePath,
            isFileRoute: true,
            isUnauthorized: error.statusCode == 401 || error.statusCode == 403,
          );
        });
        return;
      }
      _setPath(filePath);
      return;
    } catch (_) {
      if (!mounted) return;
      if (isLikelyFilePath(filePath)) {
        setState(() {
          _routeFailure = _FilesRouteFailure(
            requestedPath: filePath,
            isFileRoute: true,
          );
        });
        return;
      }
      _setPath(filePath);
      return;
    }

    if (!mounted) return;

    if (isDir) {
      _setPath(filePath);
      return;
    }

    // Types with no in-app viewer — download + "Open with…" beats the
    // "No supported editor" dead end these used to hit (#1184). Shared with the
    // click path in _handleOpenNode so both agree.
    if (usesGenericFileViewer(fileType)) {
      final node = FileNode(
        name: filePath.split('/').last,
        size: 0,
        isDir: false,
        deviceName: '',
        devicePath: '',
        deviceSerial: '',
        dirPath: filePath,
        fileType: fileType,
      );
      FileBrowserCache.instance.markFileOpen(filePath);
      try {
        await Navigator.of(context).push<void>(
          MaterialPageRoute<void>(
            builder: (_) => GenericFileViewerPage(node: node),
          ),
        );
      } finally {
        // Without this the marker leaked and both navigation guards then
        // refused to open any file page until restart (#1604).
        FileBrowserCache.instance.markFileClosed(filePath);
      }
      if (!mounted) return;
      // Return the URL to the parent folder, as every other branch does — the
      // route otherwise stays pointed at the file, so reopening it is a no-op.
      context.go(AppRoutes.filesPath(parentPath(filePath)));
      return;
    }

    switch (fileType) {
      case 'abdoc':
        await _openEditorWithUrl(
          filePath: filePath,
          builder: (targetRoute, closeRoute) => DocumentEditorPage(
            filePath: filePath,
            overlayTargetRoute: targetRoute,
            overlayCloseRoute: closeRoute,
          ),
        );
        if (!mounted) return;
        return;

      case 'absheet':
        await _openEditorWithUrl(
          filePath: filePath,
          builder: (targetRoute, closeRoute) => SpreadsheetEditorPage(
            filePath: filePath,
            overlayTargetRoute: targetRoute,
            overlayCloseRoute: closeRoute,
          ),
        );
        if (!mounted) return;
        return;

      case 'code':
      // Source/config files classified by the backend as 'code' open in
      // the same plaintext editor for now — no syntax highlighting yet.
      case 'text':
        // Matches what clicking the row does — without this a deep link to a
        // file the browser opens happily reports "No supported editor".
        FileBrowserCache.instance.markFileOpen(filePath);
        try {
          await context.push<void>(AppRoutes.plaintextEditorPath(filePath));
        } finally {
          FileBrowserCache.instance.markFileClosed(filePath);
        }
        if (!mounted) return;
        context.go(AppRoutes.filesPath(parentPath(filePath)));
        return;

      case 'image':
        final serials = _serialsForActiveDevices();
        final serial = serials.isNotEmpty ? serials.first : null;
        final bytes = await FilesService.downloadFileBytes(
          filePath,
          serial: serial,
        );
        if (!mounted) return;
        if (bytes == null) {
          setState(() {
            _routeFailure = _FilesRouteFailure(
              requestedPath: filePath,
              isFileRoute: true,
            );
          });
          return;
        }
        await _openEditorWithUrl(
          filePath: filePath,
          builder: (_, _) => ImageViewerPage(
            bytes: bytes,
            name: fileName,
            relPath: filePath,
            serial: serial,
          ),
        );
        if (!mounted) return;
        context.go(AppRoutes.filesPath(parentPath(filePath)));
        return;

      case 'video':
      case 'audio':
        final videoSerials = _serialsForActiveDevices();
        final videoSerial = videoSerials.isNotEmpty ? videoSerials.first : null;
        final url = FilesService.constructMediaUrl(
          filePath,
          serial: videoSerial,
        );
        await _openEditorWithUrl(
          filePath: filePath,
          builder: (_, _) => VideoViewerPage(url: url, name: fileName),
        );
        if (!mounted) return;
        context.go(AppRoutes.filesPath(parentPath(filePath)));
        return;

      default:
        setState(() {
          _routeFailure = _FilesRouteFailure(
            requestedPath: filePath,
            isFileRoute: true,
            isUnsupported: !hasSupportedFilesEditorForType(fileType),
          );
        });
        break;
    }
  }

  // ── Mobile FAB (Create actions) ──────────────────────────────────────────

  void _onScroll() {
    if (!_fileBrowserScrollController.hasClients) return;
    final current = _fileBrowserScrollController.offset;
    final delta = current - _lastScrollOffset;
    _lastScrollOffset = current;

    // Always restore FAB when the user is at the very top.
    if (current <= 0) {
      if (!_fabVisible) setState(() => _fabVisible = true);
      return;
    }

    if (delta > 4 && _fabVisible) {
      setState(() => _fabVisible = false);
    } else if (delta < -4 && !_fabVisible) {
      setState(() => _fabVisible = true);
    }
  }

  Widget _buildCreateFab(BuildContext context) {
    return AnimatedOpacity(
      opacity: _fabVisible ? 1.0 : 0.0,
      duration: const Duration(milliseconds: 220),
      curve: Curves.easeInOut,
      child: IgnorePointer(
        ignoring: !_fabVisible,
        child: FloatingActionButton(
          heroTag: 'create_fab',
          onPressed: _showCreateBottomSheet,
          tooltip: 'Create',
          child: const Icon(QuarkIcons.add_rounded),
        ),
      ),
    );
  }

  void _showCreateBottomSheet() {
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                leading: _isUploading
                    ? const SizedBox(
                        width: 24,
                        height: 24,
                        child: CircularProgressIndicator.adaptive(
                          strokeWidth: 2,
                        ),
                      )
                    : const Icon(QuarkIcons.upload_rounded),
                title: Text(
                  _isUploading
                      ? (_uploadTotal > 0
                            ? 'Uploading $_uploadCompleted/$_uploadTotal…'
                            : 'Uploading…')
                      : 'Upload files',
                ),
                enabled: !_isUploading,
                onTap: _isUploading
                    ? null
                    : () {
                        Navigator.of(ctx).pop();
                        _handleUploadPressed();
                      },
              ),
              ListTile(
                leading: const Icon(QuarkIcons.create_new_folder_outlined),
                title: const Text('New folder'),
                enabled: !_isCreatingFolder,
                onTap: _isCreatingFolder
                    ? null
                    : () {
                        Navigator.of(ctx).pop();
                        _handleCreateFolderPressed();
                      },
              ),
              ListTile(
                leading: const Icon(QuarkIcons.edit_document),
                title: const Text('New file'),
                onTap: () {
                  Navigator.of(ctx).pop();
                  _handleNewFilePressed();
                },
              ),
            ],
          ),
        );
      },
    );
  }

  /// Builds the horizontal device filter chip row for issue #801.
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      drawer: QuarkDrawer(
        activeSection: QuarkDrawerSection.files,
        onTapFiles: () {
          Navigator.of(context).pop();
        },
        onTapPhotos: () {
          context.go(AppRoutes.photos);
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
      body: Column(
        children: [
          Builder(
            builder: (context) {
              // When inside an archive, show the archive path as the breadcrumb.
              final archive = _archiveContext;
              final displayPath = archive != null
                  ? (archive.subPath.isEmpty
                        ? archive.archivePath
                        : '${archive.archivePath}/${archive.subPath}')
                  : _currentPath;
              final disableNavigation =
                  _handlingPendingFile && isLikelyFilePath(_currentPath);
              if (_selectionMode) {
                return _SelectionBar(
                  selectedCount: _selectedPaths.length,
                  totalCount: _allCurrentFiles.length,
                  onSelectAll: _selectAll,
                  onDeselectAll: () => setState(() => _selectedPaths.clear()),
                  onCancel: _exitSelectionMode,
                  onDelete: _selectedPaths.isNotEmpty ? _deleteSelected : null,
                );
              }
              return FileTopBar(
                currentPath: displayPath,
                isGridView: _isGridView,
                isSearchMode: _isSearchMode,
                isUploading: _isUploading,
                isCreatingFolder: _isCreatingFolder,
                disableNavigation: disableNavigation,
                isRefreshing: isRefreshing,
                onGoHome: archive != null ? _exitArchive : _goHome,
                onGoUp: _goUpOneLevel,
                onPathSelected: archive != null ? null : _setPath,
                isUnifiedView: _isUnifiedView,
                onToggleView: () => setState(() => _isGridView = !_isGridView),
                onToggleUnifiedView: () =>
                    setState(() => _isUnifiedView = !_isUnifiedView),
                onSearchChanged: _handleSearchChanged,
                onSearchClosed: _handleSearchClosed,
                onRefresh: _refreshFileState,
                onUploadPressed: _handleUploadPressed,
                onCreateFolderPressed: _handleCreateFolderPressed,
                onNewFilePressed: _handleNewFilePressed,
                uploadTotal: _uploadTotal,
                uploadCompleted: _uploadCompleted,
                onOpenDrawer: () => Scaffold.of(context).openDrawer(),
                onOpenSettings: () => context.go(AppRoutes.settings),
                devices: _allDevices.length > 1 ? _allDevices : null,
                activeDevicePaths: _activeDevicePaths,
                onDeviceToggled: (devicePath) {
                  setState(() {
                    if (_activeDevicePaths.contains(devicePath)) {
                      _activeDevicePaths.remove(devicePath);
                      if (_activeDevicePaths.isEmpty) {
                        _activeDevicePaths = _allDevices
                            .map((d) => d.devicePath)
                            .toSet();
                      }
                    } else {
                      _activeDevicePaths.add(devicePath);
                    }
                    _reloadFiles();
                  });
                },
              );
            },
          ),
          FileBrowserHeader(
            isGridView: _isGridView,
            isSearchMode: _isSearchMode,
            filesFuture: _isSearchMode
                ? (_searchFuture ?? Future.value(const <FileNode>[]))
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

          // Hide Recent Files on mobile — show only on tablet/desktop (#959).
          if (!_isSearchMode &&
              _currentPath.isEmpty &&
              !_noHostSelected &&
              MediaQuery.sizeOf(context).width >= 600)
            RecentFilesSection(
              key: ValueKey(_recentFilesSectionKey),
              onOpenFile: _handleOpenNode,
              onFileMenuAction: _handleFileMenuAction,
              onNavigateToFolder: _setPath,
            ),

          Expanded(
            child: _noHostSelected
                ? _FirstRunSetup(
                    onConnected: () {
                      setState(() {
                        _noHostSelected =
                            AppSettings.instance.activeHost == null;
                        if (!_noHostSelected) _reloadFiles();
                      });
                    },
                  )
                : _routeFailure != null
                ? _buildFileRouteErrorState(context, _routeFailure!)
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
                                    Future.value(const <FileNode>[]))
                              : _filesFuture,
                          initialData: _isSearchMode ? null : _cachedFiles,
                          isInitialLoad: isInitialLoad,
                          onFileMenuAction: _handleFileMenuAction,
                          onOpenDirectory: _handleOpenNode,
                          isGridView: _isGridView,
                          isUnifiedView: _isUnifiedView,
                          isSearchMode: _isSearchMode,
                          onNavigateToFolder: _navigateToFolder,
                          currentPath: _currentPath,
                          errorBuilder: _buildFolderRouteErrorState,
                          loadingBuilder: _currentPath.isNotEmpty
                              ? _buildRouteResolutionLoadingShell
                              : null,
                          onDropToFolder: _handleDropToFolder,
                          onFolderDragEnter: _handleFolderDragEnter,
                          onFolderDragExit: _handleFolderDragExit,
                          scrollController: _fileBrowserScrollController,
                          inArchive: _archiveContext != null,
                          selectionMode: _selectionMode,
                          selectedPaths: _selectedPaths,
                          onSelectionChanged: _onSelectionChanged,
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
                        // Mobile create FAB — inside the file-list Stack so it
                        // sits above the footer rather than over the whole page.
                        if (MediaQuery.of(context).size.width < 860)
                          Positioned(
                            right: 16,
                            bottom: 16,
                            child: _buildCreateFab(context),
                          ),
                      ],
                    ),
                  ),
          ),
          if (!_noHostSelected) const FileStorageFooter(),
        ],
      ),
    );
  }
}

class _FirstRunSetup extends StatefulWidget {
  const _FirstRunSetup({required this.onConnected});

  final VoidCallback onConnected;

  @override
  State<_FirstRunSetup> createState() => _FirstRunSetupState();
}

class _FilesRouteFailure {
  const _FilesRouteFailure({
    required this.requestedPath,
    required this.isFileRoute,
    this.isUnauthorized = false,
    this.isUnsupported = false,
  });

  final String requestedPath;
  final bool isFileRoute;
  final bool isUnauthorized;
  final bool isUnsupported;
}

class _FirstRunSetupState extends State<_FirstRunSetup> {
  final _controller = TextEditingController();
  bool _saving = false;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _connect() async {
    final raw = _controller.text.trim();
    if (raw.isEmpty) {
      setState(() => _error = 'Please enter your Quark address.');
      return;
    }

    var address = raw;
    if (!address.startsWith('http://') && !address.startsWith('https://')) {
      address = 'http://$address';
    }

    setState(() {
      _saving = true;
      _error = null;
    });

    try {
      await AppSettings.instance.addHost(
        HostEntry(name: 'My Quark', hostAddress: address),
      );
      if (mounted) widget.onConnected();
    } catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = 'Could not connect. Check the address and try again.';
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 400),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Icon(
                QuarkIcons.storage_outlined,
                size: 56,
                color: Colors.grey,
              ),
              const SizedBox(height: 16),
              Text(
                'Connect to your Quark',
                textAlign: TextAlign.center,
                style: Theme.of(
                  context,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 8),
              Text(
                'Enter the address of your Quark device on your home network.',
                textAlign: TextAlign.center,
                style: Theme.of(
                  context,
                ).textTheme.bodyMedium?.copyWith(color: Colors.grey),
              ),
              const SizedBox(height: 24),
              TextField(
                controller: _controller,
                autofocus: true,
                keyboardType: TextInputType.url,
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => _connect(),
                decoration: InputDecoration(
                  labelText: 'Quark address',
                  hintText: 'http://quark.home.local',
                  helperText:
                      'Usually http://quark.home.local or http://192.168.x.x',
                  errorText: _error,
                  border: const OutlineInputBorder(),
                  prefixIcon: const Icon(QuarkIcons.link_rounded),
                ),
              ),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: _saving ? null : _connect,
                child: _saving
                    ? const SizedBox(
                        height: 20,
                        width: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Connect'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Tracks the state when the user has navigated inside an archive file.
class _ArchiveContext {
  const _ArchiveContext({
    required this.archivePath,
    required this.subPath,
    required this.archiveSerial,
  });

  /// Path to the archive file, relative to the device files directory.
  final String archivePath;

  /// Current virtual subdirectory inside the archive. Empty string = root.
  final String subPath;

  /// Device serial of the device that holds the archive.
  final String archiveSerial;
}

class _ArchiveTextPreview extends StatelessWidget {
  const _ArchiveTextPreview({required this.name, required this.text});
  final String name;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(name)),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: SelectableText(
          text,
          style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
        ),
      ),
    );
  }
}

/// Top bar shown in place of [FileTopBar] while multi-select mode is active.
class _SelectionBar extends StatelessWidget {
  const _SelectionBar({
    required this.selectedCount,
    required this.totalCount,
    required this.onSelectAll,
    required this.onDeselectAll,
    required this.onCancel,
    this.onDelete,
  });

  final int selectedCount;
  final int totalCount;
  final VoidCallback onSelectAll;
  final VoidCallback onDeselectAll;
  final VoidCallback onCancel;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Container(
      height: 56,
      color: colors.surfaceContainerHighest,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: Row(
        children: [
          IconButton(
            icon: const Icon(Icons.close),
            tooltip: 'Cancel selection',
            onPressed: onCancel,
          ),
          Text(
            '$selectedCount selected',
            style: const TextStyle(fontWeight: FontWeight.w600),
          ),
          const Spacer(),
          TextButton(
            onPressed: selectedCount < totalCount ? onSelectAll : onDeselectAll,
            child: Text(
              selectedCount < totalCount ? 'Select all' : 'Deselect all',
            ),
          ),
          const SizedBox(width: 4),
          IconButton(
            icon: Icon(
              Icons.delete_outline,
              color: onDelete != null
                  ? colors.error
                  : colors.onSurface.withValues(alpha: 0.38),
            ),
            tooltip: 'Delete selected',
            onPressed: onDelete,
          ),
        ],
      ),
    );
  }
}
