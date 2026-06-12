import 'package:autobutler/router.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/utils/safe_set_state_mixin.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:autobutler/widgets/layout/autobutler_app_bar.dart';
import 'package:flutter/material.dart';
import 'package:autobutler_icons/autobutler_icons.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;

class DocsPage extends StatefulWidget {
  const DocsPage({super.key});

  @override
  State<DocsPage> createState() => _DocsPageState();
}

class _DocsPageState extends State<DocsPage> with SafeSetStateMixin {
  List<CirrusFileNode> _files = [];
  List<CirrusFileNode> _filtered = [];
  bool _loading = true;
  String? _error;
  final _searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _searchController.addListener(_applyFilter);
    _load();
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setStateSafely(() {
      _loading = true;
      _error = null;
    });
    try {
      final files = await CirrusService.getFilesByType('abdoc');
      setStateSafely(() {
        _files = files;
        _applyFilter();
        _loading = false;
      });
    } catch (e) {
      setStateSafely(() {
        _error = e.toString();
        _loading = false;
      });
    }
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

  Future<void> _openDoc(CirrusFileNode node) async {
    await context.push(
      AppRoutes.docFile(node.apiPath, serial: node.deviceSerial),
    );
    // Refresh in case the doc was renamed or deleted.
    _load();
  }

  Future<void> _createNewDoc() async {
    final nameController = TextEditingController();
    final name = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New Document'),
        content: TextField(
          controller: nameController,
          autofocus: true,
          decoration: const InputDecoration(
            hintText: 'Document name',
            border: OutlineInputBorder(),
          ),
          onSubmitted: (v) => Navigator.pop(ctx, v.trim()),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, nameController.text.trim()),
            child: const Text('Create'),
          ),
        ],
      ),
    );
    if (name == null || name.isEmpty || !mounted) return;

    try {
      final fileName = name.endsWith('.abdoc') ? name : '$name.abdoc';
      final bytes = '{"ops":[{"insert":"\\n"}]}'.codeUnits;
      final file = http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: fileName,
      );
      await CirrusService.uploadFilesFromFormData('', [file]);
      if (!mounted) return;
      context.push(AppRoutes.docFile(fileName));
      _load();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to create doc: $e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AutobutlerAppBar(
        label: 'Docs',
        icon: AutobutlerIcons.description_outlined,
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: 'New document',
            onPressed: _createNewDoc,
          ),
          IconButton(
            icon: const Icon(AutobutlerIcons.refresh_rounded),
            tooltip: 'Reload',
            onPressed: _load,
          ),
          IconButton(
            icon: const Icon(AutobutlerIcons.settings_outlined),
            tooltip: 'Settings',
            onPressed: () => context.go('/settings'),
          ),
        ],
      ),
      drawer: AutobutlerDrawer(
        activeSection: AutobutlerDrawerSection.docs,
        onTapCirrus: () => context.go('/cirrus'),
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
          hintText: 'Search docs…',
          prefixIcon: const Icon(AutobutlerIcons.search_rounded, size: 20),
          suffixIcon: _searchController.text.isNotEmpty
              ? IconButton(
                  icon: const Icon(AutobutlerIcons.clear_rounded, size: 18),
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
    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              AutobutlerIcons.error_outline,
              size: 40,
              color: colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(_error!, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton(onPressed: _load, child: const Text('Retry')),
          ],
        ),
      );
    }
    if (_filtered.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              AutobutlerIcons.description_outlined,
              size: 48,
              color: colorScheme.onSurface.withValues(alpha: 0.3),
            ),
            const SizedBox(height: 12),
            Text(
              _searchController.text.isNotEmpty
                  ? 'No docs match your search.'
                  : 'No docs yet.',
              textAlign: TextAlign.center,
              style: TextStyle(
                color: colorScheme.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ],
        ),
      );
    }

    return ListView.builder(
      itemCount: _filtered.length,
      itemBuilder: (context, i) => _buildDocTile(_filtered[i], colorScheme),
    );
  }

  Widget _buildDocTile(CirrusFileNode node, ColorScheme colorScheme) {
    // Strip the filename from dirPath to get the folder path for the subtitle.
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
          color: colorScheme.primary.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(6),
        ),
        child: Icon(
          AutobutlerIcons.description_outlined,
          size: 18,
          color: colorScheme.primary,
        ),
      ),
      title: Text(
        node.name.replaceAll(RegExp(r'\.abdoc$', caseSensitive: false), ''),
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
      onTap: () => _openDoc(node),
    );
  }
}
