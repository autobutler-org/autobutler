import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';

void main() {
  group('AppRoutes.encodeFilePath', () {
    test('leaves an ordinary path untouched', () {
      expect(
        AppRoutes.encodeFilePath('photos/2024/beach.jpg'),
        'photos/2024/beach.jpg',
      );
    });

    test('strips leading slashes', () {
      expect(AppRoutes.encodeFilePath('/photos/beach.jpg'), 'photos/beach.jpg');
    });

    test('encodes spaces but keeps the separators', () {
      expect(
        AppRoutes.encodeFilePath('my folder/my doc.abdoc'),
        'my%20folder/my%20doc.abdoc',
      );
    });

    test('encodes a literal percent so it survives one decode', () {
      expect(
        AppRoutes.encodeFilePath('holiday 100%.abdoc'),
        'holiday%20100%25.abdoc',
      );
    });

    test('is empty for the root', () {
      expect(AppRoutes.encodeFilePath('/'), '');
      expect(AppRoutes.encodeFilePath(''), '');
    });
  });

  group('route builders emit URLs go_router can echo back verbatim', () {
    // #1604: these were built by raw interpolation, so a name with a space
    // produced '/files/my doc.abdoc' while the live location read
    // '/files/my%20doc.abdoc'. Every site comparing the two mismatched.
    for (final name in const [
      'plain.abdoc',
      'my doc.abdoc',
      'holiday 100%.abdoc',
      'a+b.abdoc',
      'note#1.abdoc',
    ]) {
      test('filesPath round-trips [$name]', () {
        final built = AppRoutes.filesPath('/folder/$name');
        expect(
          Uri.parse(built).toString(),
          built,
          reason: 'built route must already be in canonical URL form',
        );
      });
    }

    test('docFile, sheetFile and plaintextEditorPath encode too', () {
      expect(AppRoutes.docFile('/my doc.abdoc'), '/docs/my%20doc.abdoc');
      expect(
        AppRoutes.sheetFile('/my sheet.absheet'),
        '/sheets/my%20sheet.absheet',
      );
      expect(
        AppRoutes.plaintextEditorPath('/my notes.txt'),
        '/edit/my%20notes.txt',
      );
    });
  });

  group('AppRoutes.canonicalRoute', () {
    test('makes an unencoded route compare equal to the live location', () {
      expect(
        AppRoutes.canonicalRoute('/files/my doc.abdoc'),
        AppRoutes.canonicalRoute('/files/my%20doc.abdoc'),
      );
    });

    test('returns the input unchanged when it will not parse', () {
      expect(AppRoutes.canonicalRoute('http://[::bad'), 'http://[::bad');
    });
  });

  group('/files/:path delivers the real path to FileBrowserPage', () {
    Future<String?> pathSeenFor(WidgetTester tester, String location) async {
      String? seen;
      final router = GoRouter(
        initialLocation: '/files',
        routes: [
          GoRoute(
            path: AppRoutes.files,
            builder: (_, _) => const Scaffold(body: Text('root')),
            routes: [
              GoRoute(
                path: ':path(.*)',
                builder: (_, state) {
                  // Mirrors the real builder: go_router already decodes.
                  seen = state.pathParameters['path'];
                  return Scaffold(body: Text('leaf:$seen'));
                },
              ),
            ],
          ),
        ],
      );
      await tester.pumpWidget(MaterialApp.router(routerConfig: router));
      await tester.pumpAndSettle();
      router.go(location);
      await tester.pumpAndSettle();
      return seen;
    }

    testWidgets('a name with a space arrives decoded exactly once', (
      tester,
    ) async {
      final seen = await pathSeenFor(
        tester,
        AppRoutes.filesPath('/my doc.abdoc'),
      );
      expect(seen, '/my doc.abdoc');
    });

    // Decoding a second time threw FormatException on any name with a '%'.
    testWidgets('a name with a percent sign does not throw', (tester) async {
      final seen = await pathSeenFor(
        tester,
        AppRoutes.filesPath('/holiday 100%.abdoc'),
      );
      expect(seen, '/holiday 100%.abdoc');
    });

    testWidgets('a literal %20 in a name is preserved', (tester) async {
      final seen = await pathSeenFor(
        tester,
        AppRoutes.filesPath('/odd%20name.abdoc'),
      );
      expect(seen, '/odd%20name.abdoc');
    });
  });
}
