import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/services/vault_service.dart';
import 'package:quark/utils/auto_refresh_mixin.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/quark_widget.dart';
import 'package:quark/widgets/storage_devices/credentials_dialog.dart';
import 'package:quark/widgets/storage_devices/role_dialog.dart';
import 'package:quark/widgets/storage_devices/snapshot_backup_dialog.dart';
import 'package:quark/widgets/storage_devices/storage_devices_body.dart';

/// The System page's Storage tab: the drives the Quark can see, their status
/// and role, and the actions to mount, rename or back them up.
class StorageTab extends StatefulWidget {
  const StorageTab({this.onRefreshingChanged, super.key});

  /// Called with whether a refresh is in flight whenever that changes, so
  /// the page's app bar can show it.
  final ValueChanged<bool>? onRefreshingChanged;

  @override
  State<StorageTab> createState() => _StorageTabState();
}

class _StorageTabState extends State<StorageTab>
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
  void didChangeRefreshing() => widget.onRefreshingChanged?.call(isRefreshing);

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
      debugPrint('[storage_tab.dart] Error: $e');
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
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'mount the drive'))),
      );
    } finally {
      if (mounted) setState(() => _mounting.remove(device.serial));
    }
  }

  Future<void> _showRoleDialog(StorageDevice device) async {
    final result = await showDialog<String>(
      context: context,
      builder: (ctx) => RoleDialog(currentRole: device.role),
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
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'set the drive role'))),
      );
    }
  }

  Future<Map<String, String>?> _promptCredentials() =>
      QuarkWidget.showDialog<Map<String, String>>(
        context,
        builder: (ctx) => const CredentialsDialog(),
      );

  Future<void> _startBackup(StorageDevice target) async {
    final creds = await _promptCredentials();
    if (creds == null || !mounted) return;

    final recoveryPassword = await QuarkWidget.showDialog<String>(
      context,
      builder: (ctx) => SnapshotBackupDialog(targetName: target.name),
    );
    if (recoveryPassword == null || !mounted) return;

    try {
      final jobId = await StorageService.startSnapshotBackup(
        targetDeviceSerial: target.serial,
        username: creds['username'],
        password: creds['password'],
        recoveryPassword: recoveryPassword.isNotEmpty ? recoveryPassword : null,
      );
      if (!mounted) return;
      setState(() => _activeBackupJobId = jobId);
      _startPolling();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'start the backup'))),
      );
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
      debugPrint('[storage_tab.dart] Poll error: $e');
      // The job is a row every instance of the Quark reads, so a 404 means
      // it is gone for good and another poll will not find it. Anything else
      // may be one instance's bad moment: keep polling.
      if (e is! ApiException || e.statusCode != 404 || !mounted) return;
      _pollTimer?.cancel();
      _pollTimer = null;
      setState(() {
        _activeBackupJobId = null;
        _backupStatus = null;
      });
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'check the backup'))),
      );
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
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'verify the backup'))),
      );
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
    // Every drive action is admin-only on the Quark (#1899), so anyone else
    // gets the list without them (#1928).
    return ValueListenableBuilder<bool>(
      valueListenable: AppSettings.instance.isAdmin,
      builder: (context, isAdmin, _) => StorageDevicesBody(
        devices: _devices,
        error: _error,
        mounting: _mounting,
        vaultDeviceSerial: _vaultDeviceSerial,
        backupStatus: _backupStatus,
        activeBackupJobId: _activeBackupJobId,
        onRefresh: refresh,
        onRetry: manualRefresh,
        onManageHosts: () =>
            context.go(AppRoutes.settingsTab(SettingsTab.general)),
        onMount: isAdmin ? _mountDevice : null,
        onSetRole: isAdmin ? _showRoleDialog : null,
        onBackup: isAdmin ? _startBackup : null,
        onVerify: isAdmin ? _verifyBackup : null,
      ),
    );
  }
}
