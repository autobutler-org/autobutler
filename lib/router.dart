import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/pages/docs_page.dart';
import 'package:quark/pages/document_editor_page.dart';
import 'package:quark/pages/file_browser_page.dart';
import 'package:quark/pages/health_page.dart';
import 'package:quark/pages/login_page.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/pages/plaintext_editor_page.dart';
import 'package:quark/pages/recover_page.dart';
import 'package:quark/pages/settings_page.dart';
import 'package:quark/pages/setup_page.dart';
import 'package:quark/pages/sheets_page.dart';
import 'package:quark/pages/spreadsheet_editor_page.dart';
import 'package:quark/pages/storage_devices_page.dart';
import 'package:quark/pages/terms_page.dart';
import 'package:quark/pages/vault_page.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/auth_service.dart';

// Route paths — use these constants everywhere instead of string literals.
class AppRoutes {
  static const files = '/files';

  /// Legacy alias. The file browser lived at /cirrus for the product's whole
  /// life, so external links and bookmarks exist. This redirects to [files].
  ///
  // TODO(pre-v1.0.0, #1601): delete this constant and the two /cirrus GoRoutes
  // that redirect to /files.
  static const legacyCirrus = '/cirrus';

  /// Deep-link pattern for opening a specific file in the correct viewer.
  /// e.g. /view/photos/2024/beach.jpg resolves the type and opens the viewer.
  static const viewFile = '/view';

  /// Deep-link pattern for a specific path inside the file browser.
  /// e.g. /files/photos/2024 navigates directly to photos/2024.
  static const filesDeep = '/files/:path(.*)';

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
  static const terms = '/terms';
  static const plaintextEditor = '/edit';

  /// Build a URL for a specific plaintext file.
  /// e.g. plaintextEditorPath('notes/readme.txt') → '/edit/notes/readme.txt'
  /// Device serial is passed as a query param when non-empty.
  static String plaintextEditorPath(String path, {String? serial}) {
    final clean = path.replaceAll(RegExp(r'^/+'), '');
    final base = '$plaintextEditor/$clean';
    return (serial != null && serial.isNotEmpty)
        ? '$base?serial=${Uri.encodeQueryComponent(serial)}'
        : base;
  }

  /// Build a deep-link URL that opens [path] in the correct viewer.
  /// e.g. viewFile('photos/beach.jpg') → '/view/photos/beach.jpg'
  /// Device serial is passed as a query param when non-empty.
  static String viewFilePath(String path, {String? serial}) {
    final clean = path.replaceAll(RegExp(r'^/+'), '');
    final base = '$viewFile/$clean';
    return (serial != null && serial.isNotEmpty)
        ? '$base?serial=${Uri.encodeQueryComponent(serial)}'
        : base;
  }

  /// Build a deep-link URL for a given files path.
  /// e.g. filesPath('photos/2024') → '/files/photos/2024'
  static String filesPath(String path) {
    final clean = path.replaceAll(RegExp(r'^/+'), '');
    return clean.isEmpty ? files : '$files/$clean';
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
  initialLocation: AppRoutes.files,
  redirect: _authRedirect,
  // Refresh the router whenever the session token changes so a 401-triggered
  // token clear immediately redirects to the login page.
  refreshListenable: Listenable.merge([
    AppSettings.instance.sessionTokenNotifier,
    AppSettings.instance.hasAcceptedTerms,
  ]),
  routes: [
    GoRoute(
      path: AppRoutes.files,
      builder: (context, state) => const FileBrowserPage(),
      routes: [
        GoRoute(
          // Matches /files/<anything>, including slashes.
          // Always renders FileBrowserPage so go_router owns the page and
          // URL changes (back, go-up, breadcrumb) correctly trigger
          // didUpdateWidget. FileBrowserPage._openPendingFile stats the path
          // and launches the right viewer for files.
          path: ':path(.*)',
          builder: (context, state) {
            final raw = state.pathParameters['path'] ?? '';
            final filePath = Uri.decodeComponent(raw);
            return FileBrowserPage(initialPath: filePath);
          },
        ),
      ],
    ),
    GoRoute(
      // /view/:path redirects to /files/:path for backward compatibility.
      path: '${AppRoutes.viewFile}/:path(.*)',
      redirect: (context, state) {
        final raw = state.pathParameters['path'] ?? '';
        final serial = state.uri.queryParameters['serial'];
        final base = AppRoutes.filesPath(Uri.decodeComponent(raw));
        return serial != null && serial.isNotEmpty
            ? '$base?serial=${Uri.encodeQueryComponent(serial)}'
            : base;
      },
    ),
    GoRoute(
      // TODO(pre-v1.0.0, #1601): delete this route with the /cirrus alias.
      // /cirrus/:path redirects to /files/:path. The browser lived at /cirrus
      // before the rename, so old links and bookmarks must keep resolving.
      path: '${AppRoutes.legacyCirrus}/:path(.*)',
      redirect: (context, state) {
        final raw = state.pathParameters['path'] ?? '';
        final serial = state.uri.queryParameters['serial'];
        final base = AppRoutes.filesPath(Uri.decodeComponent(raw));
        return serial != null && serial.isNotEmpty
            ? '$base?serial=${Uri.encodeQueryComponent(serial)}'
            : base;
      },
    ),
    GoRoute(
      // TODO(pre-v1.0.0, #1601): delete this route with the /cirrus alias.
      // Bare /cirrus → /files.
      path: AppRoutes.legacyCirrus,
      redirect: (context, state) => AppRoutes.files,
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
          SetupPage(onSetupComplete: () => context.go(AppRoutes.files)),
    ),
    GoRoute(
      path: AppRoutes.login,
      builder: (context, state) =>
          LoginPage(onLoginSuccess: () => context.go(AppRoutes.files)),
    ),
    GoRoute(
      path: AppRoutes.recover,
      builder: (context, state) => const RecoverPage(),
    ),
    GoRoute(path: AppRoutes.terms, builder: (context, _) => const TermsPage()),
    GoRoute(
      // Matches /edit/<anything including slashes> — opens the plaintext editor.
      path: '${AppRoutes.plaintextEditor}/:path(.*)',
      builder: (context, state) {
        final raw = state.pathParameters['path'] ?? '';
        final filePath = Uri.decodeComponent(raw);
        final serial = state.uri.queryParameters['serial'] ?? '';
        return PlaintextEditorPage(filePath: filePath, deviceSerial: serial);
      },
    ),
  ],
  errorBuilder: (context, state) =>
      Scaffold(body: Center(child: Text('Page not found: ${state.uri}'))),
);

/// Top-level redirect — handles auth gating.
Future<String?> _authRedirect(BuildContext context, GoRouterState state) async {
  final publicRoutes = {
    AppRoutes.setup,
    AppRoutes.login,
    AppRoutes.recover,
    AppRoutes.terms,
  };

  // Public routes are always accessible.
  if (publicRoutes.contains(state.matchedLocation)) return null;

  // No host configured — let the main app handle the "add host" prompt.
  if (AppSettings.instance.activeHost == null) return null;

  // Terms must be accepted before accessing the app.
  if (!AppSettings.instance.hasAcceptedTerms.value) return AppRoutes.terms;

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
