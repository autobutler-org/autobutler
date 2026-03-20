import 'package:autobutler/router.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/auth_service.dart';
import 'package:autobutler/services/health_service.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/services/connected_devices_service.dart';
import 'package:autobutler/services/sbom_service.dart';
import 'package:autobutler/services/storage_service.dart';
import 'package:autobutler/utils/autobutler_widget.dart';
import 'package:autobutler/widgets/autobutler_drawer.dart';
import 'package:autobutler/widgets/refresh_icon_button.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  List<HostEntry> _hosts = [];
  int _active = -1;
  ThemeMode _theme = ThemeMode.system;
  String? _installedVersion;
  List<String> _availableVersions = [];
  String? _selectedUpdateVersion;
  bool _isLoadingVersionInfo = false;
  bool _isUpdatingVersion = false;
  String? _versionLoadError;

  // SBOM state
  GoSbom? _goSbom;
  List<FlutterPackage>? _flutterSbom;
  bool _isLoadingSbom = false;
  String? _sbomError;

  int _refreshIntervalSeconds = 15;

  // Connected devices state
  List<ConnectedDevice> _connectedDevices = [];
  bool _isLoadingDevices = false;
  String? _devicesError;

  // Storage devices state
  List<StorageDevice> _storageDevices = [];
  bool _isLoadingStorage = false;
  String? _storageError;

  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() {
    _hosts = AppSettings.instance.hosts;
    _active = AppSettings.instance.activeIndex;
    _theme = AppSettings.instance.themeMode.value;
    _refreshIntervalSeconds = AppSettings.instance.refreshIntervalSeconds;
    setState(() {});
    _loadVersionInfo();
    _loadSbom();
    _loadDevices();
    _loadStorageDevices();
  }

  Future<void> _loadDevices() async {
    if (AppSettings.instance.activeHost == null) {
      setState(() {
        _connectedDevices = [];
        _devicesError = null;
        _isLoadingDevices = false;
      });
      return;
    }
    setState(() {
      _isLoadingDevices = true;
      _devicesError = null;
    });
    try {
      final devices = await ConnectedDevicesService.listDevices();
      if (!mounted) return;
      setState(() {
        _connectedDevices = devices;
        _isLoadingDevices = false;
      });
    } catch (e) {
      debugPrint('[settings_page.dart] Error: $e');
      if (!mounted) return;
      setState(() {
        _devicesError = e.toString();
        _isLoadingDevices = false;
      });
    }
  }

  Future<void> _deleteDevice(int id) async {
    try {
      await ConnectedDevicesService.deleteDevice(id);
      await _loadDevices();
    } catch (e) {
      debugPrint('[settings_page.dart] Error: $e');
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Failed to remove device: $e')));
    }
  }

  Future<void> _loadStorageDevices() async {
    if (AppSettings.instance.activeHost == null) {
      setState(() {
        _storageDevices = [];
        _storageError = null;
        _isLoadingStorage = false;
      });
      return;
    }
    setState(() {
      _isLoadingStorage = true;
      _storageError = null;
    });
    try {
      final devices = await StorageService.listDevices();
      if (!mounted) return;
      setState(() {
        _storageDevices = devices;
        _isLoadingStorage = false;
      });
    } catch (e) {
      debugPrint('[settings_page.dart] Error loading storage devices: $e');
      if (!mounted) return;
      setState(() {
        _storageError = e.toString();
        _isLoadingStorage = false;
      });
    }
  }

  Future<void> _renameStorageDevice(StorageDevice device) async {
    final controller = TextEditingController(text: device.name);
    final newName = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Rename device'),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(labelText: 'Display name'),
          autofocus: true,
          onSubmitted: (v) => Navigator.of(ctx).pop(v.trim()),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(controller.text.trim()),
            child: const Text('Rename'),
          ),
        ],
      ),
    );
    WidgetsBinding.instance.addPostFrameCallback((_) {
      controller.dispose();
    });
    if (newName == null || newName.isEmpty) return;
    try {
      await StorageService.renameDevice(device.devicePath, newName);
      await _loadStorageDevices();
    } catch (e) {
      debugPrint('[settings_page.dart] Error renaming storage device: $e');
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Failed to rename device: $e')));
    }
  }

  Future<void> _loadSbom() async {
    setState(() {
      _isLoadingSbom = true;
      _sbomError = null;
    });

    GoSbom? nextGoSbom;
    List<FlutterPackage>? nextFlutterSbom;
    final errors = <String>[];

    if (AppSettings.instance.activeHost != null) {
      try {
        nextGoSbom = await SbomService.getGoSbom();
      } catch (e) {
        debugPrint('[settings_page.dart] Error: $e');
        errors.add('Go SBOM: $e');
      }
    }

    try {
      nextFlutterSbom = await SbomService.getFlutterSbom();
    } catch (e) {
      debugPrint('[settings_page.dart] Error: $e');
      errors.add('Flutter SBOM: $e');
    }

    if (!mounted) return;
    setState(() {
      _goSbom = nextGoSbom;
      _flutterSbom = nextFlutterSbom;
      _sbomError = errors.isEmpty ? null : errors.join('\n');
      _isLoadingSbom = false;
    });
  }

  Future<void> _loadVersionInfo() async {
    if (AppSettings.instance.activeHost == null) {
      setState(() {
        _installedVersion = null;
        _availableVersions = const [];
        _selectedUpdateVersion = null;
        _versionLoadError = null;
        _isLoadingVersionInfo = false;
      });
      return;
    }

    setState(() {
      _isLoadingVersionInfo = true;
      _versionLoadError = null;
    });

    try {
      final installed = await CirrusService.getInstalledVersion();
      final versions = await CirrusService.listAvailableVersions();
      if (!mounted) return;

      final installedVersion =
          (installed['semver'] as String?) ??
          (installed['version'] as String?) ??
          'Unknown';
      final availableVersions = versions
          .map((m) => (m['version'] as String?) ?? '')
          .where((v) => v.isNotEmpty)
          .toList(growable: false);
      final selectedVersion = availableVersions.contains(_selectedUpdateVersion)
          ? _selectedUpdateVersion
          : (availableVersions.isNotEmpty ? availableVersions.first : null);

      setState(() {
        _installedVersion = installedVersion;
        _availableVersions = availableVersions;
        _selectedUpdateVersion = selectedVersion;
        _isLoadingVersionInfo = false;
      });
    } catch (e) {
      debugPrint('[settings_page.dart] Error: $e');
      if (!mounted) return;
      setState(() {
        _versionLoadError = e.toString();
        _isLoadingVersionInfo = false;
      });
    }
  }

  Future<void> _performUpdate() async {
    final version = _selectedUpdateVersion;
    if (version == null || _isUpdatingVersion) return;

    setState(() {
      _isUpdatingVersion = true;
    });

    try {
      await CirrusService.updateToVersion(version);
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Update started for $version')));
      await _loadVersionInfo();
    } catch (e) {
      debugPrint('[settings_page.dart] Error: $e');
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Update failed: ${e.toString()}')));
    } finally {
      if (mounted) {
        setState(() {
          _isUpdatingVersion = false;
        });
      }
    }
  }

  Future<void> _addOrEditHost({int? index}) async {
    final isEdit = index != null;
    final idx = index ?? 0;
    final nameController = TextEditingController(
      text: isEdit ? _hosts[idx].name : '',
    );
    final hostController = TextEditingController(
      text: isEdit ? _hosts[idx].hostAddress : '',
    );

    final result = await AutobutlerWidget.showDialog<bool>(
      context,
      builder: (context) => AutobutlerWidget.alertDialog(
        title: Text(isEdit ? 'Edit host' : 'Add host'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AutobutlerWidget.textField(
              controller: nameController,
              autofocus: true,
              hintText: 'Name',
            ),
            const SizedBox(height: 8),
            AutobutlerWidget.textField(
              controller: hostController,
              hintText: 'http://<hostname>:<port>',
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () async {
              final name = nameController.text.trim();
              final host = hostController.text.trim();
              if (name.isEmpty || host.isEmpty) return;
              final entry = HostEntry(name: name, hostAddress: host);
              final navigator = Navigator.of(context);
              if (isEdit) {
                await AppSettings.instance.updateHost(idx, entry);
              } else {
                await AppSettings.instance.addHost(entry);
              }
              navigator.pop(true);
            },
            child: const Text('Save'),
          ),
        ],
      ),
    );
    WidgetsBinding.instance.addPostFrameCallback((_) {
      nameController.dispose();
      hostController.dispose();
    });

    if (result == true) {
      _load();
    }
  }

  Future<void> _removeHost(int index) async {
    final confirm = await AutobutlerWidget.showDialog(
      context,
      builder: (context) => AutobutlerWidget.alertDialog(
        title: const Text('Remove host'),
        content: const Text('Are you sure you want to remove this host?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Remove'),
          ),
        ],
      ),
    );

    if (confirm == true) {
      await AppSettings.instance.removeHost(index);
      _load();
    }
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
        title: const Text('Settings'),
        centerTitle: true,
      ),
      drawer: AutobutlerDrawer(
        activeSection: AutobutlerDrawerSection.settings,
        onTapCirrus: () {
          context.go(AppRoutes.cirrus);
        },
        onTapPhotos: () {
          context.go(AppRoutes.photos);
        },
        onTapHealth: () {
          context.go(AppRoutes.health);
        },
        onTapSettings: () {
          Navigator.of(context).pop();
        },
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text(
            'Autobutler',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          // Sign out — only show if there's an active session
          if (AppSettings.instance.sessionToken != null) ...[
            const Text(
              'Account',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Card(
              child: ListTile(
                leading: const Icon(Icons.logout),
                title: const Text('Sign out'),
                onTap: _signOut,
              ),
            ),
            const SizedBox(height: 24),
          ],
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    'Installed version',
                    style: TextStyle(fontWeight: FontWeight.w600),
                  ),
                  const SizedBox(height: 6),
                  if (AppSettings.instance.activeHost == null)
                    const Text('No target host configured')
                  else if (_isLoadingVersionInfo)
                    const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  else if (_versionLoadError != null)
                    Text(
                      'Failed to load version info',
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    )
                  else
                    Text(
                      _installedVersion ?? 'Unknown',
                      style: const TextStyle(
                        fontSize: 28,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  const SizedBox(height: 16),
                  if (_availableVersions.isEmpty &&
                      !_isLoadingVersionInfo &&
                      _versionLoadError == null &&
                      AppSettings.instance.activeHost != null)
                    const Text('No updates available')
                  else if (_availableVersions.isNotEmpty) ...[
                    DropdownButtonFormField<String>(
                      initialValue: _selectedUpdateVersion,
                      items: _availableVersions
                          .map(
                            (v) => DropdownMenuItem<String>(
                              value: v,
                              child: Text(v),
                            ),
                          )
                          .toList(),
                      onChanged: (_isLoadingVersionInfo || _isUpdatingVersion)
                          ? null
                          : (v) {
                              setState(() {
                                _selectedUpdateVersion = v;
                              });
                            },
                      decoration: const InputDecoration(
                        labelText: 'Update Autobutler to version',
                        border: OutlineInputBorder(),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Align(
                      alignment: Alignment.centerRight,
                      child: ElevatedButton.icon(
                        onPressed:
                            (_selectedUpdateVersion == null ||
                                _isUpdatingVersion)
                            ? null
                            : _performUpdate,
                        icon: _isUpdatingVersion
                            ? const SizedBox(
                                width: 14,
                                height: 14,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
                              )
                            : const Icon(Icons.update),
                        label: Text(
                          _isUpdatingVersion ? 'Updating...' : 'Start update',
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            'Auto-refresh interval',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          DropdownButtonFormField<int>(
            initialValue: _refreshIntervalSeconds,
            decoration: const InputDecoration(
              border: OutlineInputBorder(),
              contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            ),
            items: const [
              DropdownMenuItem(value: 0, child: Text('Disabled')),
              DropdownMenuItem(value: 10, child: Text('10 seconds')),
              DropdownMenuItem(value: 15, child: Text('15 seconds')),
              DropdownMenuItem(value: 30, child: Text('30 seconds')),
              DropdownMenuItem(value: 60, child: Text('1 minute')),
              DropdownMenuItem(value: 120, child: Text('2 minutes')),
              DropdownMenuItem(value: 300, child: Text('5 minutes')),
            ],
            onChanged: (v) async {
              if (v == null) return;
              await AppSettings.instance.setRefreshIntervalSeconds(v);
              setState(() => _refreshIntervalSeconds = v);
            },
          ),
          const SizedBox(height: 24),
          const Text(
            'Theme',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          RadioGroup<ThemeMode>(
            groupValue: _theme,
            onChanged: (v) async {
              if (v == null) return;
              await AppSettings.instance.setThemeMode(v);
              setState(() {
                _theme = v;
              });
            },
            child: const Column(
              children: [
                RadioListTile<ThemeMode>(
                  title: Text('System'),
                  value: ThemeMode.system,
                ),
                RadioListTile<ThemeMode>(
                  title: Text('Light'),
                  value: ThemeMode.light,
                ),
                RadioListTile<ThemeMode>(
                  title: Text('Dark'),
                  value: ThemeMode.dark,
                ),
              ],
            ),
          ),
          // Storage devices
          if (AppSettings.instance.activeHost != null) ...[
            const Text(
              'Storage',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Card(
              child: ExpansionTile(
                title: const Text(
                  'Storage devices',
                  style: TextStyle(fontWeight: FontWeight.w600),
                ),
                subtitle: Text(
                  _isLoadingStorage
                      ? 'Loading...'
                      : _storageError != null
                      ? 'Failed to load'
                      : _storageDevices.isEmpty
                      ? 'No devices found'
                      : '${_storageDevices.length} device${_storageDevices.length == 1 ? '' : 's'}',
                ),
                children: [
                  Align(
                    alignment: Alignment.centerRight,
                    child: Padding(
                      padding: const EdgeInsets.only(right: 12),
                      child: RefreshIconButton(
                        isRefreshing: _isLoadingStorage,
                        onPressed: _loadStorageDevices,
                        tooltip: 'Refresh',
                      ),
                    ),
                  ),
                  if (_isLoadingStorage)
                    const Padding(
                      padding: EdgeInsets.all(8),
                      child: Center(
                        child: SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                      ),
                    )
                  else if (_storageError != null)
                    ListTile(
                      leading: Icon(
                        Icons.error_outline,
                        color: Theme.of(context).colorScheme.error,
                      ),
                      title: const Text('Failed to load storage devices'),
                      subtitle: Text(_storageError!),
                    )
                  else if (_storageDevices.isEmpty)
                    const ListTile(title: Text('No storage devices found'))
                  else
                    ..._storageDevices.map((device) {
                      return ListTile(
                        leading: Icon(
                          device.isInternal
                              ? Icons.storage_rounded
                              : Icons.usb_rounded,
                        ),
                        title: Text(
                          device.name.isNotEmpty
                              ? device.name
                              : device.devicePath,
                        ),
                        subtitle: Text(
                          '${device.usedDisplay} · ${device.usedPercent.toStringAsFixed(1)}% used · ${device.fileSystem}',
                          style: const TextStyle(fontSize: 12),
                        ),
                        trailing: IconButton(
                          icon: const Icon(Icons.edit_outlined),
                          tooltip: 'Rename',
                          onPressed: () => _renameStorageDevice(device),
                        ),
                      );
                    }),
                ],
              ),
            ),
          ],
          const SizedBox(height: 24),
          const Text(
            'Backend hosts',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          RadioGroup<int>(
            groupValue: _active,
            onChanged: (v) async {
              if (v == null) return;
              await AppSettings.instance.setActiveIndex(v);
              _load();
            },
            child: Column(
              children: _hosts.asMap().entries.map((e) {
                final idx = e.key;
                final host = e.value;
                return Card(
                  child: ListTile(
                    leading: Radio<int>(value: idx),
                    title: Text(host.name),
                    subtitle: Text(host.hostAddress),
                    trailing: PopupMenuButton<String>(
                      onSelected: (action) {
                        if (action == 'edit') {
                          _addOrEditHost(index: idx);
                        } else if (action == 'remove') {
                          _removeHost(idx);
                        }
                      },
                      itemBuilder: (_) => [
                        const PopupMenuItem(value: 'edit', child: Text('Edit')),
                        const PopupMenuItem(
                          value: 'remove',
                          child: Text('Remove'),
                        ),
                      ],
                    ),
                    onTap: () async {
                      await AppSettings.instance.setActiveIndex(idx);
                      _load();
                    },
                  ),
                );
              }).toList(),
            ),
          ),
          const SizedBox(height: 8),
          ElevatedButton.icon(
            onPressed: () => _addOrEditHost(),
            icon: const Icon(Icons.add),
            label: const Text('Add host'),
          ),
          const SizedBox(height: 24),
          const Text(
            'Connected Devices',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          if (AppSettings.instance.activeHost == null)
            const Text('No target host configured')
          else
            Card(
              child: ExpansionTile(
                title: const Text(
                  'Client connections',
                  style: TextStyle(fontWeight: FontWeight.w600),
                ),
                subtitle: Text(
                  _isLoadingDevices
                      ? 'Loading...'
                      : _devicesError != null
                      ? 'Failed to load devices'
                      : _connectedDevices.isEmpty
                      ? 'No devices recorded yet'
                      : '${_connectedDevices.length} device${_connectedDevices.length == 1 ? '' : 's'}',
                ),
                children: [
                  Align(
                    alignment: Alignment.centerRight,
                    child: Padding(
                      padding: const EdgeInsets.only(right: 12),
                      child: RefreshIconButton(
                        isRefreshing: _isLoadingDevices,
                        onPressed: _loadDevices,
                        tooltip: 'Refresh devices',
                      ),
                    ),
                  ),
                  if (_isLoadingDevices)
                    const Padding(
                      padding: EdgeInsets.all(8),
                      child: Center(
                        child: SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                      ),
                    )
                  else if (_devicesError != null)
                    ListTile(
                      leading: Icon(
                        Icons.error_outline,
                        color: Theme.of(context).colorScheme.error,
                      ),
                      title: const Text('Failed to load devices'),
                      subtitle: Text(_devicesError!),
                    )
                  else if (_connectedDevices.isEmpty)
                    const ListTile(title: Text('No devices recorded yet'))
                  else
                    ..._connectedDevices.map((device) {
                      return ListTile(
                        leading: const Icon(Icons.devices),
                        title: Text(device.ipAddress),
                        subtitle: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            if (device.userAgent.isNotEmpty)
                              Text(
                                device.userAgent,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(fontSize: 12),
                              ),
                            Text(
                              '${device.requestCount} request${device.requestCount == 1 ? '' : 's'} · last seen ${_formatRelative(device.lastSeenAt)}',
                              style: const TextStyle(fontSize: 12),
                            ),
                          ],
                        ),
                        isThreeLine: device.userAgent.isNotEmpty,
                        trailing: IconButton(
                          icon: const Icon(Icons.delete_outline),
                          tooltip: 'Remove',
                          onPressed: () => _deleteDevice(device.id),
                        ),
                      );
                    }),
                ],
              ),
            ),
          const SizedBox(height: 24),

          const Text(
            'Network Drive',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          _NetworkDriveCard(host: AppSettings.instance.activeHost),

          const SizedBox(height: 24),

          const Text(
            'Software Bill of Materials',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          if (_isLoadingSbom)
            const Center(
              child: Padding(
                padding: EdgeInsets.all(16),
                child: CircularProgressIndicator(),
              ),
            )
          else ...[
            if (_sbomError != null)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Text(
                  'Failed to load some SBOM sources:\n$_sbomError',
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            if (_flutterSbom != null)
              _SbomExpansionTile(
                title: 'Flutter dependencies',
                subtitle: '${_flutterSbom!.length} packages',
                items: _flutterSbom!
                    .map(
                      (p) => _SbomEntry(
                        name: p.name,
                        version: p.version,
                        url: p.url,
                      ),
                    )
                    .toList(),
              ),
            const SizedBox(height: 8),
            if (_goSbom != null)
              _SbomExpansionTile(
                title: 'Go dependencies',
                subtitle:
                    '${_goSbom!.dependencies.length} packages · ${_goSbom!.goVersion}',
                items: _goSbom!.dependencies
                    .map((d) => _SbomEntry(name: d.path, version: d.version))
                    .toList(),
              ),
            if (_goSbom == null && _flutterSbom == null)
              const Text('No SBOM data available.'),
          ],
        ],
      ),
    );
  }

  Future<void> _signOut() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Sign out'),
        content: const Text('Are you sure you want to sign out?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Sign out'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await AuthService.logout();
    if (!mounted) return;
    if (mounted) context.go(AppRoutes.cirrus);
  }

  String _formatRelative(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inSeconds < 60) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }
}

class _SbomEntry {
  const _SbomEntry({required this.name, required this.version, this.url});
  final String name;
  final String version;
  final String? url;
}

class _SbomExpansionTile extends StatelessWidget {
  const _SbomExpansionTile({
    required this.title,
    required this.subtitle,
    required this.items,
  });

  final String title;
  final String subtitle;
  final List<_SbomEntry> items;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ExpansionTile(
        title: Text(title, style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(subtitle),
        children: items
            .map(
              (item) => ListTile(
                dense: true,
                title: Text(item.name, style: const TextStyle(fontSize: 13)),
                trailing: Text(
                  item.version,
                  style: TextStyle(
                    fontSize: 12,
                    color: Theme.of(context).colorScheme.secondary,
                    fontFamily: 'monospace',
                  ),
                ),
              ),
            )
            .toList(),
      ),
    );
  }
}

/// Shows instructions for mounting AutoButler as a network drive.
/// Fetches the butler's hostname from the health endpoint so the paths
/// reflect the device's actual LAN name rather than the connection URL.
class _NetworkDriveCard extends StatefulWidget {
  const _NetworkDriveCard({required this.host});

  final String? host;

  @override
  State<_NetworkDriveCard> createState() => _NetworkDriveCardState();
}

class _NetworkDriveCardState extends State<_NetworkDriveCard> {
  String? _hostname;

  @override
  void initState() {
    super.initState();
    _fetchHostname();
  }

  Future<void> _fetchHostname() async {
    if (AppSettings.instance.activeHost == null) return;
    try {
      final status = await HealthService.getHealth();
      if (mounted && status.hostname.isNotEmpty) {
        setState(() => _hostname = status.hostname);
      }
    } catch (_) {
      // Fall back to extracting from URL — better than nothing.
    }
  }

  String get _displayHostname {
    if (_hostname != null && _hostname!.isNotEmpty) return _hostname!;
    final h = widget.host;
    if (h == null) return 'autobutler.local';
    final uri = Uri.tryParse(h);
    return uri?.host ?? h;
  }

  String get _webdavUrl {
    final h = widget.host;
    if (h == null) return 'http://autobutler.local/webdav';
    return '$h/webdav';
  }

  @override
  Widget build(BuildContext context) {
    final hostname = _displayHostname;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Mount as network drive',
              style: TextStyle(fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 8),
            const Text(
              'Access your AutoButler files directly from your operating system\'s file browser.',
            ),
            const SizedBox(height: 16),

            // macOS
            const Text('macOS', style: TextStyle(fontWeight: FontWeight.w500)),
            const SizedBox(height: 4),
            const Text('Finder → Go → Connect to Server (⌘K), then enter:'),
            const SizedBox(height: 4),
            _CodeBlock(text: 'smb://$hostname.local'),
            const SizedBox(height: 12),

            // Windows
            const Text('Windows', style: TextStyle(fontWeight: FontWeight.w500)),
            const SizedBox(height: 4),
            const Text('File Explorer → Map network drive, then enter:'),
            const SizedBox(height: 4),
            _CodeBlock(text: '\\\\$hostname.local'),
            const SizedBox(height: 12),

            // Linux
            const Text('Linux', style: TextStyle(fontWeight: FontWeight.w500)),
            const SizedBox(height: 4),
            const Text('Files → Other Locations, or mount via terminal:'),
            const SizedBox(height: 4),
            _CodeBlock(text: 'smb://$hostname.local'),
            const SizedBox(height: 12),

            // WebDAV fallback
            const Text(
              'WebDAV (all platforms)',
              style: TextStyle(fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 4),
            const Text('Use any WebDAV client with:'),
            const SizedBox(height: 4),
            _CodeBlock(text: _webdavUrl),
            const SizedBox(height: 8),
            const Text(
              'Log in with your AutoButler username and password.',
              style: TextStyle(fontSize: 12),
            ),
          ],
        ),
      ),
    );
  }
}

class _CodeBlock extends StatelessWidget {
  const _CodeBlock({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(4),
      ),
      child: SelectableText(
        text,
        style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
      ),
    );
  }
}
