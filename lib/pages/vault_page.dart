import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';
import 'package:quark/services/vault_service.dart';
import 'package:quark/utils/connection_error.dart';
import 'package:quark/utils/quark_widget.dart';
import 'package:quark/utils/web_download_stub.dart'
    if (dart.library.html) 'package:quark/utils/web_download_web.dart'
    as web_download;
import 'package:quark/widgets/core/quark_disconnected_state.dart';
import 'package:quark/widgets/layout/quark_app_bar.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:quark/widgets/quark_drawer.dart';
import 'package:quark_icons/quark_icons.dart';

class VaultPage extends StatefulWidget {
  const VaultPage({super.key});

  @override
  State<VaultPage> createState() => _VaultPageState();
}

class _VaultPageState extends State<VaultPage> {
  VaultStatus? _status;
  bool _loading = true;

  /// The thrown object, not its message — the render decides whether it means
  /// "your Quark is unreachable" or "the request failed" (#1637). Distinct
  /// from [_buildDeviceDisconnectedView], which is the vault's own USB drive
  /// being unplugged from a Quark the app can reach perfectly well.
  Object? _error;

  List<VaultEntryItem> _entries = [];
  List<VaultFolder> _folders = [];
  int? _selectedFolderId;
  String _searchQuery = '';

  final _setupPasswordCtrl = TextEditingController();
  final _setupConfirmCtrl = TextEditingController();
  String? _setupError;

  final _unlockPasswordCtrl = TextEditingController();
  String? _unlockError;
  bool _unlocking = false;

  @override
  void initState() {
    super.initState();
    _loadStatus();
  }

