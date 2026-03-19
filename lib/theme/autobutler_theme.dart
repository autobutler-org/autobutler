import 'package:flutter/material.dart';

import 'autobutler_colors.dart';

abstract final class AutobutlerTheme {
  static ThemeData dark() {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: AutobutlerColors.primary,
      brightness: Brightness.dark,
      surface: AutobutlerColors.card,
      onSurface: AutobutlerColors.foreground,
      primary: AutobutlerColors.primary,
      onPrimary: AutobutlerColors.primaryForeground,
      secondary: AutobutlerColors.sidebar,
      onSecondary: AutobutlerColors.secondaryForeground,
      error: AutobutlerColors.error,
      onError: AutobutlerColors.errorForeground,
      outline: AutobutlerColors.border,
      outlineVariant: AutobutlerColors.border,
    );

    return ThemeData(
      brightness: Brightness.dark,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: AutobutlerColors.background,
      useMaterial3: true,

      appBarTheme: const AppBarTheme(
        backgroundColor: AutobutlerColors.sidebar,
        foregroundColor: AutobutlerColors.foreground,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
      ),

      cardTheme: CardThemeData(
        color: AutobutlerColors.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
          side: const BorderSide(color: AutobutlerColors.border),
        ),
        elevation: 0,
      ),

      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: AutobutlerColors.input,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          borderSide: const BorderSide(color: AutobutlerColors.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          borderSide: const BorderSide(color: AutobutlerColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          borderSide: const BorderSide(color: AutobutlerColors.primary),
        ),
        labelStyle: const TextStyle(color: AutobutlerColors.secondaryForeground),
        hintStyle: const TextStyle(color: AutobutlerColors.mutedForeground),
      ),

      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: AutobutlerColors.primary,
          foregroundColor: AutobutlerColors.primaryForeground,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          ),
        ),
      ),

      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: AutobutlerColors.secondaryForeground,
          side: const BorderSide(color: AutobutlerColors.border),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          ),
        ),
      ),

      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: AutobutlerColors.primary,
        ),
      ),

      dividerTheme: const DividerThemeData(
        color: AutobutlerColors.border,
        thickness: 1,
      ),

      drawerTheme: const DrawerThemeData(
        backgroundColor: AutobutlerColors.sidebar,
      ),

      listTileTheme: const ListTileThemeData(
        textColor: AutobutlerColors.foreground,
        iconColor: AutobutlerColors.secondaryForeground,
      ),

      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return AutobutlerColors.primary;
          }
          return AutobutlerColors.mutedForeground;
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return AutobutlerColors.primary.withValues(alpha: 0.3);
          }
          return AutobutlerColors.input;
        }),
      ),

      checkboxTheme: CheckboxThemeData(
        fillColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return AutobutlerColors.primary;
          }
          return Colors.transparent;
        }),
        side: const BorderSide(color: AutobutlerColors.border, width: 1.5),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusSm),
        ),
      ),

      snackBarTheme: SnackBarThemeData(
        backgroundColor: AutobutlerColors.card,
        contentTextStyle: const TextStyle(color: AutobutlerColors.foreground),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
        ),
      ),

      popupMenuTheme: PopupMenuThemeData(
        color: AutobutlerColors.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          side: const BorderSide(color: AutobutlerColors.border),
        ),
      ),

      dialogTheme: DialogThemeData(
        backgroundColor: AutobutlerColors.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
          side: const BorderSide(color: AutobutlerColors.border),
        ),
      ),

      iconTheme: const IconThemeData(
        color: AutobutlerColors.secondaryForeground,
        size: 20,
      ),
    );
  }

  static ThemeData light() {
    return ThemeData(
      colorScheme: ColorScheme.fromSeed(
        seedColor: AutobutlerColors.primary,
        brightness: Brightness.light,
      ),
      useMaterial3: true,
    );
  }
}
