import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/material.dart';

/// A standalone [IconButton] that cycles the app theme between light and dark,
/// backed by [AppSettings]. Drop it into any [AppBar.actions] list.
class ThemeToggleButton extends StatelessWidget {
  const ThemeToggleButton({super.key});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<ThemeMode>(
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
    );
  }
}
