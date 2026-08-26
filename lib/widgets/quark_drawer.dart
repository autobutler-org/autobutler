import 'dart:async';

import 'package:flutter/material.dart';
import 'package:quark/models/plugin_manifest.dart';
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
  plugins,
  plugin,
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
    this.onTapPlugins,
    this.plugins = const [],
    this.activePluginId,
    this.onTapPlugin,
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
  final FutureOr<void> Function()? onTapPlugins;

  /// Plugin manifests to append to the drawer after the built-in items.
  final List<PluginManifest> plugins;

  /// The ID of the currently active plugin, if any.
  final String? activePluginId;

  /// Called when a plugin nav item is tapped.
  final void Function(PluginManifest plugin)? onTapPlugin;

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
          ListTile(
            leading: const Icon(Icons.extension_outlined),
            title: const Text('Plugins'),
            selected: activeSection == QuarkDrawerSection.plugins,
            onTap: () => onTapPlugins?.call(),
          ),
          // Installed plugin nav items, appended after the built-in ones.
          for (final plugin in plugins)
            if (plugin.contributes.navItem != null)
              ListTile(
                leading: Icon(iconFromName(plugin.contributes.navItem!.icon)),
                title: Text(plugin.contributes.navItem!.label),
                selected:
                    activeSection == QuarkDrawerSection.plugin &&
                    activePluginId == plugin.id,
                onTap: () => onTapPlugin?.call(plugin),
              ),
        ],
      ),
    );
  }

  /// Resolves a Material icon by name string.
  ///
  /// Single source of truth for icon name -> [IconData]; both this drawer and
  /// [PluginRenderer] use it. Unknown names fall back to [Icons.extension].
  static IconData iconFromName(String name) {
    const map = <String, IconData>{
      'waving_hand': Icons.waving_hand,
      'extension': Icons.extension,
      'download': Icons.download,
      'settings': Icons.settings,
      'folder': Icons.folder,
      'photo': Icons.photo,
      'health': Icons.monitor_heart_outlined,
      'home': Icons.home,
      'star': Icons.star,
      'info': Icons.info_outline,
      'check': Icons.check_circle_outline,
      'warning': Icons.warning_amber_outlined,
      'error': Icons.error_outline,
    };
    return map[name] ?? Icons.extension;
  }
}
