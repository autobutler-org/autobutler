import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/pages/terms_page.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/auth_service.dart';

/// #1624: accepting terms navigated to /files and left it to [authRedirect] to
/// move the user on to login. That redirect swallowed every failure of the
/// status call and returned "stay put", so a transient connection failure at
/// exactly that moment stranded a signed-out user on a file browser that could
/// only render errors.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const secureStorage = MethodChannel(
    'plugins.it_nomads.com/flutter_secure_storage',
  );

  setUpAll(() {
    // The session token lives in secure storage on native platforms, and
    // there's no plugin behind it in a unit test.
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, (_) async => null);
  });

  tearDownAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, null);
  });

  final settings = AppSettings.instance;

  Future<void> clearHosts() async {
    while (settings.hosts.isNotEmpty) {
      await settings.removeHost(settings.hosts.length - 1);
    }
  }

  setUp(() async {
    await clearHosts();
    authStatusProbe = AuthService.checkStatus;
  });

  tearDown(() async {
    await clearHosts();
    authStatusProbe = AuthService.checkStatus;
  });

  /// Acceptance is recorded per host address and the singleton keeps that set
  /// for the whole run, so every test needs a Quark it has never seen.
  var hostSerial = 0;
  Future<void> addUnacceptedHost() async {
    hostSerial++;
    await settings.addHost(
      HostEntry(
        name: 'Quark $hostSerial',
        hostAddress: 'https://quark-$hostSerial.local',
      ),
    );
  }

  /// The real gate over stub pages, so the test drives the actual rules
  /// without mounting the whole app.
  Future<GoRouter> pumpGatedRouter(
    WidgetTester tester, {
    String initialLocation = AppRoutes.files,
  }) async {
    final router = GoRouter(
      initialLocation: initialLocation,
      redirect: authRedirect,
      refreshListenable: routerRefreshListenable,
      routes: [
        GoRoute(
          path: AppRoutes.files,
          builder: (_, _) => const Scaffold(body: Text('files')),
        ),
        GoRoute(
          path: AppRoutes.login,
          builder: (_, _) => const Scaffold(body: Text('login')),
        ),
        GoRoute(
          path: AppRoutes.setup,
          builder: (_, _) => const Scaffold(body: Text('setup')),
        ),
        GoRoute(
          path: AppRoutes.settings,
          builder: (_, _) => const Scaffold(body: Text('settings')),
        ),
        GoRoute(path: AppRoutes.terms, builder: (_, _) => const TermsPage()),
      ],
    );
    await tester.pumpWidget(MaterialApp.router(routerConfig: router));
    await tester.pumpAndSettle();
    return router;
  }

  group('accepting terms lands on', () {
    setUp(addUnacceptedHost);

    testWidgets('login, when the Quark is set up and reachable', (
      tester,
    ) async {
      authStatusProbe = () async => const AuthStatus(setupComplete: true);

      await pumpGatedRouter(tester);
      expect(find.text('I Agree'), findsOneWidget);

      await tester.tap(find.text('I Agree'));
      await tester.pumpAndSettle();

      expect(find.text('login'), findsOneWidget);
      expect(find.text('files'), findsNothing);
    });

    // The reported failure. On main this landed on 'files': the terms page
    // went there and the redirect's catch returned null.
    testWidgets('login, even when the status call fails at that moment', (
      tester,
    ) async {
      authStatusProbe = () async => throw Exception('connection refused');

      await pumpGatedRouter(tester);
      await tester.tap(find.text('I Agree'));
      await tester.pumpAndSettle();

      expect(find.text('login'), findsOneWidget);
      expect(find.text('files'), findsNothing);
    });

    testWidgets('setup, for a Quark nobody has claimed yet', (tester) async {
      authStatusProbe = () async => const AuthStatus(setupComplete: false);

      await pumpGatedRouter(tester);
      await tester.tap(find.text('I Agree'));
      await tester.pumpAndSettle();

      expect(find.text('setup'), findsOneWidget);
    });

    testWidgets('files, when a session already exists', (tester) async {
      await settings.setSessionToken('a-token');
      addTearDown(() => settings.setSessionToken(null));
      authStatusProbe = () async =>
          throw StateError('must not probe with a live session');

      await pumpGatedRouter(tester);
      await tester.tap(find.text('I Agree'));
      await tester.pumpAndSettle();

      expect(find.text('files'), findsOneWidget);
    });
  });

  group('an unreachable Quark does not strand a signed-out user', () {
    setUp(() async {
      await addUnacceptedHost();
      await settings.acceptTerms();
      authStatusProbe = () async => throw Exception('connection refused');
    });

    testWidgets('/files redirects to login rather than rendering errors', (
      tester,
    ) async {
      await pumpGatedRouter(tester);

      expect(find.text('login'), findsOneWidget);
      expect(find.text('files'), findsNothing);
    });

    // Settings is not the escape hatch any more: the login page owns host
    // management, so a bad address is corrected there (#1639). Settings stays
    // behind the auth gate like every other route.
    testWidgets('settings still redirects to login, not around the gate', (
      tester,
    ) async {
      final router = await pumpGatedRouter(tester);
      expect(find.text('login'), findsOneWidget);

      router.push(AppRoutes.settings);
      await tester.pumpAndSettle();

      expect(find.text('settings'), findsNothing);
      expect(find.text('login'), findsOneWidget);
    });
  });

  // The terms gate runs ahead of everything, so an unaccepted Quark sees terms
  // rather than the login page it would otherwise be sent to.
  testWidgets('settings still requires terms for the active Quark', (
    tester,
  ) async {
    await addUnacceptedHost();
    authStatusProbe = () async => const AuthStatus(setupComplete: true);

    await pumpGatedRouter(tester, initialLocation: AppRoutes.settings);

    expect(find.text('I Agree'), findsOneWidget);
    expect(find.text('settings'), findsNothing);
  });
}
