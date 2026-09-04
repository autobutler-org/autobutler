import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

void main() {
  group('QuarkTheme.from', () {
    for (final (name, tokens, brightness) in [
      ('dark', QuarkTokens.dark, Brightness.dark),
      ('light', QuarkTokens.light, Brightness.light),
    ]) {
      test('$name: the color scheme matches the tokens', () {
        final theme = QuarkTheme.from(tokens, brightness);
        final scheme = theme.colorScheme;

        expect(theme.brightness, brightness);
        expect(scheme.brightness, brightness);
        expect(scheme.primary, tokens.primary);
        expect(scheme.onPrimary, tokens.primaryForeground);
        expect(scheme.surface, tokens.card);
        expect(scheme.onSurface, tokens.foreground);
        expect(scheme.secondary, tokens.sidebar);
        expect(scheme.onSecondary, tokens.secondaryForeground);
        expect(scheme.error, tokens.error);
        expect(scheme.onError, tokens.errorForeground);
        expect(scheme.outline, tokens.border);
        expect(scheme.outlineVariant, tokens.border);
        expect(theme.scaffoldBackgroundColor, tokens.background);
      });

      test('$name: the tokens ride along as a theme extension', () {
        final theme = QuarkTheme.from(tokens, brightness);
        expect(theme.extension<QuarkTokens>(), tokens);
      });
    }

    test('edited tokens reach the theme', () {
      final edited = QuarkTokens.dark.copyWith(
        primary: const Color(0xFFAA0000),
        background: const Color(0xFF010203),
        radiusLg: 21,
      );
      final theme = QuarkTheme.from(edited, Brightness.dark);

      expect(theme.colorScheme.primary, const Color(0xFFAA0000));
      expect(theme.scaffoldBackgroundColor, const Color(0xFF010203));
      expect(
        theme.cardTheme.shape,
        RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(21),
          side: BorderSide(color: edited.border),
        ),
      );
    });

    test('dark() and light() are from() with the shipped tokens', () {
      expect(
        QuarkTheme.dark().colorScheme,
        QuarkTheme.from(QuarkTokens.dark, Brightness.dark).colorScheme,
      );
      expect(
        QuarkTheme.light().colorScheme,
        QuarkTheme.from(QuarkTokens.light, Brightness.light).colorScheme,
      );
      expect(QuarkTheme.dark().extension<QuarkTokens>(), QuarkTokens.dark);
      expect(QuarkTheme.light().extension<QuarkTokens>(), QuarkTokens.light);
    });
  });
}
