import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/models/plugin_manifest.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/local_trust_overrides_stub.dart'
    if (dart.library.io) 'package:quark/services/local_trust_overrides_io.dart';
import 'package:quark/services/plugin_service.dart';
import 'package:quark/services/plugin_state.dart';
import 'package:quark/theme/quark_theme.dart';

Future<void> main() async {
  usePathUrlStrategy();
  WidgetsFlutterBinding.ensureInitialized();
  await AppSettings.instance.load();
  // Butlers on the local network serve self-signed certificates. Install the
  // trust policy after settings load so it can consult the configured host.
  installLocalTrustHttpOverrides();
  runApp(const QuarkApp());
}

class QuarkApp extends StatefulWidget {
  const QuarkApp({super.key});

  @override
  State<QuarkApp> createState() => _QuarkAppState();
}

class _QuarkAppState extends State<QuarkApp> {
  List<PluginManifest> _plugins = const [];
  late GoRouter _router;

  @override
  void initState() {
    super.initState();
    _router = buildRouter(plugins: _plugins);
    _loadPlugins();
  }

  Future<void> _loadPlugins() async {
    if (AppSettings.instance.activeHost == null) return;
    try {
      final plugins = await PluginService.listPlugins();
      if (!mounted) return;
      PluginState.instance.setPlugins(plugins);
      setState(() {
        _plugins = plugins;
        _router = buildRouter(plugins: _plugins);
      });
    } catch (_) {
      // Plugins are non-critical; fail silently on load errors.
    }
  }

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<ThemeMode>(
      valueListenable: AppSettings.instance.themeMode,
      builder: (context, mode, _) {
        return MaterialApp.router(
          debugShowCheckedModeBanner: false,
          title: 'Quark',
          theme: QuarkTheme.light(),
          darkTheme: QuarkTheme.dark(),
          themeMode: mode,
          routerConfig: _router,
          localizationsDelegates: const [
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
            FlutterQuillLocalizations.delegate,
          ],
          supportedLocales: const [Locale('en')],
        );
      },
    );
  }
}
