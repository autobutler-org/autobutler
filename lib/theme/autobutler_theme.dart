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
        labelStyle: const TextStyle(
          color: AutobutlerColors.secondaryForeground,
        ),
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
        style: TextButton.styleFrom(foregroundColor: AutobutlerColors.primary),
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
    const lightBackground = Color(0xFFF8FAFC);
    const lightCard = Color(0xFFFFFFFF);
    const lightSidebar = Color(0xFFF1F5F9);
    const lightBorder = Color(0xFFE2E8F0);
    const lightInput = Color(0xFFFFFFFF);
    const lightForeground = Color(0xFF0F172A);
    const lightMutedForeground = Color(0xFF64748B);
    const lightSecondaryForeground = Color(0xFF475569);
    const lightPrimary = AutobutlerColors.primary;
    const lightPrimaryForeground = Color(0xFFFFFFFF);
    const lightError = Color(0xFFDC2626);
    const lightErrorForeground = Color(0xFFFFFFFF);

    final colorScheme = ColorScheme.fromSeed(
      seedColor: lightPrimary,
      brightness: Brightness.light,
      surface: lightCard,
      onSurface: lightForeground,
      primary: lightPrimary,
      onPrimary: lightPrimaryForeground,
      secondary: lightSidebar,
      onSecondary: lightSecondaryForeground,
      error: lightError,
      onError: lightErrorForeground,
      outline: lightBorder,
      outlineVariant: lightBorder,
    );

    return ThemeData(
      brightness: Brightness.light,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: lightBackground,
      useMaterial3: true,

      appBarTheme: const AppBarTheme(
        backgroundColor: lightSidebar,
        foregroundColor: lightForeground,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
      ),

      cardTheme: CardThemeData(
        color: lightCard,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
          side: const BorderSide(color: lightBorder),
        ),
        elevation: 0,
      ),

      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: lightInput,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          borderSide: const BorderSide(color: lightBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          borderSide: const BorderSide(color: lightBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          borderSide: const BorderSide(color: lightPrimary),
        ),
        labelStyle: const TextStyle(color: lightSecondaryForeground),
        hintStyle: const TextStyle(color: lightMutedForeground),
      ),

      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: lightPrimary,
          foregroundColor: lightPrimaryForeground,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          ),
        ),
      ),

      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: lightSecondaryForeground,
          side: const BorderSide(color: lightBorder),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          ),
        ),
      ),

      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(foregroundColor: lightPrimary),
      ),

      dividerTheme: const DividerThemeData(color: lightBorder, thickness: 1),

      drawerTheme: const DrawerThemeData(backgroundColor: lightSidebar),

      listTileTheme: const ListTileThemeData(
        textColor: lightForeground,
        iconColor: lightSecondaryForeground,
      ),

      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return lightPrimary;
          return lightMutedForeground;
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return lightPrimary.withValues(alpha: 0.3);
          }
          return lightBorder;
        }),
      ),

      checkboxTheme: CheckboxThemeData(
        fillColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return lightPrimary;
          return Colors.transparent;
        }),
        side: const BorderSide(color: lightBorder, width: 1.5),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusSm),
        ),
      ),

      snackBarTheme: SnackBarThemeData(
        backgroundColor: lightCard,
        contentTextStyle: const TextStyle(color: lightForeground),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
        ),
      ),

      popupMenuTheme: PopupMenuThemeData(
        color: lightCard,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          side: const BorderSide(color: lightBorder),
        ),
      ),

      dialogTheme: DialogThemeData(
        backgroundColor: lightCard,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
          side: const BorderSide(color: lightBorder),
        ),
      ),

      iconTheme: const IconThemeData(color: lightSecondaryForeground, size: 20),
    );
  }
}
