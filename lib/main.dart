import 'package:autobutler/router.dart';
import 'package:autobutler/theme/autobutler_theme.dart';
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/material.dart';

Future<void> main() async {
  usePathUrlStrategy();
  WidgetsFlutterBinding.ensureInitialized();
  await AppSettings.instance.load();
  runApp(const AutobutlerApp());
}

class AutobutlerApp extends StatelessWidget {
  const AutobutlerApp({super.key});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<ThemeMode>(
      valueListenable: AppSettings.instance.themeMode,
      builder: (context, mode, _) {
        return MaterialApp.router(
          debugShowCheckedModeBanner: false,
          title: 'Autobutler',
          theme: AutobutlerTheme.light(),
          darkTheme: AutobutlerTheme.dark(),
          themeMode: mode,
          routerConfig: router,
        );
      },
    );
  }
}
