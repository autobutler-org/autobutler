import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/file_kind.dart';
import 'package:quark/utils/presentation_viewer_files.dart';

/// Which files the presentation viewer opens (#3268).
void main() {
  test('opens PPTX and PPTM in any case', () {
    for (final name in ['q1.pptx', 'macros.pptm', 'LOUD.PPTX']) {
      expect(opensInPresentationViewer(name), isTrue, reason: name);
    }
  });

  test('leaves legacy PPT, Quark slides and others alone', () {
    for (final name in ['old.ppt', 'deck.qslide', 'notes.docx', 'show.key']) {
      expect(opensInPresentationViewer(name), isFalse, reason: name);
    }
    // .ppt is still labeled a presentation, on the generic viewer.
    expect(fileKindForName('old.ppt'), FileKind.slideshow);
    expect(fileKindForName('macros.pptm'), FileKind.slideshow);
  });
}
