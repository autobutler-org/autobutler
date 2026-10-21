import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/controllers/presentation_viewer_controller.dart';
import 'package:quark/controllers/slides_controller.dart';
import 'package:quark/pages/presentation_viewer_page.dart';
import 'package:quark/router.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark/services/slides_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/slides/slide_image.dart';
import 'package:quark_slides/quark_slides.dart';
import 'package:quark_widgets/quark_widgets.dart';
import 'package:shared_preferences/shared_preferences.dart';

SlidesPreview _preview() => (
  deck: Presentation(
    title: 'Q1',
    slides: [
      Slide(
        id: 's1',
        notes: 'Welcome everyone',
        elements: [
          ImageElement(
            id: 'i1',
            frame: ElementFrame(x: 0, y: 0, width: 200, height: 100),
            source: 'part:ppt/media/image1.png',
          ),
        ],
      ),
      Slide(id: 's2'),
      Slide(id: 's3', notes: 'Questions'),
    ],
  ),
  warnings: const [
    (slide: 2, message: 'SmartArt not shown'),
    (slide: 3, message: 'SmartArt not shown'),
  ],
);

/// The presentation viewer page (#3268): a PowerPoint file read on the
/// Quark, stepped through a slide at a time, its notes, its warnings,
/// Present and "Open as Quark slides".
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues({}));

  Future<GoRouter> pumpViewer(
    WidgetTester tester, {
    LoadSlidesPreviewFn? load,
    ImportPowerPointFn? import,
    String location = '/presentation/talks/q1.pptx',
    Size size = const Size(1280, 800),
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    final router = GoRouter(
      initialLocation: location,
      routes: [
        GoRoute(
          path: '${AppRoutes.files}/:path(.*)',
          builder: (_, state) =>
              Scaffold(body: Text('folder:${state.pathParameters['path']}')),
        ),
        GoRoute(
          path: '${AppRoutes.slides}/:path(.*)/present',
          builder: (_, state) => Scaffold(
            body: Text(
              'present:${state.pathParameters['path']}:'
              '${state.uri.queryParameters['slide']}:'
              '${(state.extra as Presentation?)?.title}',
            ),
          ),
        ),
        GoRoute(
          path: '${AppRoutes.slides}/:path(.*)',
          builder: (_, state) =>
              Scaffold(body: Text('qslide:${state.pathParameters['path']}')),
        ),
        GoRoute(
          path: '${AppRoutes.presentationViewer}/:path(.*)',
          builder: (_, state) {
            final path = state.pathParameters['path'] ?? '';
            final controller = PresentationViewerController(
              path: path,
              startIndex:
                  (int.tryParse(state.uri.queryParameters['slide'] ?? '') ??
                      1) -
                  1,
              loadSlides: load ?? (path, {serial}) async => _preview(),
              importPowerPoint:
                  import ??
                  (path, {serial}) async => (
                    path: 'talks/q1.qslide',
                    slides: 3,
                    warnings: const <PowerPointImportWarning>[],
                  ),
              partUrl: (part, {required filePath, serial}) => Uri.parse(
                'https://quark.local/api/v0/previews/part?part=$part'
                '&filePath=$filePath',
              ),
            );
            addTearDown(controller.dispose);
            return PresentationViewerPage(
              filePath: path,
              controller: controller,
            );
          },
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

  Finder key(String name) => find.byKey(ValueKey(name));

  for (final size in const [Size(360, 640), Size(1280, 800)]) {
    testWidgets('draws the presentation at ${size.width}', (tester) async {
      await pumpViewer(tester, size: size);
      expect(tester.takeException(), isNull);
      expect(find.text('q1.pptx'), findsOneWidget);
      expect(find.text('Slide 1 of 3'), findsOneWidget);
      expect(key('presentation_viewer_slide'), findsOneWidget);
      expect(key('presentation_viewer_present'), findsOneWidget);
      expect(key('presentation_viewer_open_as_qslide'), findsOneWidget);
      expect(key('presentation_viewer_download'), findsOneWidget);
      // The strip needs a wide window; a phone steps with the arrows.
      expect(
        key('presentation_viewer_strip'),
        size.width < 900 ? findsNothing : findsOneWidget,
      );
      // Each kind of warning once, with the slides it was on.
      expect(key('presentation_viewer_warning_0'), findsOneWidget);
      expect(key('presentation_viewer_warning_1'), findsNothing);
      expect(find.text('SmartArt not shown (Slides 2 and 3)'), findsOneWidget);
    });
  }

  testWidgets('steps through the slides with the arrows and the strip', (
    tester,
  ) async {
    await pumpViewer(tester);
    await tester.tap(key('presentation_viewer_next'));
    await tester.pumpAndSettle();
    expect(find.text('Slide 2 of 3'), findsOneWidget);
    await tester.tap(key('presentation_viewer_thumb_2'));
    await tester.pumpAndSettle();
    expect(find.text('Slide 3 of 3'), findsOneWidget);
    await tester.tap(key('presentation_viewer_previous'));
    await tester.pumpAndSettle();
    expect(find.text('Slide 2 of 3'), findsOneWidget);
  });

  testWidgets('opens at the slide the URL names', (tester) async {
    await pumpViewer(tester, location: '/presentation/talks/q1.pptx?slide=3');
    expect(find.text('Slide 3 of 3'), findsOneWidget);
  });

  testWidgets('shows the speaker notes behind the toggle', (tester) async {
    await pumpViewer(tester, size: const Size(360, 640));
    expect(find.text('Welcome everyone'), findsNothing);
    await tester.tap(key('presentation_viewer_notes'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(find.text('Welcome everyone'), findsOneWidget);
  });

  testWidgets('the warnings can be dismissed', (tester) async {
    await pumpViewer(tester);
    await tester.tap(key('presentation_viewer_warnings_dismiss'));
    await tester.pumpAndSettle();
    expect(key('presentation_viewer_warnings'), findsNothing);
  });

  testWidgets('a picture loads from inside the file', (tester) async {
    await pumpViewer(tester);
    final images = tester.widgetList<SlideImage>(find.byType(SlideImage));
    expect(images, isNotEmpty);
    final url = Uri.parse((images.first.image as NetworkImage).url);
    expect(url.path, '/api/v0/previews/part');
    expect(url.queryParameters['part'], 'ppt/media/image1.png');
    expect(url.queryParameters['filePath'], 'talks/q1.pptx');
  });

  testWidgets('Present hands the deck read to the presenter', (tester) async {
    await pumpViewer(tester);
    await tester.tap(key('presentation_viewer_next'));
    await tester.pumpAndSettle();
    await tester.tap(key('presentation_viewer_present'));
    await tester.pumpAndSettle();
    expect(find.text('present:talks/q1.pptx:2:Q1'), findsOneWidget);
  });

  testWidgets('Open as Quark slides imports and opens the copy', (
    tester,
  ) async {
    final imported = <String>[];
    await pumpViewer(
      tester,
      import: (path, {serial}) async {
        imported.add(path);
        return (
          path: 'talks/q1.qslide',
          slides: 3,
          warnings: const <PowerPointImportWarning>[],
        );
      },
    );
    await tester.tap(key('presentation_viewer_open_as_qslide'));
    await tester.pumpAndSettle();
    expect(imported, ['talks/q1.pptx']);
    expect(find.text('qslide:talks/q1.qslide'), findsOneWidget);
  });

  testWidgets('a file the Quark cannot read says so in a sentence', (
    tester,
  ) async {
    await pumpViewer(
      tester,
      load: (path, {serial}) async => throw ApiException(400, 'preview'),
    );
    expect(
      find.text(
        Errors.message(ApiException(400, 'x'), 'open the presentation'),
      ),
      findsOneWidget,
    );
    expect(find.textContaining('ApiException'), findsNothing);
  });

  testWidgets('back with nothing underneath goes to the folder', (
    tester,
  ) async {
    await pumpViewer(tester);
    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();
    expect(find.text('folder:talks'), findsOneWidget);
  });
}
