import 'dart:async';

import 'package:flutter/material.dart';
import 'package:autobutler_icons/autobutler_icons.dart';

enum AutobutlerDrawerSection {
  cirrus,
  photos,
  docs,
  sheets,
  devices,
  health,
  vault,
  settings,
}

class AutobutlerDrawer extends StatelessWidget {
  const AutobutlerDrawer({
    super.key,
    required this.activeSection,
    this.onTapCirrus,
    this.onTapPhotos,
    this.onTapDocs,
    this.onTapSheets,
    this.onTapDevices,
    this.onTapHealth,
    this.onTapVault,
    this.onTapSettings,
  });

  final AutobutlerDrawerSection activeSection;
  final FutureOr<void> Function()? onTapCirrus;
  final FutureOr<void> Function()? onTapPhotos;
  final FutureOr<void> Function()? onTapDocs;
  final FutureOr<void> Function()? onTapSheets;
  final FutureOr<void> Function()? onTapDevices;
  final FutureOr<void> Function()? onTapHealth;
  final FutureOr<void> Function()? onTapVault;
  final FutureOr<void> Function()? onTapSettings;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Drawer(
      child: ListView(
        padding: EdgeInsets.zero,
        children: [
          DrawerHeader(
            decoration: BoxDecoration(color: theme.colorScheme.primary),
            child: Text(
              'Autobutler',
              style: theme.textTheme.titleLarge?.copyWith(
                color: theme.colorScheme.onPrimary,
              ),
            ),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.storage_rounded),
            title: const Text('Files'),
            selected: activeSection == AutobutlerDrawerSection.cirrus,
            onTap: () => onTapCirrus?.call(),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.photo_library_outlined),
            title: const Text('Photos'),
            selected: activeSection == AutobutlerDrawerSection.photos,
            onTap: () => onTapPhotos?.call(),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.description_outlined),
            title: const Text('Docs'),
            selected: activeSection == AutobutlerDrawerSection.docs,
            onTap: () => onTapDocs?.call(),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.table_chart_outlined),
            title: const Text('Sheets'),
            selected: activeSection == AutobutlerDrawerSection.sheets,
            onTap: () => onTapSheets?.call(),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.device_hub_outlined),
            title: const Text('Devices'),
            selected: activeSection == AutobutlerDrawerSection.devices,
            onTap: () => onTapDevices?.call(),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.monitor_heart_outlined),
            title: const Text('Health'),
            selected: activeSection == AutobutlerDrawerSection.health,
            onTap: () => onTapHealth?.call(),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.lock_outline),
            title: const Text('Vault'),
            selected: activeSection == AutobutlerDrawerSection.vault,
            onTap: () => onTapVault?.call(),
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.settings_outlined),
            title: const Text('Settings'),
            selected: activeSection == AutobutlerDrawerSection.settings,
            onTap: () => onTapSettings?.call(),
          ),
        ],
      ),
    );
  }
}
