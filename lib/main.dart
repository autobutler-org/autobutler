import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/local_trust_overrides_stub.dart'
    if (dart.library.io) 'package:quark/services/local_trust_overrides_io.dart';
import 'package:quark/theme/quark_theme.dart';

Future<void> main() async {
  usePathUrlStrategy();
  WidgetsFlutterBinding.ensureInitialized();
  await AppSettings.instance.load();
  // Quarks on the local network serve self-signed certificates. Install the
  // trust policy after settings load so it can consult the configured host.
  installLocalTrustHttpOverrides();
  runApp(const QuarkApp());
}

class QuarkApp extends StatelessWidget {
  const QuarkApp({super.key});

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
          routerConfig: router,
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
