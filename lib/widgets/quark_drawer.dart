import 'dart:async';

import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

enum QuarkDrawerSection {
  cirrus,
  photos,
  docs,
  sheets,
  devices,
  health,
  vault,
  settings,
}

class QuarkDrawer extends StatelessWidget {
  const QuarkDrawer({
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

  final QuarkDrawerSection activeSection;
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
              'Quark',
              style: theme.textTheme.titleLarge?.copyWith(
                color: theme.colorScheme.onPrimary,
              ),
            ),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.storage_rounded),
            title: const Text('Files'),
            selected: activeSection == QuarkDrawerSection.cirrus,
            onTap: () => onTapCirrus?.call(),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.photo_library_outlined),
            title: const Text('Photos'),
            selected: activeSection == QuarkDrawerSection.photos,
            onTap: () => onTapPhotos?.call(),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.description_outlined),
            title: const Text('Docs'),
            selected: activeSection == QuarkDrawerSection.docs,
            onTap: () => onTapDocs?.call(),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.table_chart_outlined),
            title: const Text('Sheets'),
            selected: activeSection == QuarkDrawerSection.sheets,
            onTap: () => onTapSheets?.call(),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.device_hub_outlined),
            title: const Text('Devices'),
            selected: activeSection == QuarkDrawerSection.devices,
            onTap: () => onTapDevices?.call(),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.monitor_heart_outlined),
            title: const Text('Health'),
            selected: activeSection == QuarkDrawerSection.health,
            onTap: () => onTapHealth?.call(),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.lock_outline),
            title: const Text('Vault'),
            selected: activeSection == QuarkDrawerSection.vault,
            onTap: () => onTapVault?.call(),
          ),
          ListTile(
            leading: const Icon(QuarkIcons.settings_outlined),
            title: const Text('Settings'),
            selected: activeSection == QuarkDrawerSection.settings,
            onTap: () => onTapSettings?.call(),
          ),
        ],
      ),
    );
  }
}