  @override
  void dispose() {
    _setupPasswordCtrl.dispose();
    _setupConfirmCtrl.dispose();
    _unlockPasswordCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadStatus() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final status = await VaultService.getStatus();
      setState(() {
        _status = status;
        _loading = false;
      });
      if (status.initialized && !status.locked) {
        _loadEntries();
      }
    } catch (e) {
      setState(() {
        _loading = false;
        _error = e;
      });
    }
  }

  Future<void> _loadEntries() async {
    try {
      final results = await Future.wait([
        VaultService.listEntries(),
        VaultService.listFolders(),
      ]);
      setState(() {
        _entries = results[0] as List<VaultEntryItem>;
        _folders = results[1] as List<VaultFolder>;
      });
    } catch (e) {
      setState(() => _error = e);
    }
  }

  List<VaultEntryItem> get _filteredEntries {
    var items = _entries;
    if (_selectedFolderId != null) {
      items = items.where((e) => e.folderId == _selectedFolderId).toList();
    }
    if (_searchQuery.isNotEmpty) {
      final q = _searchQuery.toLowerCase();
      items = items
          .where(
            (e) =>
                e.name.toLowerCase().contains(q) ||
                e.urlHost.toLowerCase().contains(q),
          )
          .toList();
    }
    return items;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: QuarkAppBar(
        label: 'Vault',
        icon: QuarkIcons.lock_outline,
        actions: [
          if (_status?.initialized == true && !(_status?.locked ?? true)) ...[
            IconButton(
              icon: const Icon(Icons.add),
              tooltip: 'New entry',
              onPressed: () => _showEntryEditor(context),
            ),
            IconButton(
              icon: const Icon(QuarkIcons.lock_open),
              tooltip: 'Lock vault',
              onPressed: _lockVault,
            ),
            IconButton(
              icon: const Icon(QuarkIcons.refresh),
              tooltip: 'Refresh',
              onPressed: _loadEntries,
            ),
            PopupMenuButton<String>(
              icon: const Icon(Icons.more_vert),
              onSelected: (value) {
                switch (value) {
                  case 'import':
                    _showImportDialog(context);
                  case 'export_json':
                    _doExport('json');
                  case 'export_csv':
                    _doExport('csv');
                }
              },
              itemBuilder: (_) => const [
                PopupMenuItem(
                  value: 'import',
                  child: ListTile(
                    leading: Icon(Icons.file_upload_outlined),
                    title: Text('Import'),
                    dense: true,
                    contentPadding: EdgeInsets.zero,
                  ),
                ),
                PopupMenuItem(
                  value: 'export_json',
                  child: ListTile(
                    leading: Icon(Icons.file_download_outlined),
                    title: Text('Export as JSON'),
                    dense: true,
                    contentPadding: EdgeInsets.zero,
                  ),
                ),
                PopupMenuItem(
                  value: 'export_csv',
                  child: ListTile(
                    leading: Icon(Icons.file_download_outlined),
                    title: Text('Export as CSV'),
                    dense: true,
                    contentPadding: EdgeInsets.zero,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
      drawer: QuarkDrawer(
        activeSection: QuarkDrawerSection.vault,
        onTapFiles: () => context.go(AppRoutes.files),
        onTapPhotos: () => context.go(AppRoutes.photos),
        onTapDocs: () => context.go(AppRoutes.docs),
        onTapSheets: () => context.go(AppRoutes.sheets),
        onTapDevices: () => context.go(AppRoutes.devices),
        onTapHealth: () => context.go(AppRoutes.health),
        onTapVault: () => Navigator.pop(context),
        onTapSettings: () => context.go(AppRoutes.settings),
      ),
      body: _buildBody(),
      floatingActionButton:
          (_status?.initialized == true &&
              !(_status?.locked ?? true) &&
              MediaQuery.of(context).size.width < 860)
          ? FloatingActionButton(
              onPressed: () => _showEntryEditor(context),
              child: const Icon(QuarkIcons.add),
            )
          : null,
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    final error = _error;
    if (error != null) {
      if (isQuarkUnreachableError(error)) {
        return QuarkDisconnectedView(
          onRetry: _loadStatus,
          onManageHosts: () => context.go(AppRoutes.settings),
        );
      }
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('$error', style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 16),
            ElevatedButton(onPressed: _loadStatus, child: const Text('Retry')),
          ],
        ),
      );
    }
    final status = _status;
    if (status == null) return const SizedBox.shrink();

    if (!status.initialized) return _buildSetupView();
    if (!status.deviceConnected) return _buildDeviceDisconnectedView();
    if (status.locked) return _buildUnlockView();
    return _buildEntryList();
  }

  Widget _buildSetupView() {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 400),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(QuarkIcons.shield_outlined, size: 64),
              const SizedBox(height: 16),
              Text(
                'Set up your vault',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              const Text(
                'Choose a master password to encrypt your credentials. '
                'This cannot be recovered if forgotten.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),
              TextField(
                controller: _setupPasswordCtrl,
                obscureText: true,
                decoration: const InputDecoration(
                  labelText: 'Master password',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _setupConfirmCtrl,
                obscureText: true,
                decoration: const InputDecoration(
                  labelText: 'Confirm password',
                  border: OutlineInputBorder(),
                ),
              ),
              if (_setupError != null) ...[
                const SizedBox(height: 12),
                Text(_setupError!, style: const TextStyle(color: Colors.red)),
              ],
              const SizedBox(height: 24),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    if (_setupPasswordCtrl.text.length < 8) {
                      setState(
                        () => _setupError =
                            'Password must be at least 8 characters',
                      );
                      return;
                    }
                    if (_setupPasswordCtrl.text != _setupConfirmCtrl.text) {
                      setState(() => _setupError = 'Passwords do not match');
                      return;
                    }
                    try {
                      await VaultService.setup(_setupPasswordCtrl.text);
                      _setupPasswordCtrl.clear();
                      _setupConfirmCtrl.clear();
                      _loadStatus();
                    } catch (e) {
                      setState(
                        () => _setupError = isQuarkUnreachableError(e)
                            ? quarkDisconnectedInline
                            : e.toString(),
                      );
                    }
                  },
                  child: const Text('Create Vault'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildDeviceDisconnectedView() {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 400),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                QuarkIcons.usb_off,
                size: 64,
                color: Theme.of(context).colorScheme.error,
              ),
              const SizedBox(height: 16),
              Text(
                'Vault device disconnected',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              const Text(
                'The external storage device containing your vault is not connected. '
                'Please reconnect the device to access your vault.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),
              FilledButton.icon(
                onPressed: _loadStatus,
                icon: const Icon(QuarkIcons.refresh),
                label: const Text('Check again'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildUnlockView() {
    final lockReason = _status?.lockReason ?? '';
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 400),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(QuarkIcons.lock_outline, size: 64),
              const SizedBox(height: 16),
              Text(
                'Vault is locked',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              if (lockReason.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(
                  lockReason,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context).colorScheme.outline,
                  ),
                ),
              ],
              const SizedBox(height: 24),
              TextField(
                controller: _unlockPasswordCtrl,
                obscureText: true,
                autofocus: true,
                decoration: const InputDecoration(
                  labelText: 'Master password',
                  border: OutlineInputBorder(),
                ),
                onSubmitted: (_) => _doUnlock(),
              ),
              if (_unlockError != null) ...[
                const SizedBox(height: 12),
                Text(_unlockError!, style: const TextStyle(color: Colors.red)),
              ],
              const SizedBox(height: 24),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: _unlocking ? null : _doUnlock,
                  child: _unlocking
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('Unlock'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _doUnlock() async {
    setState(() {
      _unlockError = null;
      _unlocking = true;
    });
    try {
      final ok = await VaultService.unlock(_unlockPasswordCtrl.text);
      if (!ok) {
        setState(() {
          _unlockError = 'Incorrect password';
          _unlocking = false;
        });
        return;
      }
      _unlockPasswordCtrl.clear();
      _loadStatus();
    } catch (e) {
      setState(() {
        _unlockError = isQuarkUnreachableError(e)
            ? quarkDisconnectedInline
            : e.toString();
        _unlocking = false;
      });
    }
  }

  Future<void> _lockVault() async {
    try {
      await VaultService.lock();
      _loadStatus();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to lock: $e')));
      }
    }
  }

  Widget _buildEntryList() {
    final filtered = _filteredEntries;

    return Column(
      children: [
        _buildToolbar(),
        Expanded(
          child: filtered.isEmpty
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        QuarkIcons.key_off_outlined,
                        size: 64,
                        color: Theme.of(context).colorScheme.outline,
                      ),
                      const SizedBox(height: 16),
                      Text(
                        _entries.isEmpty
                            ? 'No credentials yet'
                            : 'No matching entries',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      if (_entries.isEmpty) ...[
                        const SizedBox(height: 8),
                        const Text('Tap + to add your first password'),
                      ],
                    ],
                  ),
                )
              : ListView.builder(
                  itemCount: filtered.length,
                  itemBuilder: (context, index) {
                    final entry = filtered[index];
                    return _EntryTile(
                      entry: entry,
                      onTap: () => _showEntryDetail(context, entry.id),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildToolbar() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              decoration: const InputDecoration(
                hintText: 'Search vault...',
                prefixIcon: Icon(QuarkIcons.search),
                border: OutlineInputBorder(),
                isDense: true,
                contentPadding: EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 8,
                ),
              ),
              onChanged: (v) => setState(() => _searchQuery = v),
            ),
          ),
          const SizedBox(width: 8),
          if (_folders.isNotEmpty)
            PopupMenuButton<int?>(
              icon: const Icon(QuarkIcons.folder_outlined),
              tooltip: 'Filter by folder',
              onSelected: (v) => setState(() => _selectedFolderId = v),
              itemBuilder: (_) => [
                const PopupMenuItem(value: null, child: Text('All')),
                ..._folders.map(
                  (f) => PopupMenuItem(value: f.id, child: Text(f.name)),
                ),
              ],
            ),
        ],
      ),
    );
  }

  Future<void> _showImportDialog(BuildContext context) async {
    final result = await FilePicker.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['csv', 'json'],
      withData: true,
    );
    if (result == null || result.files.isEmpty) return;

    final file = result.files.first;
    if (file.bytes == null) return;

    if (!mounted) return;
    final format = await QuarkWidget.showDialog<String>(
      context, // ignore: use_build_context_synchronously
      builder: (ctx) => QuarkWidget.alertDialog(
        title: const Text('Import format'),
        content: Text('Importing "${file.name}". Choose the format:'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, 'auto'),
            child: const Text('Auto-detect'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, 'bitwarden'),
            child: const Text('Bitwarden CSV'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
        ],
      ),
    );
    if (format == null || !mounted) return;

    try {
      final resp = await VaultService.importEntries(
        fileBytes: file.bytes!,
        fileName: file.name,
        format: format,
      );
      _loadEntries();
      if (!mounted) return;
      final imported = resp['imported'] ?? 0;
      final skipped = resp['skipped'] ?? 0;
      final errors = (resp['errors'] as List?)?.length ?? 0;
      ScaffoldMessenger.of(
        context, // ignore: use_build_context_synchronously
      ).showSnackBar(
        SnackBar(
          content: Text(
            'Imported $imported entries'
            '${skipped > 0 ? ', $skipped skipped (duplicates)' : ''}'
            '${errors > 0 ? ', $errors errors' : ''}',
          ),
        ),
      );
    } on VaultLockedException {
      _loadStatus();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context, // ignore: use_build_context_synchronously
        ).showSnackBar(SnackBar(content: Text('Import failed: $e')));
      }
    }
  }

  Future<void> _doExport(String format) async {
    try {
      final bytes = await VaultService.exportEntries(format: format);
      final fileName = format == 'csv' ? 'quark_vault.csv' : 'quark_vault.json';
      await web_download.saveBytesForDownload(
        Uint8List.fromList(bytes),
        fileName,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Exported as $fileName')));
    } on VaultLockedException {
      _loadStatus();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Export failed: $e')));
      }
    }
  }

  Future<void> _showEntryDetail(BuildContext context, int entryId) async {
    try {
      final detail = await VaultService.getEntry(entryId);
      if (!mounted) return;
      final deleted = await Navigator.push<bool>(
        context, // ignore: use_build_context_synchronously
        MaterialPageRoute(
          builder: (_) => _EntryDetailPage(
            entry: detail,
            folders: _folders,
            onSave: (updated) async {
              await VaultService.updateEntry(
                id: detail.id,
                name: updated['name'] as String,
                url: updated['url'] as String? ?? '',
                username: updated['username'] as String? ?? '',
                password: updated['password'] as String? ?? '',
                notes: updated['notes'] as String? ?? '',
                totpSecret: updated['totpSecret'] as String? ?? '',
                folderId: updated['folderId'] as int?,
              );
              _loadEntries();
            },
            onDelete: () async {
              await VaultService.deleteEntry(detail.id);
              _loadEntries();
            },
          ),
        ),
      );
      if (deleted == true) _loadEntries();
    } on VaultLockedException {
      _loadStatus();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context, // ignore: use_build_context_synchronously
        ).showSnackBar(SnackBar(content: Text('Error: $e')));
      }
    }
  }

  Future<void> _showEntryEditor(BuildContext context) async {
    final result = await Navigator.push<bool>(
      context,
      MaterialPageRoute(builder: (_) => _EntryEditorPage(folders: _folders)),
    );
    if (result == true) _loadEntries();
  }
}

