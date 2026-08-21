import 'package:flutter/material.dart';

import 'quark_colors.dart';

abstract final class QuarkTheme {
  static ThemeData dark() {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: QuarkColors.primary,
      brightness: Brightness.dark,
      surface: QuarkColors.card,
      onSurface: QuarkColors.foreground,
      primary: QuarkColors.primary,
      onPrimary: QuarkColors.primaryForeground,
      secondary: QuarkColors.sidebar,
      onSecondary: QuarkColors.secondaryForeground,
      error: QuarkColors.error,
      onError: QuarkColors.errorForeground,
      outline: QuarkColors.border,
      outlineVariant: QuarkColors.border,
    );

    return ThemeData(
      brightness: Brightness.dark,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: QuarkColors.background,
      useMaterial3: true,

      appBarTheme: const AppBarTheme(
        backgroundColor: QuarkColors.sidebar,
        foregroundColor: QuarkColors.foreground,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
      ),

      cardTheme: CardThemeData(
        color: QuarkColors.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
          side: const BorderSide(color: QuarkColors.border),
        ),
        elevation: 0,
      ),

      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: QuarkColors.input,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          borderSide: const BorderSide(color: QuarkColors.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          borderSide: const BorderSide(color: QuarkColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          borderSide: const BorderSide(color: QuarkColors.primary),
        ),
        labelStyle: const TextStyle(color: QuarkColors.secondaryForeground),
        hintStyle: const TextStyle(color: QuarkColors.mutedForeground),
      ),

      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: QuarkColors.primary,
          foregroundColor: QuarkColors.primaryForeground,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          ),
        ),
      ),

      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: QuarkColors.secondaryForeground,
          side: const BorderSide(color: QuarkColors.border),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          ),
        ),
      ),

      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(foregroundColor: QuarkColors.primary),
      ),

      dividerTheme: const DividerThemeData(
        color: QuarkColors.border,
        thickness: 1,
      ),

      drawerTheme: const DrawerThemeData(backgroundColor: QuarkColors.sidebar),

      listTileTheme: const ListTileThemeData(
        textColor: QuarkColors.foreground,
        iconColor: QuarkColors.secondaryForeground,
      ),

      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return QuarkColors.primary;
          }
          return QuarkColors.mutedForeground;
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return QuarkColors.primary.withValues(alpha: 0.3);
          }
          return QuarkColors.input;
        }),
      ),

      checkboxTheme: CheckboxThemeData(
        fillColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return QuarkColors.primary;
          }
          return Colors.transparent;
        }),
        side: const BorderSide(color: QuarkColors.border, width: 1.5),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusSm),
        ),
      ),

      snackBarTheme: SnackBarThemeData(
        backgroundColor: QuarkColors.card,
        contentTextStyle: const TextStyle(color: QuarkColors.foreground),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
        ),
      ),

      popupMenuTheme: PopupMenuThemeData(
        color: QuarkColors.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          side: const BorderSide(color: QuarkColors.border),
        ),
      ),

      dialogTheme: DialogThemeData(
        backgroundColor: QuarkColors.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
          side: const BorderSide(color: QuarkColors.border),
        ),
      ),

      iconTheme: const IconThemeData(
        color: QuarkColors.secondaryForeground,
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
    const lightPrimary = QuarkColors.primary;
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
          borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
          side: const BorderSide(color: lightBorder),
        ),
        elevation: 0,
      ),

      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: lightInput,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          borderSide: const BorderSide(color: lightBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          borderSide: const BorderSide(color: lightBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
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
            borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          ),
        ),
      ),

      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: lightSecondaryForeground,
          side: const BorderSide(color: lightBorder),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
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
          borderRadius: BorderRadius.circular(QuarkColors.radiusSm),
        ),
      ),

      snackBarTheme: SnackBarThemeData(
        backgroundColor: lightCard,
        contentTextStyle: const TextStyle(color: lightForeground),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
        ),
      ),

      popupMenuTheme: PopupMenuThemeData(
        color: lightCard,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          side: const BorderSide(color: lightBorder),
        ),
      ),

      dialogTheme: DialogThemeData(
        backgroundColor: lightCard,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
          side: const BorderSide(color: lightBorder),
        ),
      ),

      iconTheme: const IconThemeData(color: lightSecondaryForeground, size: 20),
    );
  }
}
