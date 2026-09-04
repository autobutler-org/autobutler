import 'package:flutter/material.dart';
import 'package:quark_widgets/quark_widgets.dart';

import 'token_fields.dart';

/// One widget in the gallery: a live example built from fake data.
///
/// The builder is handed a `log` callback. Wire every callback the widget
/// exposes to it, so the gallery's event panel shows what the widget emits and
/// a callback that never fires is visible.
class GalleryEntry {
  /// Creates an entry for [name], filed under [group], rendering [build].
  const GalleryEntry({
    required this.name,
    required this.group,
    required this.build,
  });

  /// The class name of the widget, matching its entry in `docs.g.dart`.
  final String name;

  /// The heading this entry is listed under, usually its `lib/src/` directory.
  final String group;

  /// Builds the example. Pass `log` to every callback the widget takes.
  final Widget Function(BuildContext context, void Function(String event) log)
  build;
}

/// Every widget the gallery can show.
///
/// A package test fails when a widget exported from `quark_widgets.dart` has no
/// entry here, so this list stays complete as widgets land.
final List<GalleryEntry> registry = [
  GalleryEntry(
    name: 'Theme tokens',
    group: 'Theme',
    build: (context, log) => const _TokenSwatches(),
  ),
];

/// Every token in the current theme, drawn from the theme itself.
///
/// This is the gallery's own canary: edit a color in the theme panel and the
/// matching swatch has to move with it.
class _TokenSwatches extends StatelessWidget {
  const _TokenSwatches();

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Colors', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: tokens.spacingSm),
        Wrap(
          spacing: tokens.spacingSm,
          runSpacing: tokens.spacingSm,
          children: [
            for (final field in colorFields)
              _Swatch(name: field.name, color: field.read(tokens)),
          ],
        ),
        SizedBox(height: tokens.spacingLg),
        Text('Radii', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: tokens.spacingSm),
        Wrap(
          spacing: tokens.spacingSm,
          runSpacing: tokens.spacingSm,
          children: [
            _RadiusSample(name: 'radiusSm', radius: tokens.radiusSm),
            _RadiusSample(name: 'radiusMd', radius: tokens.radiusMd),
            _RadiusSample(name: 'radiusLg', radius: tokens.radiusLg),
          ],
        ),
        SizedBox(height: tokens.spacingLg),
        Text('Spacing', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: tokens.spacingSm),
        for (final step in [
          ('spacingXs', tokens.spacingXs),
          ('spacingSm', tokens.spacingSm),
          ('spacingMd', tokens.spacingMd),
          ('spacingLg', tokens.spacingLg),
          ('spacingXl', tokens.spacingXl),
        ])
          Padding(
            padding: EdgeInsets.only(bottom: tokens.spacingXs),
            child: Row(
              children: [
                SizedBox(
                  width: 96,
                  child: Text(
                    step.$1,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                    ),
                  ),
                ),
                Container(width: step.$2, height: 12, color: tokens.primary),
                SizedBox(width: tokens.spacingSm),
                Text(
                  step.$2.toStringAsFixed(0),
                  style: TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 12,
                    color: tokens.mutedForeground,
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

class _Swatch extends StatelessWidget {
  const _Swatch({required this.name, required this.color});

  final String name;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return SizedBox(
      width: 132,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            height: 44,
            decoration: BoxDecoration(
              color: color,
              border: Border.all(color: tokens.border),
              borderRadius: BorderRadius.circular(tokens.radiusMd),
            ),
          ),
          SizedBox(height: tokens.spacingXs),
          Text(
            name,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            overflow: TextOverflow.ellipsis,
          ),
          Text(
            toHex(color),
            style: TextStyle(
              fontFamily: 'monospace',
              fontSize: 11,
              color: tokens.mutedForeground,
            ),
          ),
        ],
      ),
    );
  }
}

class _RadiusSample extends StatelessWidget {
  const _RadiusSample({required this.name, required this.radius});

  final String name;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return SizedBox(
      width: 132,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            height: 44,
            decoration: BoxDecoration(
              color: tokens.card,
              border: Border.all(color: tokens.primary, width: 2),
              borderRadius: BorderRadius.circular(radius),
            ),
          ),
          SizedBox(height: tokens.spacingXs),
          Text(
            '$name ${radius.toStringAsFixed(0)}',
            style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
          ),
        ],
      ),
    );
  }
}
