import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/presentation_viewer_controller.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark/services/slides_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_slides/quark_slides.dart';

/// The presentation viewer's state (#3268): reading the deck, stepping
/// through it, the notes toggle, dismissing the warnings, and where a
/// picture is fetched from.
void main() {
  Presentation deck(int count) => Presentation(
    title: 'Deck',
    slides: [
      for (var i = 1; i <= count; i++) Slide(id: 's$i', notes: 'Notes $i'),
    ],
  );

  SlidesPreview preview(int count) => (
    deck: deck(count),
    warnings: const [(slide: 2, message: 'SmartArt not shown')],
  );

  PresentationViewerController controllerFor({
    LoadSlidesPreviewFn? load,
    int startIndex = 0,
  }) => PresentationViewerController(
    path: 'talks/q1.pptx',
    serial: 'usb1',
    startIndex: startIndex,
    loadSlides:
        load ??
        (path, {serial}) async {
          expect(path, 'talks/q1.pptx');
          expect(serial, 'usb1');
          return preview(3);
        },
    importPowerPoint: (path, {serial}) async => (
      path: 'talks/q1.qslide',
      slides: 3,
      warnings: const <PowerPointImportWarning>[],
    ),
    partUrl: (part, {required filePath, serial}) =>
        Uri.parse('https://quark/part?p=$part&f=$filePath&s=$serial'),
  );

  test('reads the deck and opens at the slide asked for', () async {
    final c = controllerFor(startIndex: 1);
    expect(c.isLoading, isTrue);
    await c.load();
    expect(c.isLoading, isFalse);
    expect(c.error, isNull);
    expect(c.currentSlide?.id, 's2');
    expect(c.position, 'Slide 2 of 3');
    expect(c.warnings.single.message, 'SmartArt not shown');
    c.dispose();
  });

  test('a start past the end is the last slide', () async {
    final c = controllerFor(startIndex: 9);
    await c.load();
    expect(c.index, 2);
    expect(c.isLast, isTrue);
    c.dispose();
  });

  test('steps through the slides and stops at either end', () async {
    final c = controllerFor();
    await c.load();
    expect(c.isFirst, isTrue);
    c.previous();
    expect(c.index, 0);
    c.next();
    c.next();
    c.next();
    expect(c.index, 2);
    c.goTo(1);
    expect(c.currentSlide?.id, 's2');
    c.dispose();
  });

  test('a failed read keeps the error for the page to word', () async {
    final c = controllerFor(
      load: (path, {serial}) async => throw ApiException(400, 'preview'),
    );
    await c.load();
    expect(c.error, isA<ApiException>());
    expect(c.deck, isNull);
    expect(c.isLoading, isFalse);
    c.dispose();
  });

  test('a read overtaken by a newer one is dropped', () async {
    final first = Completer<SlidesPreview>();
    var calls = 0;
    final c = controllerFor(
      load: (path, {serial}) => ++calls == 1 ? first.future : _now(preview(2)),
    );
    final stale = c.load();
    await c.load();
    first.complete(preview(5));
    await stale;
    expect(c.slides, hasLength(2));
    c.dispose();
  });

  test('toggles the notes and dismisses the warnings', () async {
    final c = controllerFor();
    await c.load();
    expect(c.notesVisible, isFalse);
    c.toggleNotes();
    expect(c.notesVisible, isTrue);
    c.dismissWarnings();
    expect(c.warnings, isEmpty);
    c.dispose();
  });

  test('fetches only pictures inside the file', () {
    final c = controllerFor();
    expect(
      c.imageUrl('part:ppt/media/image3.png').toString(),
      'https://quark/part?p=ppt/media/image3.png&f=talks/q1.pptx&s=usb1',
    );
    expect(c.imageUrl('photos/beach.jpg'), isNull);
    expect(c.imageUrl('asset:abc'), isNull);
    c.dispose();
  });

  test('imports the presentation beside itself', () async {
    final c = controllerFor();
    final imported = await c.importAsQuarkSlides();
    expect(imported.path, 'talks/q1.qslide');
    c.dispose();
  });
}

Future<SlidesPreview> _now(SlidesPreview value) async => value;
