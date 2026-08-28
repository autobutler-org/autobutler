import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/services/vault_service.dart';
import 'package:quark/utils/auto_refresh_mixin.dart';
import 'package:quark/utils/connection_error.dart';
import 'package:quark/utils/quark_widget.dart';
import 'package:quark/widgets/core/quark_disconnected_state.dart';
import 'package:quark/widgets/core/quark_storage_bar.dart';
import 'package:quark/widgets/layout/quark_app_bar.dart';
import 'package:quark/widgets/quark_drawer.dart';
import 'package:quark/widgets/refresh_icon_button.dart';
import 'package:quark_icons/quark_icons.dart';

class StorageDevicesPage extends StatefulWidget {
  const StorageDevicesPage({super.key});

  @override
  State<StorageDevicesPage> createState() => _StorageDevicesPageState();
}

class _StorageDevicesPageState extends State<StorageDevicesPage>
    with WidgetsBindingObserver, AutoRefreshMixin {
  List<StorageDevice>? _devices;

  /// The thrown object, not its message — the render decides whether it means
  /// "your Quark is unreachable" or "the request failed" (#1637).
  Object? _error;
  final Set<String> _mounting = {};
  String? _activeBackupJobId;
  BackupJobStatus? _backupStatus;
  Timer? _pollTimer;
  String _vaultDeviceSerial = '';

  @override
  Future<void> refresh() async {
    if (AppSettings.instance.activeHost == null) {
      setState(() {
        _devices = null;
        _error = null;
      });
      return;
    }
    try {
      final results = await Future.wait([
        StorageService.listDevices(),
        VaultService.getStorageLocation()
            .then((loc) => loc.deviceSerial)
            .catchError((_) => ''),
      ]);
      if (!mounted) return;
      setState(() {
        _devices = results[0] as List<StorageDevice>;
        _vaultDeviceSerial = results[1] as String;
        _error = null;
      });
    } catch (e) {
      debugPrint('[storage_devices_page.dart] Error: $e');
      if (!mounted) return;
      setState(() => _error = e);
    }
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    super.dispose();
  }

  Future<void> _mountDevice(StorageDevice device) async {
    setState(() => _mounting.add(device.serial));
    try {
      await StorageService.mountDevice(device.serial);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('${device.name} mounted successfully')),
      );
      await refresh();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Mount failed: $e')));
    } finally {
      if (mounted) setState(() => _mounting.remove(device.serial));
    }
  }

  Future<void> _showRoleDialog(StorageDevice device) async {
    final result = await showDialog<String>(
      context: context,
      builder: (ctx) => _RoleDialog(currentRole: device.role),
    );
    if (result == null || result == device.role || !mounted) return;

    final creds = await _promptCredentials();
    if (creds == null || !mounted) return;

    try {
      await StorageService.setDeviceRole(
        serial: device.serial,
        role: result,
        username: creds['username']!,
        password: creds['password']!,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Role set to ${_roleLabel(result)}')),
      );
      await refresh();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Failed: $e')));
    }
  }

  Future<Map<String, String>?> _promptCredentials() async {
    final usernameCtrl = TextEditingController();
    final passwordCtrl = TextEditingController();
    final result = await QuarkWidget.showDialog<Map<String, String>>(
      context,
      builder: (ctx) => QuarkWidget.alertDialog(
        title: const Text('Authenticate'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('Enter your credentials to continue.'),
            const SizedBox(height: 12),
            TextField(
              controller: usernameCtrl,
              decoration: const InputDecoration(
                labelText: 'Username',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: passwordCtrl,
              obscureText: true,
              decoration: const InputDecoration(
                labelText: 'Password',
                border: OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, {
              'username': usernameCtrl.text,
              'password': passwordCtrl.text,
            }),
            child: const Text('Continue'),
          ),
        ],
      ),
    );
    usernameCtrl.dispose();
    passwordCtrl.dispose();
    return result;
  }

  Future<void> _startBackup(StorageDevice target) async {
    final creds = await _promptCredentials();
    if (creds == null || !mounted) return;

    final recoveryCtrl = TextEditingController();
    final includeVault = await QuarkWidget.showDialog<bool>(
      context,
      builder: (ctx) => QuarkWidget.alertDialog(
        title: const Text('Snapshot Backup'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Back up all devices to ${target.name.isNotEmpty ? target.name : "this drive"}.',
            ),
            const SizedBox(height: 16),
            const Text(
              'Optionally include your vault with a recovery password:',
              style: TextStyle(fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: recoveryCtrl,
              obscureText: true,
              decoration: const InputDecoration(
                labelText: 'Recovery password (optional)',
                helperText: 'Min 8 characters. Separate from master password.',
                border: OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Start Backup'),
          ),
        ],
      ),
    );
    if (includeVault != true || !mounted) {
      recoveryCtrl.dispose();
      return;
    }

    try {
      final jobId = await StorageService.startSnapshotBackup(
        targetDeviceSerial: target.serial,
        username: creds['username'],
        password: creds['password'],
        recoveryPassword: recoveryCtrl.text.isNotEmpty
            ? recoveryCtrl.text
            : null,
      );
      recoveryCtrl.dispose();
      if (!mounted) return;
      setState(() => _activeBackupJobId = jobId);
      _startPolling();
    } catch (e) {
      recoveryCtrl.dispose();
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Backup failed: $e')));
    }
  }

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(
      const Duration(seconds: 2),
      (_) => _pollStatus(),
    );
  }

  Future<void> _pollStatus() async {
    final jobId = _activeBackupJobId;
    if (jobId == null) return;
    try {
      final status = await StorageService.getSnapshotBackupStatus(jobId);
      if (!mounted) return;
      setState(() => _backupStatus = status);
      if (!status.isRunning) {
        _pollTimer?.cancel();
        _pollTimer = null;
        if (status.isComplete) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Backup completed successfully')),
          );
        } else if (status.isFailed) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(
                'Backup failed: ${status.errorMsg ?? "unknown error"}',
              ),
            ),
          );
        }
        await refresh();
      }
    } catch (e) {
      debugPrint('[storage_devices_page.dart] Poll error: $e');
    }
  }

  Future<void> _verifyBackup(StorageDevice device) async {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Verifying backup integrity...')),
    );
    try {
      final result = await StorageService.verifySnapshotBackup(
        deviceSerial: device.serial,
        full: true,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            result.isHealthy
                ? 'Backup verified: ${result.ok} files OK'
                : 'Issues found: ${result.missing.length} missing, ${result.corrupted.length} corrupted',
          ),
          duration: const Duration(seconds: 4),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Verify failed: $e')));
    }
  }

  static String _roleLabel(String role) {
    switch (role) {
      case 'default-storage':
        return 'Default Storage';
      case 'snapshot-backup':
        return 'Snapshot Backup';
      default:
        return 'Unassigned';
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: QuarkAppBar(
        label: 'Devices',
        icon: QuarkIcons.device_hub_outlined,
        actions: [
          RefreshIconButton(
            isRefreshing: isRefreshing,
            onPressed: manualRefresh,
          ),
        ],
      ),
      drawer: QuarkDrawer(
        activeSection: QuarkDrawerSection.devices,
        onTapFiles: () => context.go(AppRoutes.files),
        onTapPhotos: () => context.go(AppRoutes.photos),
        onTapDocs: () => context.go(AppRoutes.docs),
        onTapSheets: () => context.go(AppRoutes.sheets),
        onTapDevices: () => Navigator.of(context).pop(),
        onTapHealth: () => context.go(AppRoutes.health),
        onTapVault: () => context.go(AppRoutes.vault),
        onTapSettings: () => context.go(AppRoutes.settings),
      ),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (AppSettings.instance.activeHost == null) {
      return const Center(child: Text('No quark host configured.'));
    }
    final error = _error;
    if (error != null) {
      if (isQuarkUnreachableError(error)) {
        return QuarkDisconnectedView(
          onRetry: manualRefresh,
          onManageHosts: () => context.go(AppRoutes.settings),
        );
      }
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Error: $error',
            style: const TextStyle(color: Colors.red),
          ),
        ),
      );
    }
    if (_devices == null) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_devices!.isEmpty) {
      return const Center(child: Text('No storage devices detected.'));
    }
    final widgets = <Widget>[];

    if (_backupStatus != null && _activeBackupJobId != null) {
      widgets.add(_BackupProgressCard(status: _backupStatus!));
    }

    for (final device in _devices!) {
      final isVaultDevice =
          (_vaultDeviceSerial.isEmpty && device.isInternal) ||
          (_vaultDeviceSerial.isNotEmpty &&
              device.serial == _vaultDeviceSerial);
      widgets.add(
        _DeviceCard(
          device: device,
          isMounting: _mounting.contains(device.serial),
          isVaultDevice: isVaultDevice,
          onMount: device.serial.isNotEmpty ? () => _mountDevice(device) : null,
          onSetRole: device.serial.isNotEmpty
              ? () => _showRoleDialog(device)
              : null,
          onBackup: device.role == 'snapshot-backup'
              ? () => _startBackup(device)
              : null,
          onVerify: device.role == 'snapshot-backup'
              ? () => _verifyBackup(device)
              : null,
          isBackupRunning: _backupStatus?.isRunning == true,
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: refresh,
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: widgets.length,
        separatorBuilder: (context, index) => const SizedBox(height: 12),
        itemBuilder: (context, i) => widgets[i],
      ),
    );
  }
}

