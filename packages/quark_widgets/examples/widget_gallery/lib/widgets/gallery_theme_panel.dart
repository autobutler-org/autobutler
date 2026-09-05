import 'package:flutter/material.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../token_fields.dart';
import 'hex_field.dart';
import 'number_slider.dart';

/// The right panel: the live token editor behind the gallery's theme.
class GalleryThemePanel extends StatelessWidget {
  /// Creates the editor for [tokens] at [brightness].
  const GalleryThemePanel({
    required this.tokens,
    required this.brightness,
    required this.onToggleBrightness,
    required this.onTokensChanged,
    super.key,
  });

  /// The token set the panel edits.
  final QuarkTokens tokens;

  /// The brightness the gallery is showing, driving the dark switch.
  final Brightness brightness;

  /// Called when the dark switch is flipped.
  final VoidCallback onToggleBrightness;

  /// Called with the new token set after any field is edited.
  final ValueChanged<QuarkTokens> onTokensChanged;

  @override
  Widget build(BuildContext context) {
    final themeTokens = QuarkTokens.of(context);

    return ListView(
      padding: EdgeInsets.all(themeTokens.spacingMd),
      children: [
        Text('Theme', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: themeTokens.spacingSm),
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          dense: true,
          title: const Text('Dark'),
          value: brightness == Brightness.dark,
          onChanged: (_) => onToggleBrightness(),
        ),
        SizedBox(height: themeTokens.spacingSm),
        for (final field in colorFields)
          HexField(
            // Rebuild the controllers when the token set is swapped wholesale.
            key: ValueKey('${field.name}-$brightness'),
            name: field.name,
            value: field.read(tokens),
            onSubmitted: (color) => onTokensChanged(field.write(tokens, color)),
          ),
        SizedBox(height: themeTokens.spacingMd),
        for (final field in numberFields)
          NumberSlider(
            name: field.name,
            value: field.read(tokens),
            max: field.max,
            onChanged: (value) => onTokensChanged(field.write(tokens, value)),
          ),
      ],
    );
  }
}
