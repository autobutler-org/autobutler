import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/widgets/autobutler_brand_button.dart';
import 'package:flutter/material.dart';

class AutobutlerAppBar extends StatelessWidget implements PreferredSizeWidget {
  const AutobutlerAppBar({
    required this.label,
    required this.icon,
    this.actions = const [],
    super.key,
  });

  final String label;
  final IconData icon;
  final List<Widget> actions;

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    return AppBar(
      leadingWidth: AutobutlerBrandButton.preferredWidth + 8,
      leading: Builder(
        builder: (ctx) => Padding(
          padding: const EdgeInsets.only(left: 8),
          child: AutobutlerBrandButton(
            label: label,
            icon: icon,
            onTap: () => Scaffold.of(ctx).openDrawer(),
          ),
        ),
      ),
      title: null,
      actions: [
        ...actions,
        ValueListenableBuilder<ThemeMode>(
          valueListenable: AppSettings.instance.themeMode,
          builder: (context, mode, _) {
            final IconData iconData;
            final String tooltip;
            final ThemeMode nextMode;

            switch (mode) {
              case ThemeMode.light:
                iconData = Icons.dark_mode;
                tooltip = 'Switch to dark mode';
                nextMode = ThemeMode.dark;
              case ThemeMode.dark:
                iconData = Icons.light_mode;
                tooltip = 'Switch to light mode';
                nextMode = ThemeMode.light;
              case ThemeMode.system:
                iconData = Icons.brightness_auto;
                tooltip = 'Switch to light mode';
                nextMode = ThemeMode.light;
            }

            return IconButton(
              icon: Icon(iconData),
              tooltip: tooltip,
              onPressed: () => AppSettings.instance.setThemeMode(nextMode),
            );
          },
        ),
      ],
    );
  }
}