class _EntryTile extends StatelessWidget {
  final VaultEntryItem entry;
  final VoidCallback onTap;

  const _EntryTile({required this.entry, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: CircleAvatar(
        child: Text(entry.name.isNotEmpty ? entry.name[0].toUpperCase() : '?'),
      ),
      title: Text(entry.name),
      subtitle: entry.urlHost.isNotEmpty ? Text(entry.urlHost) : null,
      trailing: const Icon(QuarkIcons.chevron_right),
      onTap: onTap,
    );
  }
}

class _EntryDetailPage extends StatefulWidget {
  final VaultEntryDetail entry;
  final List<VaultFolder> folders;
  final Future<void> Function(Map<String, dynamic>) onSave;
  final Future<void> Function() onDelete;

  const _EntryDetailPage({
    required this.entry,
    required this.folders,
    required this.onSave,
    required this.onDelete,
  });

  @override
  State<_EntryDetailPage> createState() => _EntryDetailPageState();
}

class _EntryDetailPageState extends State<_EntryDetailPage> {
  bool _showPassword = false;
  bool _editing = false;
  late TextEditingController _nameCtrl;
  late TextEditingController _urlCtrl;
  late TextEditingController _usernameCtrl;
  late TextEditingController _passwordCtrl;
  late TextEditingController _notesCtrl;
  int? _folderId;

