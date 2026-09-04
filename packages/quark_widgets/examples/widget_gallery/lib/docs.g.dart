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
  'QuarkAppBar': 'The app bar every main page wears: a [QuarkBrandButton] on the left that\nopens the drawer, no title, and the page\'s own actions on the right.\n\nThe theme toggle is not built in. It reads the app\'s settings, so the page\nappends its own wired copy to [actions] and the package stays free of app\nstate.\n\nKey prefixes: `brand_button`, from the [QuarkBrandButton] it renders.\n\n```dart\nScaffold(\n  appBar: QuarkAppBar(\n    label: \'Photos\',\n    icon: QuarkIcons.photo_library_outlined,\n    actions: [RefreshIconButton(...), const AppThemeToggle()],\n  ),\n);\n```',
  'QuarkBrandButton': 'The branded badge-and-label control that leads every main page\'s top bar.\n\nIt is a button, not a title: tapping it is how the app opens the navigation\ndrawer. When used as [AppBar.leading], set [AppBar.leadingWidth] to at least\n[QuarkBrandButton.preferredWidth] or the row overflows.\n\nKey prefixes: `brand_button` on the control itself.\n\n```dart\nQuarkBrandButton(\n  label: \'Files\',\n  onTap: () => Scaffold.of(context).openDrawer(),\n);\n```',
  'QuarkDrawer': 'The app\'s navigation drawer: one row per [QuarkDrawerSection], with the\ncurrent one marked.\n\nThe drawer navigates nothing itself. Each row calls back and the page\nroutes, so the package stays free of the router.\n\nKey prefixes: `drawer_<section>` on each row, for example `drawer_photos`.\n\n```dart\nQuarkDrawer(\n  activeSection: QuarkDrawerSection.photos,\n  onTapFiles: () => context.go(AppRoutes.files),\n);\n```',
  'QuarkTokens': 'The design tokens every Quark widget draws from: colors, corner radii, and\na spacing scale.\n\nTokens are values, not constants, so they can be edited at runtime — the\nwidget gallery\'s theme panel rebuilds the whole app from an edited\n[QuarkTokens]. A widget that hardcodes a color instead of reading a token\nstops following the panel, which is how you spot it.\n\nReach them through the theme:\n\n```dart\nfinal tokens = QuarkTokens.of(context);\nContainer(\n  color: tokens.card,\n  padding: EdgeInsets.all(tokens.spacingMd),\n);\n```\n\n[QuarkTokens.dark] and [QuarkTokens.light] are the two sets the app ships.',
  'RefreshIconButton': 'A refresh [IconButton] that swaps its glyph for a spinner while a refresh\nis in flight, and refuses taps until it finishes.\n\nWhether a refresh is running is an input, not something the button tracks:\nthe page owns the load.\n\nKey prefixes: `refresh_button` on the control itself.\n\n```dart\nRefreshIconButton(\n  isRefreshing: controller.isLoading,\n  onPressed: controller.refresh,\n);\n```',
  'ThemeToggleButton': 'An [IconButton] that switches the app between light and dark.\n\nThe current [mode] comes in and the chosen one goes out; the package never\nreads or writes the app\'s settings. From [ThemeMode.system] the button\ncommits to light, because the first tap is a user saying they want the\nother one, not the one they are already looking at.\n\nKey prefixes: `theme_toggle` on the control itself.\n\n```dart\nThemeToggleButton(\n  mode: settings.themeMode.value,\n  onChanged: settings.setThemeMode,\n);\n```',
};
