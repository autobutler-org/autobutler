import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/document_viewer_controller.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_widgets/quark_widgets.dart';

QuarkDocument _doc(String text) => QuarkDocument.parse(
  '{"type":"meta","version":1,"format":"docx"}\n'
  '{"type":"paragraph","runs":[{"text":"$text"}]}\n'
  '{"type":"end","truncated":false,"words":1,"characters":1,"paragraphs":1}\n',
);

String _text(QuarkDocument document) =>
    (document.blocks.single as QuarkDocumentParagraph).runs.single.text;

/// The document viewer's state (#2987): reading the block stream and
/// "Open as Quark doc".
void main() {
  test('loads the document from the path and device it was given', () async {
    final asked = <(String, String?)>[];
    final controller = DocumentViewerController(
      path: 'work/q3.docx',
      serial: 'usb1',
      loadDocument: (path, {serial}) async {
        asked.add((path, serial));
        return _doc('hello');
      },
    );
    addTearDown(controller.dispose);
    expect(controller.isLoading, isTrue);

    await controller.load();

    expect(asked, [('work/q3.docx', 'usb1')]);
    expect(controller.isLoading, isFalse);
    expect(controller.error, isNull);
    expect(_text(controller.document), 'hello');
  });

  test('a failed read keeps the error for the page to word', () async {
    final controller = DocumentViewerController(
      path: 'q3.docx',
      loadDocument: (path, {serial}) async => throw ApiException(400, 'x'),
    );
    addTearDown(controller.dispose);

    await controller.load();

    expect(controller.error, isA<ApiException>());
    expect(controller.isLoading, isFalse);
    expect(controller.document.isEmpty, isTrue);
  });

  test('a read overtaken by a newer one is dropped', () async {
    final first = Completer<QuarkDocument>();
    var calls = 0;
    final controller = DocumentViewerController(
      path: 'q3.docx',
      loadDocument: (path, {serial}) async =>
          ++calls == 1 ? first.future : _doc('second'),
    );
    addTearDown(controller.dispose);

    final stale = controller.load();
    await controller.load();
    first.complete(_doc('first'));
    await stale;

    expect(_text(controller.document), 'second');
  });

  test('Open as Quark doc passes overwrite and reports busy', () async {
    final done = Completer<DocumentConversion>();
    final asked = <bool>[];
    final controller = DocumentViewerController(
      path: 'work/q3.docx',
      loadDocument: (path, {serial}) async => _doc('x'),
      convertDocument: (path, {serial, overwrite = false}) {
        asked.add(overwrite);
        return done.future;
      },
    );
    addTearDown(controller.dispose);

    final converting = controller.openAsQuarkDoc(overwrite: true);
    expect(controller.isConverting, isTrue);
    done.complete((path: 'work/q3.qdoc', warnings: const <String>[]));

    expect((await converting).path, 'work/q3.qdoc');
    expect(asked, [true]);
    expect(controller.isConverting, isFalse);
  });

  test('a refused conversion throws and clears busy', () async {
    final controller = DocumentViewerController(
      path: 'q3.docx',
      convertDocument: (path, {serial, overwrite = false}) async =>
          throw ApiException(409, 'exists'),
    );
    addTearDown(controller.dispose);

    await expectLater(
      controller.openAsQuarkDoc(),
      throwsA(isA<ApiException>().having((e) => e.statusCode, 'status', 409)),
    );
    expect(controller.isConverting, isFalse);
  });
}
