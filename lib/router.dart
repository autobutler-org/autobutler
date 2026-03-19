import 'package:autobutler/pages/file_browser_page.dart';
import 'package:autobutler/pages/health_page.dart';
import 'package:autobutler/pages/login_page.dart';
import 'package:autobutler/pages/photos_page.dart';
import 'package:autobutler/pages/recover_page.dart';
import 'package:autobutler/pages/settings_page.dart';
import 'package:autobutler/pages/setup_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/auth_service.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

// Route paths — use these constants everywhere instead of string literals.
class AppRoutes {
  static const cirrus = '/cirrus';
  static const photos = '/photos';
  static const health = '/health';
  static const settings = '/settings';
  static const setup = '/setup';
  static const login = '/login';
  static const recover = '/recover';
  // Note: image/video viewers are push-only overlays (take runtime data),
  // not deep-linkable URL routes.
}

final router = GoRouter(
  initialLocation: AppRoutes.cirrus,
  redirect: _authRedirect,
  routes: [
    GoRoute(
      path: AppRoutes.cirrus,
      builder: (context, state) => const FileBrowserPage(),
    ),
    GoRoute(
      path: AppRoutes.photos,
      builder: (context, state) => const PhotosPage(),
    ),
    GoRoute(
      path: AppRoutes.health,
      builder: (context, state) => const HealthPage(),
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

/// Top-level redirect — handles auth gating and butler instance mismatch detection.
Future<String?> _authRedirect(BuildContext context, GoRouterState state) async {
  final publicRoutes = {AppRoutes.setup, AppRoutes.login, AppRoutes.recover};

  // Public routes are always accessible.
  if (publicRoutes.contains(state.matchedLocation)) return null;

  // No host configured — let the main app handle the "add host" prompt.
  if (AppSettings.instance.activeHost == null) return null;

  // Check server-side status — always, even when authenticated, to catch
  // instance mismatch (e.g. connected to a different butler than expected).
  try {
    final status = await AuthService.checkStatus();

    // Warn the user if we detected a different butler at this host address.
    // This can happen after a device swap or on a network with another butler
    // at the same address (e.g. a neighbour's butler — issue #414).
    if (status.instanceMismatch && context.mounted) {
      final confirmed = await showDialog<bool>(
        context: context,
        barrierDismissible: false,
        builder: (ctx) => AlertDialog(
          title: const Text('Different butler detected'),
          content: Text(
            'The butler at ${AppSettings.instance.activeHost ?? "this host"} '
            'has a different identity than the one you previously connected to. '
            'This can happen if you\'re on a different network, or if another '
            'butler is reachable at the same address.\n\n'
            'Your session has been cleared. Please log in to confirm '
            'you\'re connecting to the right butler.',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              child: const Text('OK, connect'),
            ),
          ],
        ),
      );
      if (confirmed == true) {
        final host = AppSettings.instance.activeHost;
        if (host != null) {
          await AppSettings.instance.clearInstanceId(host);
        }
        await AppSettings.instance.setSessionToken(null);
      }
    }

    if (!status.setupComplete) return AppRoutes.setup;

    // If no session token, go to login.
    if (AppSettings.instance.sessionToken == null) return AppRoutes.login;

    return null;
  } catch (_) {
    // Can't reach butler — allow through; individual pages will surface errors.
    return null;
  }
}
