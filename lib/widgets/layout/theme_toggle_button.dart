import 'package:flutter/material.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// [ThemeToggleButton] wired to [AppSettings].
///
/// The package widget takes the current mode in and hands the chosen one back
/// out, so this is where the app's setting is read and written. Drop it into
/// any [AppBar.actions] list.
class AppThemeToggle extends StatelessWidget {
  /// Creates a theme toggle backed by [AppSettings.instance].
  const AppThemeToggle({super.key});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<ThemeMode>(
      valueListenable: AppSettings.instance.themeMode,
      builder: (context, mode, _) => ThemeToggleButton(
        mode: mode,
        onChanged: AppSettings.instance.setThemeMode,
      ),
    );
  }
}
