import 'package:flutter/material.dart';

/// An [IconButton] that switches the app between light and dark.
///
/// The current [mode] comes in and the chosen one goes out; the package never
/// reads or writes the app's settings. From [ThemeMode.system] the button
/// commits to light, because the first tap is a user saying they want the
/// other one, not the one they are already looking at.
///
/// Key prefixes: `theme_toggle` on the control itself.
///
/// ```dart
/// ThemeToggleButton(
///   mode: settings.themeMode.value,
///   onChanged: settings.setThemeMode,
/// );
/// ```
class ThemeToggleButton extends StatelessWidget {
  /// Creates a toggle showing the alternative to [mode].
  const ThemeToggleButton({
    required this.mode,
    required this.onChanged,
    super.key,
  });

  /// The theme mode currently in effect, which decides the glyph and tooltip.
  final ThemeMode mode;

  /// Called with the mode the user asked for. Never called with [mode].
  final ValueChanged<ThemeMode> onChanged;

  @override
  Widget build(BuildContext context) {
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
      key: const ValueKey('theme_toggle'),
      icon: Icon(iconData),
      tooltip: tooltip,
      onPressed: () => onChanged(nextMode),
    );
  }
}
