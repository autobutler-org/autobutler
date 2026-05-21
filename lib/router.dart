import 'package:autobutler/pages/document_editor_page.dart';
import 'package:autobutler/pages/docs_page.dart';
import 'package:autobutler/pages/file_browser_page.dart';
import 'package:autobutler/pages/health_page.dart';
import 'package:autobutler/pages/storage_devices_page.dart';
import 'package:autobutler/pages/login_page.dart';
import 'package:autobutler/pages/photos_page.dart';
import 'package:autobutler/pages/recover_page.dart';
import 'package:autobutler/pages/settings_page.dart';
import 'package:autobutler/pages/setup_page.dart';
import 'package:autobutler/pages/sheets_page.dart';
import 'package:autobutler/pages/spreadsheet_editor_page.dart';
import 'package:autobutler/pages/vault_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/auth_service.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

// Route paths — use these constants everywhere instead of string literals.
class AppRoutes {
  static const cirrus = '/cirrus';

  /// Deep-link pattern for a specific path inside the file browser.
  /// e.g. /cirrus/photos/2024 navigates directly to photos/2024.
  static const cirrusDeep = '/cirrus/:path(.*)';

  static const photos = '/photos';
  static const docs = '/docs';
  static const sheets = '/sheets';
  static const devices = '/devices';
  static const health = '/health';
  static const vault = '/vault';
  static const settings = '/settings';
  static const setup = '/setup';
  static const login = '/login';
  static const recover = '/recover';

  /// Build a deep-link URL for a given cirrus path.
  /// e.g. cirrusPath('photos/2024') → '/cirrus/photos/2024'
  static String cirrusPath(String path) {
    final clean = path.replaceAll(RegExp(r'^/+'), '');
    return clean.isEmpty ? cirrus : '/cirrus/$clean';
  }

  /// Build a URL for a specific document file.
  /// e.g. docFile('reports/q1.abdoc') → '/docs/reports/q1.abdoc'
  /// Device serial is passed as a query param when non-empty.
  static String docFile(String path, {String? serial}) {
    final clean = path.replaceAll(RegExp(r'^/+'), '');
    final base = '$docs/$clean';
    return (serial != null && serial.isNotEmpty)
        ? '$base?serial=${Uri.encodeQueryComponent(serial)}'
        : base;
  }

  /// Build a URL for a specific spreadsheet file.
  /// e.g. sheetFile('data/budget.absheet') → '/sheets/data/budget.absheet'
  static String sheetFile(String path, {String? serial}) {
    final clean = path.replaceAll(RegExp(r'^/+'), '');
    final base = '$sheets/$clean';
    return (serial != null && serial.isNotEmpty)
        ? '$base?serial=${Uri.encodeQueryComponent(serial)}'
        : base;
  }
}

final router = GoRouter(
  initialLocation: AppRoutes.cirrus,
  redirect: _authRedirect,
  routes: [
    GoRoute(
      path: AppRoutes.cirrus,
      builder: (context, state) => const FileBrowserPage(),
      routes: [
        GoRoute(
          // Matches /cirrus/<anything>, including slashes.
          path: ':path(.*)',
          builder: (context, state) {
            final raw = state.pathParameters['path'] ?? '';
            return FileBrowserPage(initialPath: Uri.decodeComponent(raw));
          },
        ),
      ],
    ),
    GoRoute(
      path: AppRoutes.photos,
      builder: (context, state) => const PhotosPage(),
    ),
    GoRoute(
      path: AppRoutes.docs,
      builder: (context, state) => const DocsPage(),
      routes: [
        GoRoute(
          // Matches /docs/<anything including slashes> — opens the doc editor.
          path: ':path(.*)',
          builder: (context, state) {
            final raw = state.pathParameters['path'] ?? '';
            final filePath = Uri.decodeComponent(raw);
            final serial = state.uri.queryParameters['serial'] ?? '';
            return DocumentEditorPage(filePath: filePath, deviceSerial: serial);
          },
        ),
      ],
    ),
    GoRoute(
      path: AppRoutes.sheets,
      builder: (context, state) => const SheetsPage(),
      routes: [
        GoRoute(
          // Matches /sheets/<anything including slashes> — opens the sheet editor.
          path: ':path(.*)',
          builder: (context, state) {
            final raw = state.pathParameters['path'] ?? '';
            final filePath = Uri.decodeComponent(raw);
            final serial = state.uri.queryParameters['serial'] ?? '';
            return SpreadsheetEditorPage(
              filePath: filePath,
              deviceSerial: serial,
            );
          },
        ),
      ],
    ),
    GoRoute(
      path: AppRoutes.devices,
      builder: (context, state) => const StorageDevicesPage(),
    ),
    GoRoute(
      path: AppRoutes.health,
      builder: (context, state) => const HealthPage(),
    ),
    GoRoute(
      path: AppRoutes.vault,
      builder: (context, state) => const VaultPage(),
    ),
    GoRoute(
      path: AppRoutes.settings,
      builder: (context, state) => const SettingsPage(),
    ),
    GoRoute(
      path: AppRoutes.setup,
      builder: (context, state) =>
          SetupPage(onSetupComplete: () => context.go(AppRoutes.cirrus)),
    ),
    GoRoute(
      path: AppRoutes.login,
      builder: (context, state) =>
          LoginPage(onLoginSuccess: () => context.go(AppRoutes.cirrus)),
    ),
    GoRoute(
      path: AppRoutes.recover,
      builder: (context, state) => const RecoverPage(),
    ),
  ],
  errorBuilder: (context, state) =>
      Scaffold(body: Center(child: Text('Page not found: ${state.uri}'))),
);

/// Top-level redirect — handles auth gating.
Future<String?> _authRedirect(BuildContext context, GoRouterState state) async {
  final publicRoutes = {AppRoutes.setup, AppRoutes.login, AppRoutes.recover};

  // Public routes are always accessible.
  if (publicRoutes.contains(state.matchedLocation)) return null;

  // No host configured — let the main app handle the "add host" prompt.
  if (AppSettings.instance.activeHost == null) return null;

  // Already authenticated.
  if (AppSettings.instance.sessionToken != null) return null;

  // Check server-side status.
  try {
    final status = await AuthService.checkStatus();
    if (!status.setupComplete) return AppRoutes.setup;
    return AppRoutes.login;
  } catch (_) {
    // Can't reach butler — allow through; individual pages will surface errors.
    return null;
  }
}
