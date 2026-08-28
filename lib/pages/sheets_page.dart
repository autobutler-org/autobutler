import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:quark/models/file_node.dart';
import 'package:quark/router.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/services/content_search_service.dart';
import 'package:quark/utils/connection_error.dart';
import 'package:quark/utils/file_browser_dialog_utils.dart';
import 'package:quark/utils/safe_set_state_mixin.dart';
import 'package:quark/widgets/core/quark_disconnected_state.dart';
import 'package:quark/widgets/layout/quark_app_bar.dart';
import 'package:quark/widgets/quark_drawer.dart';
import 'package:quark_icons/quark_icons.dart';

class SheetsPage extends StatefulWidget {
  const SheetsPage({super.key});

  @override
  State<SheetsPage> createState() => _SheetsPageState();
}

class _SheetsPageState extends State<SheetsPage> with SafeSetStateMixin {
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
    setStateSafely(() {
      _loading = true;
      _error = null;
    });
    try {
      final files = await FilesService.getFilesByType('qsheet');
      setStateSafely(() {
        _files = files;
        _applyFilter();
        _loading = false;
      });
    } catch (e) {
      setStateSafely(() {
        _error = e;
        _loading = false;
      });
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

  Future<void> _openSheet(FileNode node) async {
    await context.push(
      AppRoutes.sheetFile(node.apiPath, serial: node.deviceSerial),
    );
    _load();
  }

  Future<void> _createNewSheet() async {
    final name = await promptForNewFileName(
      context,
      title: 'New Spreadsheet',
      hintText: 'Spreadsheet name',
    );
    if (name == null || name.isEmpty || !mounted) return;

    try {
      final fileName = name.endsWith('.qsheet') ? name : '$name.qsheet';
      final bytes =
          '{"tabs":[{"name":"Sheet 1","data":{"columns":[],"rows":[]}}]}'
              .codeUnits;
      final file = http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: fileName,
      );
      await FilesService.uploadFilesFromFormData('', [file]);
      if (!mounted) return;
      context.push(AppRoutes.sheetFile(fileName));
      _load();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to create sheet: $e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: QuarkAppBar(
        label: 'Sheets',
        icon: QuarkIcons.table_chart_outlined,
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: 'New spreadsheet',
            onPressed: _createNewSheet,
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
        ],
      ),
      drawer: QuarkDrawer(
        activeSection: QuarkDrawerSection.sheets,
        onTapFiles: () => context.go('/files'),
        onTapPhotos: () => context.go('/photos'),
        onTapDocs: () => context.go('/docs'),
        onTapSheets: () => Navigator.of(context).pop(),
        onTapDevices: () => context.go('/devices'),
        onTapHealth: () => context.go('/health'),
        onTapVault: () => context.go(AppRoutes.vault),
        onTapSettings: () => context.go('/settings'),
      ),
      body: Column(
        children: [
          _buildSearchBar(colorScheme),
          Expanded(child: _buildBody(colorScheme)),
        ],
      ),
    );
  }

  Widget _buildSearchBar(ColorScheme colorScheme) {
    return Container(
      decoration: BoxDecoration(
        color: colorScheme.secondary,
        border: Border(bottom: BorderSide(color: colorScheme.outline)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: TextField(
        controller: _searchController,
        decoration: InputDecoration(
          hintText: 'Search sheets…',
          prefixIcon: const Icon(QuarkIcons.search_rounded, size: 20),
          suffixIcon: _searchController.text.isNotEmpty
              ? IconButton(
                  icon: const Icon(QuarkIcons.clear_rounded, size: 18),
                  onPressed: () => _searchController.clear(),
                )
              : null,
          isDense: true,
          filled: true,
          fillColor: colorScheme.surfaceContainerHighest,
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colorScheme.outline),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colorScheme.outline),
          ),
        ),
      ),
    );
  }

  Widget _buildBody(ColorScheme colorScheme) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    final error = _error;
    if (error != null) {
      if (isQuarkUnreachableError(error)) {
        return QuarkDisconnectedView(
          onRetry: _load,
          onManageHosts: () => context.go(AppRoutes.settings),
        );
      }
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(QuarkIcons.error_outline, size: 40, color: colorScheme.error),
            const SizedBox(height: 12),
            Text('$error', textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton(onPressed: _load, child: const Text('Retry')),
          ],
        ),
      );
    }
    // Content matches are rendered by the list below, so the empty state must
    // account for them too. Checking only _filtered here would short-circuit
    // every content-only search — the common case, since a query that matches
    // a sheet's contents usually does not also match its filename. Both the
    // guard and the list read the same flag so they cannot disagree about
    // whether there is anything to show.
    final hasContentResults =
        _searchController.text.isNotEmpty && _contentResults.isNotEmpty;

    if (_filtered.isEmpty && !hasContentResults) {
      final isSearching = _searchController.text.isNotEmpty;
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              QuarkIcons.table_chart_outlined,
              size: 48,
              color: colorScheme.onSurface.withValues(alpha: 0.3),
            ),
            const SizedBox(height: 12),
            Text(
              isSearching ? 'No sheets match your search.' : 'No sheets yet.',
              textAlign: TextAlign.center,
              style: TextStyle(
                color: colorScheme.onSurface.withValues(alpha: 0.5),
              ),
            ),
            if (_contentSearching) ...const [
              SizedBox(height: 16),
              SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ],
            if (!isSearching) ...[
              const SizedBox(height: 16),
              FilledButton.icon(
                onPressed: _createNewSheet,
                icon: const Icon(Icons.add),
                label: const Text('Create new sheet'),
              ),
            ],
          ],
        ),
      );
    }

    final totalItems =
        _filtered.length + (hasContentResults ? _contentResults.length + 1 : 0);

    return ListView.builder(
      itemCount: totalItems,
      itemBuilder: (context, i) {
        if (i < _filtered.length) {
          return _buildSheetTile(_filtered[i], colorScheme);
        }
        final ci = i - _filtered.length;
        if (ci == 0) {
          return Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: Row(
              children: [
                Icon(
                  QuarkIcons.search_rounded,
                  size: 14,
                  color: colorScheme.onSurface.withValues(alpha: 0.5),
                ),
                const SizedBox(width: 6),
                Text(
                  'Content matches',
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: colorScheme.onSurface.withValues(alpha: 0.5),
                    letterSpacing: 0.5,
                  ),
                ),
              ],
            ),
          );
        }
        return _buildContentResultTile(_contentResults[ci - 1], colorScheme);
      },
    );
  }

  Widget _buildContentResultTile(
    ContentSearchResult result,
    ColorScheme colorScheme,
  ) {
    return ListTile(
      leading: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: colorScheme.tertiaryContainer,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Icon(
          QuarkIcons.search_rounded,
          size: 18,
          color: colorScheme.onTertiaryContainer,
        ),
      ),
      title: Text(
        result.filename,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        result.plainSnippet,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(
          color: colorScheme.onSurface.withValues(alpha: 0.6),
          fontSize: 12,
        ),
      ),
      onTap: () => context.push(
        AppRoutes.sheetFile(result.relPath, serial: result.deviceSerial),
      ),
    );
  }

  Widget _buildSheetTile(FileNode node, ColorScheme colorScheme) {
    final folder = node.dirPath.contains('/')
        ? node.dirPath.substring(0, node.dirPath.lastIndexOf('/'))
        : '';
    final subtitle = [
      if (node.deviceName.isNotEmpty) node.deviceName,
      if (folder.isNotEmpty) folder,
    ].join(' · ');

    return ListTile(
      leading: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: Colors.green.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(6),
        ),
        child: Icon(
          QuarkIcons.table_chart_outlined,
          size: 18,
          color: Colors.green.shade600,
        ),
      ),
      title: Text(
        node.name.replaceAll(RegExp(r'\.qsheet$', caseSensitive: false), ''),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: subtitle.isNotEmpty
          ? Text(
              subtitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 12,
                color: colorScheme.onSurface.withValues(alpha: 0.55),
              ),
            )
          : null,
      onTap: () => _openSheet(node),
    );
  }
}