class _DeviceCard extends StatelessWidget {
  const _DeviceCard({
    required this.device,
    required this.isMounting,
    this.isVaultDevice = false,
    this.onMount,
    this.onSetRole,
    this.onBackup,
    this.onVerify,
    this.isBackupRunning = false,
  });

  final StorageDevice device;
  final bool isMounting;
  final bool isVaultDevice;
  final VoidCallback? onMount;
  final VoidCallback? onSetRole;
  final VoidCallback? onBackup;
  final VoidCallback? onVerify;
  final bool isBackupRunning;

  static const _categoryColors = <String, Color>{
    'documents': Color(0xFF4A90D9),
    'media': Color(0xFF7CB342),
    'backups': Color(0xFFFF8F00),
    'other': Color(0xFF9E9E9E),
    'system': Color(0xFFAB47BC),
  };

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final usedPct = device.usedPercent.clamp(0.0, 100.0) / 100.0;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header row: name + status badge
            Row(
              children: [
                Icon(
                  device.isInternal
                      ? QuarkIcons.computer_outlined
                      : QuarkIcons.usb_outlined,
                  size: 20,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    device.name.isNotEmpty ? device.name : device.mountPoint,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (isVaultDevice) _vaultBadge(),
                if (device.role != 'unassigned') _roleBadge(device.role),
                if (device.role == 'unassigned' && device.isEnabled)
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.green.shade100,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      'Enabled',
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.green.shade800,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 6),

            // Mount point + filesystem
            Text(
              '${device.mountPoint}  ·  ${device.fileSystem}',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            if (device.model.isNotEmpty) ...[
              const SizedBox(height: 2),
              Text(
                device.model,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],

            // Storage bar (only when totalBytes is known)
            if (device.totalBytes > 0) ...[
              const SizedBox(height: 12),
              QuarkStorageBar(usedFraction: usedPct),
              const SizedBox(height: 4),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(device.usedDisplay, style: theme.textTheme.bodySmall),
                  Text(
                    '${device.usedPercent.toStringAsFixed(0)}% used',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ],

            // Category chips
            if (device.categories.isNotEmpty) ...[
              const SizedBox(height: 10),
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: device.categories.entries.map((entry) {
                  final color =
                      _categoryColors[entry.key] ?? const Color(0xFF9E9E9E);
                  return Chip(
                    avatar: CircleAvatar(backgroundColor: color, radius: 6),
                    label: Text(
                      '${_capitalize(entry.key)} · ${StorageDevice.formatBytes(entry.value)}',
                    ),
                    labelStyle: theme.textTheme.bodySmall,
                    padding: const EdgeInsets.symmetric(horizontal: 4),
                    materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    visualDensity: VisualDensity.compact,
                  );
                }).toList(),
              ),
            ],

            // Mount button for unmounted USB devices
            if (!device.isEnabled && onMount != null) ...[
              const SizedBox(height: 12),
              OutlinedButton.icon(
                onPressed: isMounting ? null : onMount,
                icon: isMounting
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(QuarkIcons.link_outlined, size: 16),
                label: Text(isMounting ? 'Mounting…' : 'Mount'),
              ),
            ],

            // Role + backup actions for enabled external devices
            if (device.isEnabled && !device.isInternal) ...[
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  if (onSetRole != null)
                    OutlinedButton.icon(
                      onPressed: onSetRole,
                      icon: const Icon(QuarkIcons.label_outline, size: 16),
                      label: const Text('Set Role'),
                    ),
                  if (onBackup != null)
                    FilledButton.icon(
                      onPressed: isBackupRunning ? null : onBackup,
                      icon: const Icon(QuarkIcons.backup_outlined, size: 16),
                      label: const Text('Back Up'),
                    ),
                  if (onVerify != null)
                    OutlinedButton.icon(
                      onPressed: isBackupRunning ? null : onVerify,
                      icon: const Icon(QuarkIcons.verified_outlined, size: 16),
                      label: const Text('Verify'),
                    ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }

  static String _capitalize(String s) =>
      s.isEmpty ? s : s[0].toUpperCase() + s.substring(1);

  static Widget _vaultBadge() {
    return Container(
      margin: const EdgeInsets.only(right: 4),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: Colors.purple.shade100,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        'Vault',
        style: TextStyle(
          fontSize: 12,
          color: Colors.purple.shade800,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }

  static Widget _roleBadge(String role) {
    final Color bg;
    final Color fg;
    final String label;
    switch (role) {
      case 'default-storage':
        bg = Colors.blue.shade100;
        fg = Colors.blue.shade800;
        label = 'Default Storage';
      case 'snapshot-backup':
        bg = Colors.orange.shade100;
        fg = Colors.orange.shade800;
        label = 'Snapshot Backup';
      default:
        return const SizedBox.shrink();
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        label,
        style: TextStyle(fontSize: 12, color: fg, fontWeight: FontWeight.w500),
      ),
    );
  }
}

class _RoleDialog extends StatelessWidget {
  final String currentRole;
  const _RoleDialog({required this.currentRole});

  @override
  Widget build(BuildContext context) {
    return SimpleDialog(
      title: const Text('Set Device Role'),
      children: [
        _roleOption(
          context,
          'default-storage',
          'Default Storage',
          'Always-connected drive for daily file storage',
        ),
        _roleOption(
          context,
          'snapshot-backup',
          'Snapshot Backup',
          'Plug in, back up everything, unplug',
        ),
        _roleOption(context, 'unassigned', 'Unassigned', 'No special role'),
      ],
    );
  }

  Widget _roleOption(
    BuildContext context,
    String value,
    String title,
    String subtitle,
  ) {
    final selected = value == currentRole;
    return ListTile(
      leading: Icon(
        selected
            ? QuarkIcons.radio_button_checked
            : QuarkIcons.radio_button_unchecked,
      ),
      title: Text(title),
      subtitle: Text(subtitle, style: const TextStyle(fontSize: 12)),
      onTap: () => Navigator.pop(context, value),
    );
  }
}

class _BackupProgressCard extends StatelessWidget {
  final BackupJobStatus status;
  const _BackupProgressCard({required this.status});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final pct = (status.progress * 100).toStringAsFixed(0);

    String statusText;
    switch (status.status) {
      case 'PENDING':
        statusText = 'Preparing backup...';
      case 'SCANNING':
        statusText = 'Scanning files...';
      case 'COPYING':
        statusText = 'Copying files ($pct%)';
      case 'COMPLETED':
        statusText = 'Backup complete';
      case 'FAILED':
        statusText = 'Backup failed';
      default:
        statusText = status.status;
    }

    return Card(
      color: status.isFailed
          ? theme.colorScheme.errorContainer
          : theme.colorScheme.primaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  status.isRunning
                      ? QuarkIcons.backup_outlined
                      : status.isComplete
                      ? QuarkIcons.check_circle_outline
                      : QuarkIcons.error_outline,
                  size: 20,
                ),
                const SizedBox(width: 8),
                Text(statusText, style: theme.textTheme.titleSmall),
              ],
            ),
            if (status.isRunning) ...[
              const SizedBox(height: 12),
              LinearProgressIndicator(value: status.progress),
              const SizedBox(height: 8),
              Text(
                '${status.filesCopied} / ${status.totalFiles} files  ·  '
                '${StorageDevice.formatBytes(status.bytesCopied)} / '
                '${StorageDevice.formatBytes(status.totalBytes)}',
                style: theme.textTheme.bodySmall,
              ),
            ],
            if (status.isComplete) ...[
              const SizedBox(height: 8),
              Text(
                '${status.filesCopied} files copied, '
                '${status.filesSkipped} skipped  ·  '
                '${StorageDevice.formatBytes(status.bytesCopied)}',
                style: theme.textTheme.bodySmall,
              ),
            ],
            if (status.isFailed && status.errorMsg != null) ...[
              const SizedBox(height: 8),
              Text(
                status.errorMsg!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
