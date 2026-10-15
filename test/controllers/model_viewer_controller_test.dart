import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/code_viewer_controller.dart';
import 'package:quark/controllers/model_viewer_controller.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// One triangle, ready-made, for a fake parser to hand back.
final QuarkModelMesh triangle = QuarkModelMesh.fromTriangles(
  positions: Float32List.fromList([0, 0, 0, 1, 0, 0, 0, 1, 0]),
  indices: Uint32List.fromList([0, 1, 2]),
);

/// `ModelViewerController` (#2988): what it reads, what it refuses to
/// download, and how each failure surfaces.
void main() {
  late List<({int start, int length})> reads;
  late List<(int, QuarkModelFormat)> parses;

  setUp(() {
    reads = [];
    parses = [];
  });

  /// A Quark serving [size] bytes, saying how big the file is unless
  /// [reportsSize] is false.
  ReadCodeRangeFn quark(int size, {bool reportsSize = true}) =>
      (path, {required start, required length, serial}) async {
        reads.add((start: start, length: length));
        final end = (start + length).clamp(0, size);
        return (
          bytes: Uint8List(end - start),
          totalSize: reportsSize ? size : null,
        );
      };

  Future<QuarkModelMesh> parseOk(
    Uint8List bytes,
    QuarkModelFormat format,
    QuarkModelLimits limits,
  ) async {
    parses.add((bytes.length, format));
    return triangle;
  }

  ModelViewerController controller({
    String path = 'prints/part.stl',
    required ReadCodeRangeFn readRange,
    ParseModelFn? parse,
    int maxBytes = 1000,
  }) {
    final controller = ModelViewerController(
      path: path,
      readRange: readRange,
      parse: parse ?? parseOk,
      limits: QuarkModelLimits(maxBytes: maxBytes),
    );
    addTearDown(controller.dispose);
    return controller;
  }

  test('reads the whole file once its size fits, and parses it', () async {
    final viewer = controller(readRange: quark(500));
    expect(viewer.isLoading, isTrue);
    await viewer.load();
    expect(viewer.isLoading, isFalse);
    expect(viewer.mesh, same(triangle));
    expect(viewer.error, isNull);
    expect(viewer.fallback, isNull);
    expect(viewer.fileSize, 500);
    expect(reads, [(start: 0, length: 1), (start: 0, length: 1001)]);
    expect(parses, [(500, QuarkModelFormat.stl)]);
  });

  test('a file the Quark says is over the cap is never downloaded', () async {
    final viewer = controller(readRange: quark(5000));
    await viewer.load();
    expect(viewer.fallback, QuarkModelParseIssue.tooLarge);
    expect(viewer.mesh, isNull);
    expect(reads, [(start: 0, length: 1)]);
    expect(parses, isEmpty);
  });

  test('without a size, the read stops one byte past the cap', () async {
    final viewer = controller(readRange: quark(5000, reportsSize: false));
    await viewer.load();
    expect(viewer.fallback, QuarkModelParseIssue.tooLarge);
    expect(reads.last.length, 1001);
    expect(parses, isEmpty);
  });

  test('a file the parser refuses falls back with its reason', () async {
    final viewer = controller(
      path: 'scans/bunny.PLY',
      readRange: quark(10),
      parse: (bytes, format, limits) async {
        parses.add((bytes.length, format));
        throw const QuarkModelParseException(
          QuarkModelParseIssue.tooManyTriangles,
        );
      },
    );
    await viewer.load();
    expect(parses.single.$2, QuarkModelFormat.ply);
    expect(viewer.fallback, QuarkModelParseIssue.tooManyTriangles);
    expect(viewer.error, isNull);
  });

  test('a failed request is an error with Retry, not a fallback', () async {
    var fail = true;
    final viewer = controller(
      readRange: (path, {required start, required length, serial}) async {
        if (fail) throw ApiException(500, 'boom');
        return (bytes: Uint8List(3), totalSize: 3);
      },
    );
    await viewer.load();
    expect(viewer.error, isA<ApiException>());
    expect(viewer.fallback, isNull);
    fail = false;
    await viewer.load();
    expect(viewer.error, isNull);
    expect(viewer.mesh, same(triangle));
  });

  test(
    'a file the viewer has no format for falls back as unreadable',
    () async {
      final viewer = controller(path: 'models/part.glb', readRange: quark(10));
      await viewer.load();
      expect(viewer.fallback, QuarkModelParseIssue.malformed);
      expect(reads, isEmpty);
    },
  );

  test('only the latest load lands', () async {
    final viewer = controller(readRange: quark(10));
    final first = viewer.load();
    final second = viewer.load();
    await Future.wait([first, second]);
    expect(viewer.mesh, same(triangle));
    expect(parses, hasLength(1));
  });

  test('wireframe toggles and notifies once per change', () {
    final viewer = controller(readRange: quark(10));
    var notified = 0;
    viewer.addListener(() => notified++);
    viewer.setWireframe(true);
    viewer.setWireframe(true);
    expect(viewer.wireframe, isTrue);
    expect(notified, 1);
  });
}
