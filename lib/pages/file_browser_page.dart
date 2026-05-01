import 'dart:async';
import 'dart:convert';

import 'package:autobutler/controllers/file_browser_cache.dart';
import 'package:autobutler/controllers/file_browser_controller.dart';
import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/pages/document_editor_page.dart';
import 'package:autobutler/pages/spreadsheet_editor_page.dart';
import 'package:data_table/data_sheet.dart';
import 'package:data_table/data_table.dart' as dt;
import 'package:autobutler/router.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/services/events_service.dart';
import 'package:autobutler/services/storage_service.dart';
import 'package:autobutler/utils/auto_refresh_mixin.dart';
import 'package:autobutler/utils/cirrus_route_path_utils.dart';
import 'package:autobutler/utils/file_browser_dialog_utils.dart';
import 'package:autobutler/utils/file_browser_drag_config.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:autobutler/utils/safe_set_state_mixin.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:autobutler/widgets/core/empty_state_widget.dart';
import 'package:autobutler/widgets/device_upload_picker.dart';
import 'package:autobutler/widgets/file_browser/file_browser_header.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:autobutler/widgets/file_browser/file_storage_footer.dart';
import 'package:autobutler/widgets/file_browser/file_top_bar.dart';
import 'package:autobutler/widgets/file_browser/new_file_dialog.dart';
import 'package:autobutler/widgets/file_browser/recent_files_section.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;

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

  Future<List<CirrusFileNode>> _filesFuture = Future.value(
    const <CirrusFileNode>[],
  );
  List<CirrusFileNode>?
  _cachedFiles; // last successful result, shown during refresh
  int _generation = 0; // incremented on each reload to discard stale fetches
  String _currentPath = '';

  /// If the deep-link URL pointed directly to a file, open its editor once
  /// the page has mounted. Only consumed once.
  String? _pendingFileOpen;

  _CirrusRouteFailure? _routeFailure;
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
  Future<List<CirrusFileNode>>? _searchFuture;
  String? _searchQuery;

  // Archive browser state — non-null when navigating inside an archive.
  _ArchiveContext? _archiveContext;

  @override
  void initState() {
    // Apply deep-link initial path before AutoRefreshMixin triggers the first load.
    final initial = widget.initialPath;
    if (initial != null && initial.isNotEmpty) {
      final normalizedInitial = normalizePath(initial);
      // Optimistically treat the deep-link path as a directory so that
      // _reloadFiles() fetches the right folder contents immediately (no root
      // flash). _openPendingFile will stat the backend and correct course if
      // the path turns out to be a file.
      _currentPath = normalizedInitial;
      _pendingFileOpen = normalizedInitial;
      // Show cached listing instantly while the fresh fetch is in flight.
      _cachedFiles = FileBrowserCache.instance.get(normalizedInitial);
    }
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
  Future<void> refresh() async {
    _noHostSelected = AppSettings.instance.activeHost == null;
    if (_noHostSelected) {
      setState(() {
        _filesFuture = Future.value(const <CirrusFileNode>[]);
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
      _filesFuture = Future.value(const <CirrusFileNode>[]);
      return;
    }

    final generation = ++_generation;
    final serials = _serialsForActiveDevices();
    final archive = _archiveContext;

    Future<List<CirrusFileNode>> fetchFuture;
    if (archive != null) {
      fetchFuture = CirrusService.listArchiveEntries(
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
        setState(() => _cachedFiles = files);
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

  // ── Optimistic updates ───────────────────────────────────────────────────

  /// Immediately remove a node from the displayed list.
  /// If the server call fails, [_refreshFileState] will reconcile.
  void _optimisticRemove(CirrusFileNode node) {
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
    final placeholder = CirrusFileNode(
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

  Future<void> _handleNewFilePressed() async {
    final fileName = await showNewFileDialog(context);
    if (fileName == null || !mounted) return;

    try {
      // Create empty content based on file type.
      final String emptyContent;
      if (fileName.endsWith('.absheet')) {
        emptyContent =
            '{"tabs":[{"name":"Sheet 1","data":{"columns":[],"rows":[]}}]}';
      } else {
        emptyContent = '{"ops":[{"insert":"\\n"}]}';
      }
      final bytes = emptyContent.codeUnits;

      final file = http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: fileName,
      );

      await CirrusService.uploadFilesFromFormData(_currentPath, [file]);

      if (!mounted) return;

      _showMessage('Created $fileName');

      // Refresh file list then open the editor.
      _refreshFileState();

      final filePath = _currentPath.isEmpty
          ? fileName
          : '$_currentPath/$fileName';

      if (fileName.endsWith('.absheet')) {
        await _openEditorWithUrl(
          filePath: filePath,
          builder: () => SpreadsheetEditorPage(filePath: filePath),
        );
      } else {
        await _openEditorWithUrl(
          filePath: filePath,
          builder: () => DocumentEditorPage(filePath: filePath),
        );
      }
    } catch (e) {
      if (!mounted) return;
      _showMessage('Failed to create file: $e');
    }
  }

  Future<void> _handleFileMenuAction(
    CirrusFileNode node,
    FileMenuAction action,
  ) async {
    // When inside an archive, handle download specially via the archive endpoint.
    if (_archiveContext != null && action == FileMenuAction.download) {
      try {
        final archive = _archiveContext!;
        final entryPath = archive.subPath.isEmpty
            ? node.name
            : '${archive.subPath}/${node.name}';
        final bytes = await CirrusService.downloadArchiveFileBytes(
          archive.archivePath,
          entryPath,
        );
        if (bytes != null && mounted) {
          await CirrusService.saveBytesToFile(bytes, node.name);
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

  Future<void> _handleCsvOpen(CirrusFileNode node) async {
    if (!mounted) return;

    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Convert to .absheet?'),
        content: Text(
          'Would you like to convert "${node.name}" to an AutoButler '
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
      final bytes = await CirrusService.downloadFileBytes(
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

      // Serialise to .absheet JSON envelope.
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
      await CirrusService.uploadFilesFromFormData(
        folder,
        [uploadFile],
        serial: serialOrNull(node.deviceSerial),
        overwrite: true,
      );
      if (!mounted) return;

      // Refresh the file list so the new .absheet appears.
      _refreshFileState();

      // Open the new .absheet in the spreadsheet editor.
      final absheetPath = folder.isEmpty ? absheetName : '$folder/$absheetName';
      await _openEditorWithUrl(
        filePath: absheetPath,
        builder: () => SpreadsheetEditorPage(
          filePath: absheetPath,
          deviceSerial: node.deviceSerial,
        ),
      );
    } catch (e) {
      if (!mounted) return;
      _showMessage('Conversion failed: $e');
    }
  }

  Future<void> _handleOpenNode(CirrusFileNode node) async {
    if (node.isDir) {
      _openDirectory(node);
      return;
    }

    // Navigate into archives as virtual directories.
    if (node.fileType == 'archive') {
      _openArchive(node);
      return;
    }

    final lowerName = node.name.toLowerCase();

    // AutoButler native document format — open in the rich text editor.
    if (lowerName.endsWith('.abdoc')) {
      await _openEditorWithUrl(
        filePath: node.apiPath,
        builder: () => DocumentEditorPage(
          filePath: node.apiPath,
          deviceSerial: node.deviceSerial,
        ),
      );
      return;
    }

    // AutoButler native spreadsheet format.
    if (lowerName.endsWith('.absheet')) {
      await _openEditorWithUrl(
        filePath: node.apiPath,
        builder: () => SpreadsheetEditorPage(
          filePath: node.apiPath,
          deviceSerial: node.deviceSerial,
        ),
      );
      return;
    }

    // CSV — offer to convert to .absheet (#1019).
    if (lowerName.endsWith('.csv')) {
      await _handleCsvOpen(node);
      return;
    }

    _showMessage('No supported editor is available for ${node.name}');
  }

  void _openDirectory(CirrusFileNode node) {
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
  void _openArchive(CirrusFileNode node) {
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
  void _descendIntoArchiveDir(CirrusFileNode node) {
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
      _searchFuture = CirrusService.searchFiles(query);
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
    context.go(AppRoutes.cirrusPath(normalized));
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
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
    final routeLabel = cirrusRouteDisplayPath(_currentPath);
    final isFileRoute = isLikelyFilePath(_currentPath);

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 420),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.folder_open, size: 48),
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

  Future<void> _retryRouteFailure(_CirrusRouteFailure failure) async {
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
    final requestError = error is CirrusRequestException ? error : null;
    final isMissingFolder = requestError?.statusCode == 404;
    final isUnauthorized =
        requestError?.statusCode == 401 || requestError?.statusCode == 403;
    final normalizedPath = normalizePath(_currentPath);
    final routeLabel = normalizedPath.isEmpty
        ? AppRoutes.cirrus
        : AppRoutes.cirrusPath(normalizedPath);
    final parent = parentPath(normalizedPath);

    return EmptyStateWidget(
      icon: isUnauthorized
          ? Icons.lock_outline
          : isMissingFolder
          ? Icons.folder_off_outlined
          : Icons.error_outline,
      headline: isUnauthorized
          ? 'Access denied'
          : isMissingFolder
          ? 'Folder not found'
          : 'Unable to open folder',
      subtext: isUnauthorized
          ? 'You do not have access to $routeLabel. Retry, move up a level, or return to /cirrus.'
          : isMissingFolder
          ? 'The folder at $routeLabel is unavailable. Retry, move up a level, or return to /cirrus.'
          : 'Cirrus could not load $routeLabel. Retry, move up a level, or return to /cirrus.',
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
          OutlinedButton(
            onPressed: _goHome,
            child: const Text('Go to /cirrus'),
          ),
        ],
      ),
    );
  }

  Widget _buildFileRouteErrorState(
    BuildContext context,
    _CirrusRouteFailure failure,
  ) {
    final routeLabel = cirrusRouteDisplayPath(failure.requestedPath);
    final parent = parentPath(failure.requestedPath);

    return EmptyStateWidget(
      icon: failure.isUnsupported
          ? Icons.description_outlined
          : Icons.error_outline,
      headline: failure.isUnsupported
          ? 'No supported editor'
          : failure.isUnauthorized
          ? 'File access denied'
          : 'File not found',
      subtext: failure.isUnsupported
          ? 'No supported editor is available for $routeLabel. Retry, open the containing folder, or return to /cirrus.'
          : failure.isUnauthorized
          ? 'You do not have access to $routeLabel. Retry, open the containing folder, or return to /cirrus.'
          : 'The file at $routeLabel is unavailable. Retry, open the containing folder, or return to /cirrus.',
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
          OutlinedButton(
            onPressed: _goHome,
            child: const Text('Go to /cirrus'),
          ),
        ],
      ),
    );
  }

  /// Push a file editor overlay, then update the route through go_router so
  /// refresh/bookmark/history all reuse the same direct file deep-link flow.
  Future<void> _openEditorWithUrl({
    required String filePath,
    required Widget Function() builder,
  }) async {
    FileBrowserCache.instance.markFileOpen(filePath);

    final navigator = Navigator.of(context);
    final router = GoRouter.of(context);
    final targetRoute = AppRoutes.cirrusPath(filePath);
    final routeBeforeOpen = router.routeInformationProvider.value.uri
        ?.toString();
    final closeRoute =
        routeBeforeOpen == null ||
            routeBeforeOpen.isEmpty ||
            routeBeforeOpen == targetRoute
        ? AppRoutes.cirrusPath(parentPath(filePath))
        : routeBeforeOpen;
    var routeSyncFailed = false;
    var routeMovedWhileOpen = false;

    void closeIfRouteMoved() {
      final currentRoute =
          router.routeInformationProvider.value.uri?.toString() ?? '';
      if (currentRoute != targetRoute && navigator.canPop()) {
        routeMovedWhileOpen = true;
        navigator.pop();
      }
    }

    router.routeInformationProvider.addListener(closeIfRouteMoved);
    try {
      final pushFuture = navigator.push(
        MaterialPageRoute(builder: (_) => builder()),
      );

      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        try {
          context.go(targetRoute);
        } catch (_) {
          routeSyncFailed = true;
          if (navigator.canPop()) {
            navigator.pop();
          }
          _showMessage('Unable to update the file route');
        }
      });

      await pushFuture;
    } finally {
      router.routeInformationProvider.removeListener(closeIfRouteMoved);
      FileBrowserCache.instance.markFileClosed();
    }

    if (!mounted) {
      return;
    }

    if (routeSyncFailed) {
      context.go(AppRoutes.cirrusPath(parentPath(filePath)));
      return;
    }

    final currentRoute =
        router.routeInformationProvider.value.uri?.toString() ?? '';
    if (!routeMovedWhileOpen && currentRoute == targetRoute) {
      context.go(closeRoute);
    }
  }

  /// Opens a deep-linked path in the appropriate viewer after mount.
  /// Asks the backend what the path actually is (file vs. directory, and file
  /// type) so that e.g. a folder named "things.abdoc" is opened as a folder
  /// rather than being launched in the document editor.
  Future<void> _openPendingFile(String filePath) async {
    if (!mounted) return;
    await _openPendingFileInner(filePath);
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
    try {
      final stat = await CirrusService.statFile(filePath);
      isDir = stat.isDir;
      fileType = stat.fileType;
    } on CirrusRequestException catch (error) {
      if (!mounted) return;
      if (isLikelyFilePath(filePath)) {
        setState(() {
          _routeFailure = _CirrusRouteFailure(
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
          _routeFailure = _CirrusRouteFailure(
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

    switch (fileType) {
      case 'abdoc':
        await _openEditorWithUrl(
          filePath: filePath,
          builder: () => DocumentEditorPage(filePath: filePath),
        );
        if (!mounted) return;
        return;

      case 'absheet':
        await _openEditorWithUrl(
          filePath: filePath,
          builder: () => SpreadsheetEditorPage(filePath: filePath),
        );
        if (!mounted) return;
        return;
      default:
        setState(() {
          _routeFailure = _CirrusRouteFailure(
            requestedPath: filePath,
            isFileRoute: true,
            isUnsupported: !hasSupportedCirrusEditorForType(fileType),
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
          child: const Icon(Icons.add_rounded),
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
                    : const Icon(Icons.upload_rounded),
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
                leading: const Icon(Icons.create_new_folder_outlined),
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
                leading: const Icon(Icons.edit_document),
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
      drawer: AutobutlerDrawer(
        activeSection: AutobutlerDrawerSection.cirrus,
        onTapCirrus: () {
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
              return FileTopBar(
                currentPath: displayPath,
                isGridView: _isGridView,
                isSearchMode: _isSearchMode,
                isUploading: _isUploading,
                isCreatingFolder: _isCreatingFolder,
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
                                    Future.value(const <CirrusFileNode>[]))
                              : _filesFuture,
                          initialData: _isSearchMode ? null : _cachedFiles,
                          isInitialLoad: isInitialLoad,
                          onFileMenuAction: _handleFileMenuAction,
                          onOpenDirectory: _isSearchMode
                              ? (_) {}
                              : _handleOpenNode,
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

class _CirrusRouteFailure {
  const _CirrusRouteFailure({
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
      setState(() => _error = 'Please enter your AutoButler address.');
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
        HostEntry(name: 'My AutoButler', hostAddress: address),
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
              const Icon(Icons.storage_outlined, size: 56, color: Colors.grey),
              const SizedBox(height: 16),
              Text(
                'Connect to your AutoButler',
                textAlign: TextAlign.center,
                style: Theme.of(
                  context,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 8),
              Text(
                'Enter the address of your AutoButler device on your home network.',
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
                  labelText: 'AutoButler address',
                  hintText: 'http://autobutler.home.local',
                  helperText:
                      'Usually http://autobutler.home.local or http://192.168.x.x',
                  errorText: _error,
                  border: const OutlineInputBorder(),
                  prefixIcon: const Icon(Icons.link_rounded),
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

  /// Path to the archive file, relative to the device cirrus directory.
  final String archivePath;

  /// Current virtual subdirectory inside the archive. Empty string = root.
  final String subPath;

  /// Device serial of the device that holds the archive.
  final String archiveSerial;
}
