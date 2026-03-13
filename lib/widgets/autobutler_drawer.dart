import 'dart:async';

import 'package:flutter/material.dart';

enum AutobutlerDrawerSection { cirrus, photos, settings }

class AutobutlerDrawer extends StatelessWidget {
  const AutobutlerDrawer({
    super.key,
    required this.activeSection,
    this.onTapCirrus,
    this.onTapPhotos,
    this.onTapSettings,
  });

  final AutobutlerDrawerSection activeSection;
  final FutureOr<void> Function()? onTapCirrus;
  final FutureOr<void> Function()? onTapPhotos;
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
            leading: const Icon(Icons.cloud),
            title: const Text('Cirrus'),
            selected: activeSection == AutobutlerDrawerSection.cirrus,
            onTap: () {
              onTapCirrus?.call();
            },
          ),
          ListTile(
            leading: const Icon(Icons.photo),
            title: const Text('Photos'),
            selected: activeSection == AutobutlerDrawerSection.photos,
            onTap: () {
              onTapPhotos?.call();
            },
          ),
          ListTile(
            leading: const Icon(Icons.settings),
            title: const Text('Settings'),
            selected: activeSection == AutobutlerDrawerSection.settings,
            onTap: () {
              onTapSettings?.call();
            },
          ),
        ],
      ),
    );
  }
}