  @override
  void initState() {
    super.initState();
    _nameCtrl = TextEditingController(text: widget.entry.name);
    _urlCtrl = TextEditingController(text: widget.entry.url);
    _usernameCtrl = TextEditingController(text: widget.entry.username);
    _passwordCtrl = TextEditingController(text: widget.entry.password);
    _notesCtrl = TextEditingController(text: widget.entry.notes);
    _folderId = widget.entry.folderId;
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _urlCtrl.dispose();
    _usernameCtrl.dispose();
    _passwordCtrl.dispose();
    _notesCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(_editing ? 'Edit Entry' : widget.entry.name),
        actions: [
          if (!_editing)
            IconButton(
              icon: const Icon(QuarkIcons.edit),
              onPressed: () => setState(() => _editing = true),
            ),
          if (!_editing)
            IconButton(
              icon: const Icon(QuarkIcons.delete_outline),
              onPressed: _confirmDelete,
            ),
          if (_editing)
            TextButton(onPressed: _saveEntry, child: const Text('Save')),
          const ThemeToggleButton(),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: _editing ? _buildEditForm() : _buildDetailView(),
      ),
    );
  }

  Widget _buildDetailView() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _DetailRow(
          label: 'Username',
          value: widget.entry.username,
          copiable: true,
        ),
        _DetailRow(
          label: 'Password',
          value: _showPassword ? widget.entry.password : '••••••••',
          copiable: true,
          copyValue: widget.entry.password,
          trailing: IconButton(
            icon: Icon(
              _showPassword ? QuarkIcons.visibility_off : QuarkIcons.visibility,
            ),
            onPressed: () => setState(() => _showPassword = !_showPassword),
          ),
        ),
        if (widget.entry.url.isNotEmpty)
          _DetailRow(label: 'URL', value: widget.entry.url, copiable: true),
        if (widget.entry.notes.isNotEmpty)
          _DetailRow(label: 'Notes', value: widget.entry.notes),
        if (widget.entry.totpSecret.isNotEmpty)
          _DetailRow(label: 'TOTP', value: '(configured)', copiable: false),
      ],
    );
  }

  Widget _buildEditForm() {
    return Column(
      children: [
        TextField(
          controller: _nameCtrl,
          decoration: const InputDecoration(
            labelText: 'Name',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _urlCtrl,
          decoration: const InputDecoration(
            labelText: 'URL',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _usernameCtrl,
          decoration: const InputDecoration(
            labelText: 'Username',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _passwordCtrl,
          obscureText: !_showPassword,
          decoration: InputDecoration(
            labelText: 'Password',
            border: const OutlineInputBorder(),
            suffixIcon: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                IconButton(
                  icon: Icon(
                    _showPassword
                        ? QuarkIcons.visibility_off
                        : QuarkIcons.visibility,
                  ),
                  onPressed: () =>
                      setState(() => _showPassword = !_showPassword),
                ),
                IconButton(
                  icon: const Icon(QuarkIcons.casino),
                  tooltip: 'Generate password',
                  onPressed: _generatePassword,
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _notesCtrl,
          maxLines: 3,
          decoration: const InputDecoration(
            labelText: 'Notes',
            border: OutlineInputBorder(),
          ),
        ),
        if (widget.folders.isNotEmpty) ...[
          const SizedBox(height: 12),
          DropdownButtonFormField<int?>(
            initialValue: _folderId,
            decoration: const InputDecoration(
              labelText: 'Folder',
              border: OutlineInputBorder(),
            ),
            items: [
              const DropdownMenuItem(value: null, child: Text('None')),
              ...widget.folders.map(
                (f) => DropdownMenuItem(value: f.id, child: Text(f.name)),
              ),
            ],
            onChanged: (v) => setState(() => _folderId = v),
          ),
        ],
      ],
    );
  }

  Future<void> _generatePassword() async {
    try {
      final pw = await VaultService.generatePassword();
      _passwordCtrl.text = pw;
      setState(() => _showPassword = true);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to generate: $e')));
      }
    }
  }

  Future<void> _saveEntry() async {
    if (_nameCtrl.text.isEmpty) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Name is required')));
      return;
    }
    try {
      await widget.onSave({
        'name': _nameCtrl.text,
        'url': _urlCtrl.text,
        'username': _usernameCtrl.text,
        'password': _passwordCtrl.text,
        'notes': _notesCtrl.text,
        'folderId': _folderId,
      });
      if (mounted) Navigator.pop(context);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Save failed: $e')));
      }
    }
  }

  Future<void> _confirmDelete() async {
    final confirmed = await QuarkWidget.showDialog<bool>(
      context,
      builder: (ctx) => QuarkWidget.alertDialog(
        title: const Text('Delete entry?'),
        content: Text('Delete "${widget.entry.name}"? This cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Delete', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
    if (confirmed == true) {
      await widget.onDelete();
      if (mounted) Navigator.pop(context, true);
    }
  }
}

class _DetailRow extends StatelessWidget {
  final String label;
  final String value;
  final bool copiable;
  final String? copyValue;
  final Widget? trailing;

  const _DetailRow({
    required this.label,
    required this.value,
    this.copiable = false,
    this.copyValue,
    this.trailing,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: Theme.of(context).colorScheme.outline,
            ),
          ),
          const SizedBox(height: 4),
          Row(
            children: [
              Expanded(child: SelectableText(value)),
              if (copiable)
                IconButton(
                  icon: const Icon(QuarkIcons.copy, size: 18),
                  onPressed: () {
                    Clipboard.setData(ClipboardData(text: copyValue ?? value));
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text('$label copied'),
                        duration: const Duration(seconds: 1),
                      ),
                    );
                  },
                ),
              ?trailing,
            ],
          ),
        ],
      ),
    );
  }
}

