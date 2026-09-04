import 'package:flutter/material.dart';

import 'quark_tokens.dart';

/// Builds Quark's [ThemeData] from a [QuarkTokens] set.
///
/// The tokens are attached to the result as a [ThemeExtension], so a widget can
/// reach values the Material [ColorScheme] has no slot for — `sidebar`,
/// `warning`, `success`, the spacing scale — with [QuarkTokens.of].
///
/// ```dart
/// MaterialApp(
///   theme: QuarkTheme.light(),
///   darkTheme: QuarkTheme.dark(),
/// );
/// ```
abstract final class QuarkTheme {
  /// Quark's dark theme, built from [QuarkTokens.dark].
  static ThemeData dark() => from(QuarkTokens.dark, Brightness.dark);

  /// Quark's light theme, built from [QuarkTokens.light].
  static ThemeData light() => from(QuarkTokens.light, Brightness.light);

  /// Builds a [ThemeData] for [brightness] out of [tokens].
  ///
  /// Every color, radius, and border in the returned theme comes from [tokens],
  /// which is what lets the widget gallery's theme panel restyle the whole app
  /// from edited values.
  static ThemeData from(QuarkTokens tokens, Brightness brightness) {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: tokens.primary,
      brightness: brightness,
      surface: tokens.card,
      onSurface: tokens.foreground,
      primary: tokens.primary,
      onPrimary: tokens.primaryForeground,
      secondary: tokens.sidebar,
      onSecondary: tokens.secondaryForeground,
      error: tokens.error,
      onError: tokens.errorForeground,
      outline: tokens.border,
      outlineVariant: tokens.border,
    );

    // The recessive track behind an off switch: the input fill reads as a well
    // on dark, but disappears on light, where the hairline is the right weight.
    final switchTrackOff = brightness == Brightness.dark
        ? tokens.input
        : tokens.border;

    return ThemeData(
      brightness: brightness,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: tokens.background,
      useMaterial3: true,
      extensions: <ThemeExtension<dynamic>>[tokens],
      appBarTheme: AppBarTheme(
        backgroundColor: tokens.sidebar,
        foregroundColor: tokens.foreground,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
      ),
      cardTheme: CardThemeData(
        color: tokens.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(tokens.radiusLg),
          side: BorderSide(color: tokens.border),
        ),
        elevation: 0,
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: tokens.input,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(tokens.radiusMd),
          borderSide: BorderSide(color: tokens.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(tokens.radiusMd),
          borderSide: BorderSide(color: tokens.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(tokens.radiusMd),
          borderSide: BorderSide(color: tokens.primary),
        ),
        labelStyle: TextStyle(color: tokens.secondaryForeground),
        hintStyle: TextStyle(color: tokens.mutedForeground),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: tokens.primary,
          foregroundColor: tokens.primaryForeground,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(tokens.radiusMd),
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: tokens.secondaryForeground,
          side: BorderSide(color: tokens.border),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(tokens.radiusMd),
          ),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(foregroundColor: tokens.primary),
      ),
      dividerTheme: DividerThemeData(color: tokens.border, thickness: 1),
      drawerTheme: DrawerThemeData(backgroundColor: tokens.sidebar),
      listTileTheme: ListTileThemeData(
        textColor: tokens.foreground,
        iconColor: tokens.secondaryForeground,
      ),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return tokens.primary;
          return tokens.mutedForeground;
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return tokens.primary.withValues(alpha: 0.3);
          }
          return switchTrackOff;
        }),
      ),
      checkboxTheme: CheckboxThemeData(
        fillColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return tokens.primary;
          return Colors.transparent;
        }),
        side: BorderSide(color: tokens.border, width: 1.5),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(tokens.radiusSm),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: tokens.card,
        contentTextStyle: TextStyle(color: tokens.foreground),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(tokens.radiusMd),
        ),
      ),
      popupMenuTheme: PopupMenuThemeData(
        color: tokens.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(tokens.radiusMd),
          side: BorderSide(color: tokens.border),
        ),
      ),
      dialogTheme: DialogThemeData(
        backgroundColor: tokens.card,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(tokens.radiusLg),
          side: BorderSide(color: tokens.border),
        ),
      ),
      iconTheme: IconThemeData(color: tokens.secondaryForeground, size: 20),
    );
  }
}
