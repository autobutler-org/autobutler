// GENERATED FILE - do not edit by hand.
//
// Class docs from packages/quark_widgets/lib/src, for the
// gallery's documentation pane. Regenerate with:
//
//     make -C packages/quark_widgets generate/docs
//
// The formatter is off so that the generator, and the test
// that checks this file is current, agree byte for byte.
// dart format off

/// Documentation for each exported widget, keyed by class name.
const Map<String, String> widgetDocs = {
  'QuarkTokens': 'The design tokens every Quark widget draws from: colors, corner radii, and\na spacing scale.\n\nTokens are values, not constants, so they can be edited at runtime — the\nwidget gallery\'s theme panel rebuilds the whole app from an edited\n[QuarkTokens]. A widget that hardcodes a color instead of reading a token\nstops following the panel, which is how you spot it.\n\nReach them through the theme:\n\n```dart\nfinal tokens = QuarkTokens.of(context);\nContainer(\n  color: tokens.card,\n  padding: EdgeInsets.all(tokens.spacingMd),\n);\n```\n\n[QuarkTokens.dark] and [QuarkTokens.light] are the two sets the app ships.',
};