class _EntryEditorPage extends StatefulWidget {
  final List<VaultFolder> folders;

  const _EntryEditorPage({required this.folders});

  @override
  State<_EntryEditorPage> createState() => _EntryEditorPageState();
}

class _EntryEditorPageState extends State<_EntryEditorPage> {
  final _nameCtrl = TextEditingController();
  final _urlCtrl = TextEditingController();
  final _usernameCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  final _notesCtrl = TextEditingController();
  int? _folderId;
  bool _showPassword = false;
  bool _saving = false;

  @override
  void dispose() {
    _nameCtrl.dispose();
    _urlCtrl.dispose();
    _usernameCtrl.dispose();
    _passwordCtrl.dispose();
    _notesCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('New Entry'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _save,
            child: const Text('Save'),
          ),
          const ThemeToggleButton(),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            TextField(
              controller: _nameCtrl,
              autofocus: true,
              decoration: const InputDecoration(
                labelText: 'Name',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _urlCtrl,
              decoration: const InputDecoration(
                labelText: 'URL',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _usernameCtrl,
              decoration: const InputDecoration(
                labelText: 'Username / Email',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _passwordCtrl,
              obscureText: !_showPassword,
              decoration: InputDecoration(
                labelText: 'Password',
                border: const OutlineInputBorder(),
                suffixIcon: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      icon: Icon(
                        _showPassword
                            ? QuarkIcons.visibility_off
                            : QuarkIcons.visibility,
                      ),
                      onPressed: () =>
                          setState(() => _showPassword = !_showPassword),
                    ),
                    IconButton(
                      icon: const Icon(QuarkIcons.casino),
                      tooltip: 'Generate password',
                      onPressed: _generatePassword,
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _notesCtrl,
              maxLines: 3,
              decoration: const InputDecoration(
                labelText: 'Notes',
                border: OutlineInputBorder(),
              ),
            ),
            if (widget.folders.isNotEmpty) ...[
              const SizedBox(height: 12),
              DropdownButtonFormField<int?>(
                initialValue: _folderId,
                decoration: const InputDecoration(
                  labelText: 'Folder',
                  border: OutlineInputBorder(),
                ),
                items: [
                  const DropdownMenuItem(value: null, child: Text('None')),
                  ...widget.folders.map(
                    (f) => DropdownMenuItem(value: f.id, child: Text(f.name)),
                  ),
                ],
                onChanged: (v) => setState(() => _folderId = v),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Future<void> _generatePassword() async {
    try {
      final pw = await VaultService.generatePassword();
      _passwordCtrl.text = pw;
      setState(() => _showPassword = true);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to generate: $e')));
      }
    }
  }

  Future<void> _save() async {
    if (_nameCtrl.text.isEmpty) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Name is required')));
      return;
    }
    setState(() => _saving = true);
    try {
      await VaultService.createEntry(
        name: _nameCtrl.text,
        url: _urlCtrl.text,
        username: _usernameCtrl.text,
        password: _passwordCtrl.text,
        notes: _notesCtrl.text,
        folderId: _folderId,
      );
      if (mounted) Navigator.pop(context, true);
    } catch (e) {
      setState(() => _saving = false);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Save failed: $e')));
      }
    }
  }
}
