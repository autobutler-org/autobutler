import 'package:autobutler/pages/file_browser_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await AppSettings.instance.load();
  _maybeAutoUpdate();
  runApp(const AutobutlerApp());
}

void _maybeAutoUpdate() {
  if (!AppSettings.instance.autoUpdate) return;
  if (AppSettings.instance.activeHost == null) return;
  CirrusService.updateToLatest().catchError((e) {
    debugPrint('Auto-update failed: $e');
  });
}

class AutobutlerApp extends StatelessWidget {
  const AutobutlerApp({super.key});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<ThemeMode>(
      valueListenable: AppSettings.instance.themeMode,
      builder: (context, mode, _) {
        return MaterialApp(
          debugShowCheckedModeBanner: false,
          title: 'Autobutler',
          theme: ThemeData(
            colorScheme: ColorScheme.fromSeed(
              seedColor: Colors.blue,
              brightness: Brightness.light,
            ),
            useMaterial3: true,
          ),
          darkTheme: ThemeData(
            colorScheme: ColorScheme.fromSeed(
              seedColor: Colors.blue,
              brightness: Brightness.dark,
            ),
            scaffoldBackgroundColor: const Color(0xFF070D19),
            useMaterial3: true,
          ),
          themeMode: mode,
          home: const FileBrowserPage(),
        );
      },
    );
  }
}
