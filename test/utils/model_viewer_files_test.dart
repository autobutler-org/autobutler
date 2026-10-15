import 'package:flutter_test/flutter_test.dart';
import 'package:quark/router.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/model_viewer_files.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// Which files the 3D model viewer takes (#2988), where the router sends
/// them, and the note shown when one falls back to its details.
void main() {
  test('STL, OBJ and PLY open in the model viewer, by extension', () {
    expect(modelFormatForName('prints/Part.STL'), QuarkModelFormat.stl);
    expect(modelFormatForName('scan.obj'), QuarkModelFormat.obj);
    expect(modelFormatForName('bunny.ply'), QuarkModelFormat.ply);
    for (final name in ['part.gltf', 'part.glb', 'part.3mf', 'stl', 'a.txt']) {
      expect(opensInModelViewer(name), isFalse, reason: name);
    }
  });

  test('modelViewerPath encodes the path and carries the serial', () {
    expect(
      AppRoutes.modelViewerPath('/my prints/part one.stl'),
      '/model/my%20prints/part%20one.stl',
    );
    expect(
      AppRoutes.modelViewerPath('a/b.obj', serial: 'usb 1'),
      '/model/a/b.obj?serial=usb+1',
    );
  });

  test('a /view link to a model opens the model viewer', () {
    expect(viewFileRedirect('prints/part.stl', null), '/model/prints/part.stl');
    expect(
      viewFileRedirect('scans/bunny.ply', 's1'),
      '/model/scans/bunny.ply?serial=s1',
    );
    // A format the viewer does not read stays with the generic viewer.
    expect(viewFileRedirect('models/part.glb', null), isNull);
  });

  test('every fallback reason has its own sentence', () {
    final sentences = {
      for (final issue in QuarkModelParseIssue.values)
        Errors.modelPreview(issue),
    };
    expect(sentences, hasLength(QuarkModelParseIssue.values.length));
    for (final sentence in sentences) {
      expect(sentence, endsWith('.'));
    }
  });
}
