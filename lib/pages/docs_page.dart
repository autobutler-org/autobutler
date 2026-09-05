import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:quark/controllers/file_type_listing_cache.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/router.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/services/content_search_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/file_browser_dialog_utils.dart';
import 'package:quark/utils/safe_set_state_mixin.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';
import 'package:quark/widgets/docs/docs_body.dart';
import 'package:quark/widgets/docs/docs_search_bar.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';

class DocsPage extends StatefulWidget {
  const DocsPage({super.key});

  @override
  State<DocsPage> createState() => _DocsPageState();
}

class _DocsPageState extends State<DocsPage> with SafeSetStateMixin {
  static const _fileType = 'qdoc';

  List<FileNode> _files = [];
  List<FileNode> _filtered = [];
  List<ContentSearchResult> _contentResults = [];
  bool _contentSearching = false;
  bool _loading = true;

  /// The thrown object, not its message — the render decides whether it means
  /// "your Quark is unreachable" or "the request failed" (#1637).
  Object? _error;
  final _searchController = TextEditingController();
  Timer? _contentSearchDebounce;

  @override
  void initState() {
    super.initState();
    final cached = FileTypeListingCache.instance.get(_fileType);
    if (cached != null) {
      _files = cached;
      _filtered = List.of(cached);
      _loading = false;
    }
    _searchController.addListener(_onSearchChanged);
    _load();
  }

  @override
  void dispose() {
    _contentSearchDebounce?.cancel();
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final cached = FileTypeListingCache.instance.get(_fileType);
    setStateSafely(() {
      _loading = cached == null && _files.isEmpty;
      _error = null;
    });
    try {
      final files = await FilesService.getFilesByType(_fileType);
      FileTypeListingCache.instance.put(_fileType, files);
      setStateSafely(() {
        _files = files;
        _applyFilter();
        _loading = false;
      });
    } catch (e) {
      setStateSafely(() {
        if (_files.isEmpty) _error = e;
        _loading = false;
      });
      if (_files.isNotEmpty && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(Errors.message(e, 'refresh your documents'))),
        );
      }
    }
  }

  void _onSearchChanged() {
    _applyFilter();
    _contentSearchDebounce?.cancel();
    final q = _searchController.text.trim();
    if (q.isEmpty) {
      setStateSafely(() {
        _contentResults = [];
        _contentSearching = false;
      });
      return;
    }
    final recent = ContentSearchService.recent(q);
    if (recent != null) {
      setStateSafely(() {
        _contentResults = recent;
        _contentSearching = false;
      });
      return;
    }
    setStateSafely(() => _contentSearching = true);
    _contentSearchDebounce = Timer(const Duration(milliseconds: 400), () async {
      final results = await ContentSearchService.search(q);
      setStateSafely(() {
        _contentResults = results;
        _contentSearching = false;
      });
    });
  }

  void _applyFilter() {
    final query = _searchController.text.trim().toLowerCase();
    setStateSafely(() {
      _filtered = query.isEmpty
          ? List.of(_files)
          : _files
                .where(
                  (f) =>
                      f.name.toLowerCase().contains(query) ||
                      f.dirPath.toLowerCase().contains(query) ||
                      f.deviceName.toLowerCase().contains(query),
                )
                .toList();
    });
  }

  Future<void> _openDoc(FileNode node) async {
    await context.push(
      AppRoutes.docFile(node.apiPath, serial: node.deviceSerial),
    );
    // Refresh in case the doc was renamed or deleted.
    _load();
  }

  Future<void> _createNewDoc() async {
    final name = await promptForNewFileName(
      context,
      title: 'New Document',
      hintText: 'Document name',
    );
    if (name == null || name.isEmpty || !mounted) return;

    try {
      final fileName = name.endsWith('.qdoc') ? name : '$name.qdoc';
      final bytes = '{"ops":[{"insert":"\\n"}]}'.codeUnits;
      final file = http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: fileName,
      );
      await FilesService.uploadFilesFromFormData('', [file]);
      if (!mounted) return;
      context.push(AppRoutes.docFile(fileName));
      _load();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(Errors.message(e, 'create the document'))),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: QuarkAppBar(
        label: 'Docs',
        icon: QuarkIcons.description_outlined,
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: 'New document',
            onPressed: _createNewDoc,
          ),
          IconButton(
            icon: const Icon(QuarkIcons.refresh_rounded),
            tooltip: 'Reload',
            onPressed: _load,
          ),
          IconButton(
            icon: const Icon(QuarkIcons.settings_outlined),
            tooltip: 'Settings',
            onPressed: () => context.go('/settings'),
          ),
          const AppThemeToggle(),
        ],
      ),
      drawer: QuarkDrawer(
        activeSection: QuarkDrawerSection.docs,
        onTapFiles: () => context.go('/files'),
        onTapPhotos: () => context.go('/photos'),
        onTapDocs: () => Navigator.of(context).pop(),
        onTapSheets: () => context.go('/sheets'),
        onTapDevices: () => context.go('/devices'),
        onTapHealth: () => context.go('/health'),
        onTapVault: () => context.go(AppRoutes.vault),
        onTapSettings: () => context.go('/settings'),
      ),
      body: Column(
        children: [
          DocsSearchBar(controller: _searchController),
          Expanded(
            child: DocsBody(
              loading: _loading,
              error: _error,
              files: _filtered,
              contentResults: _contentResults,
              contentSearching: _contentSearching,
              searchQuery: _searchController.text,
              onRetry: _load,
              onCreateNew: _createNewDoc,
              onOpenDoc: _openDoc,
            ),
          ),
        ],
      ),
    );
  }
}
