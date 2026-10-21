import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/controllers/document_viewer_controller.dart';
import 'package:quark/pages/document_viewer_page.dart';
import 'package:quark/router.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/document_viewer/document_viewer_image.dart';
import 'package:quark_widgets/quark_widgets.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _sample =
    '{"type":"meta","version":1,"format":"docx","title":"Q3"}\n'
    '{"type":"heading","level":1,"runs":[{"text":"Summary"}]}\n'
    '{"type":"paragraph","runs":[{"text":"See "},'
    '{"text":"the site","link":"https://example.com/q3"},{"text":" or "},'
    '{"text":"this","link":"javascript:alert(1)"}]}\n'
    '{"type":"table","rows":[{"cells":[{"colSpan":1,"rowSpan":1,"blocks":'
    '[{"type":"paragraph","runs":[{"text":"Revenue"}]}]}]}]}\n'
    '{"type":"image","part":"word/media/image1.png","alt":"Chart"}\n'
    '{"type":"end","truncated":false,"words":6,"characters":30,'
    '"paragraphs":3}\n';

/// The document viewer page (#2987): the document drawn from the Quark's
/// block stream, which links it follows, and "Open as Quark doc".
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues({}));

  final opened = <Uri>[];
  setUp(opened.clear);

  Future<GoRouter> pumpViewer(
    WidgetTester tester, {
    LoadDocumentFn? load,
    ConvertDocumentFn? convert,
    String filePath = 'work/q3.docx',
    Size size = const Size(1280, 800),
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    final controller = DocumentViewerController(
      path: filePath,
      loadDocument:
          load ?? (path, {serial}) async => QuarkDocument.parse(_sample),
      convertDocument:
          convert ??
          (path, {serial, overwrite = false}) async =>
              (path: 'work/q3.qdoc', warnings: const <String>[]),
    );
    addTearDown(controller.dispose);
    final router = GoRouter(
      initialLocation: AppRoutes.documentViewerPath(filePath),
      routes: [
        GoRoute(
          path: '${AppRoutes.files}/:path(.*)',
          builder: (_, state) =>
              Scaffold(body: Text('folder:${state.pathParameters['path']}')),
        ),
        GoRoute(
          path: '${AppRoutes.docs}/:path(.*)',
          builder: (_, state) =>
              Scaffold(body: Text('qdoc:${state.pathParameters['path']}')),
        ),
        GoRoute(
          path: '${AppRoutes.documentViewer}/:path(.*)',
          builder: (_, state) => DocumentViewerPage(
            filePath: state.pathParameters['path'] ?? '',
            controller: controller,
            openExternal: opened.add,
          ),
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      MaterialApp.router(
        theme: QuarkTheme.from(
          QuarkThemeColor.classic.tokensFor(Brightness.dark),
          Brightness.dark,
        ),
        routerConfig: router,
      ),
    );
    await tester.pumpAndSettle();
    return router;
  }

  for (final size in const [Size(360, 640), Size(1280, 800)]) {
    testWidgets('draws the document at ${size.width}', (tester) async {
      await pumpViewer(tester, size: size);
      expect(tester.takeException(), isNull);
      expect(find.text('q3.docx'), findsOneWidget);
      expect(find.text('Summary'), findsOneWidget);
      expect(find.text('Revenue'), findsOneWidget);
      expect(
        find.byKey(const ValueKey('document_viewer_open_as_qdoc')),
        findsOneWidget,
      );
      expect(
        find.byKey(const ValueKey('document_viewer_download')),
        findsOneWidget,
      );
    });
  }

  testWidgets('an image loads through the part endpoint', (tester) async {
    await pumpViewer(tester);
    final image = tester.widget<DocumentViewerImage>(
      find.byType(DocumentViewerImage),
    );
    expect(image.image.part, 'word/media/image1.png');
    final network =
        tester.widget<Image>(find.byType(Image).first).image as NetworkImage;
    final url = Uri.parse(network.url);
    expect(url.path, '/api/v0/previews/part');
    expect(url.queryParameters['part'], 'word/media/image1.png');
    expect(url.queryParameters['filePath'], 'work/q3.docx');
  });

  testWidgets('a web link opens; a script link does nothing', (tester) async {
    await pumpViewer(tester);
    await tester.tap(find.byKey(const ValueKey('document_link_0')));
    await tester.tap(find.byKey(const ValueKey('document_link_1')));
    await tester.pumpAndSettle();
    expect(opened, [Uri.parse('https://example.com/q3')]);
  });

  testWidgets('a refused read shows the error and Retry reloads', (
    tester,
  ) async {
    var fail = true;
    await pumpViewer(
      tester,
      load: (path, {serial}) async {
        if (fail) throw ApiException(400, 'not a document');
        return QuarkDocument.parse(_sample);
      },
    );
    expect(
      find.text(Errors.message(ApiException(400, ''), 'open the document')),
      findsOneWidget,
    );
    fail = false;
    await tester.tap(find.byKey(const ValueKey('document_view_retry')));
    await tester.pumpAndSettle();
    expect(find.text('Summary'), findsOneWidget);
  });

  testWidgets('Open as Quark doc opens the new .qdoc in the editor', (
    tester,
  ) async {
    final router = await pumpViewer(tester);
    await tester.tap(
      find.byKey(const ValueKey('document_viewer_open_as_qdoc')),
    );
    await tester.pumpAndSettle();
    expect(
      router.routeInformationProvider.value.uri.path,
      '/docs/work/q3.qdoc',
    );
    expect(find.text('qdoc:work/q3.qdoc'), findsOneWidget);
  });

  testWidgets('an existing .qdoc asks first, and Replace overwrites it', (
    tester,
  ) async {
    final asked = <bool>[];
    await pumpViewer(
      tester,
      convert: (path, {serial, overwrite = false}) async {
        asked.add(overwrite);
        if (!overwrite) throw ApiException(409, 'exists');
        return (path: 'work/q3.qdoc', warnings: const ['Images left out.']);
      },
    );
    await tester.tap(
      find.byKey(const ValueKey('document_viewer_open_as_qdoc')),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining('"q3.qdoc" is already here'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('document_conflict_replace')));
    await tester.pumpAndSettle();
    expect(asked, [false, true]);
    expect(find.text('qdoc:work/q3.qdoc'), findsOneWidget);
    expect(find.textContaining('Images left out.'), findsOneWidget);
  });

  testWidgets('an existing .qdoc can be opened as it is', (tester) async {
    final asked = <bool>[];
    await pumpViewer(
      tester,
      convert: (path, {serial, overwrite = false}) async {
        asked.add(overwrite);
        throw ApiException(409, 'exists');
      },
    );
    await tester.tap(
      find.byKey(const ValueKey('document_viewer_open_as_qdoc')),
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const ValueKey('document_conflict_open_existing')),
    );
    await tester.pumpAndSettle();
    expect(asked, [false]);
    expect(find.text('qdoc:work/q3.qdoc'), findsOneWidget);
  });

  testWidgets('a refused conversion says so and stays put', (tester) async {
    await pumpViewer(
      tester,
      convert: (path, {serial, overwrite = false}) async =>
          throw ApiException(403, 'read-only'),
    );
    await tester.tap(
      find.byKey(const ValueKey('document_viewer_open_as_qdoc')),
    );
    await tester.pumpAndSettle();
    expect(
      find.text(Errors.message(ApiException(403, ''), 'convert the document')),
      findsOneWidget,
    );
    expect(find.text('Summary'), findsOneWidget);
  });

  testWidgets('back with nothing underneath goes to the folder', (
    tester,
  ) async {
    final router = await pumpViewer(tester);
    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();
    expect(router.routeInformationProvider.value.uri.path, '/files/work');
  });
}
