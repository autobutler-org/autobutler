import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/code_viewer_controller.dart';
import 'package:quark/controllers/markdown_viewer_controller.dart';
import 'package:quark/services/code_viewer_service.dart';

/// The Markdown viewer's state (#2987): one bounded read, the source toggle,
/// and remote images staying off until asked for.
void main() {
  ReadCodeRangeFn readerOver(String content, {List<int>? asked}) {
    final bytes = Uint8List.fromList(utf8.encode(content));
    return (path, {required start, required length, serial}) async {
      asked?.add(length);
      return (
        bytes: Uint8List.sublistView(
          bytes,
          start.clamp(0, bytes.length),
          (start + length).clamp(0, bytes.length),
        ),
        totalSize: bytes.length,
      );
    };
  }

  test('load reads the file once, bounded, and parses it', () async {
    final asked = <int>[];
    final controller = MarkdownViewerController(
      path: 'a.md',
      readRange: readerOver('# Title\n\nBody\n', asked: asked),
      maxBytes: 4096,
    );
    addTearDown(controller.dispose);
    expect(controller.isLoading, isTrue);

    await controller.load();

    expect(asked, [4096]);
    expect(controller.isLoading, isFalse);
    expect(controller.error, isNull);
    expect(controller.document.isEmpty, isFalse);
    expect(controller.isTruncated, isFalse);
  });

  test('a file past the cap is read up to it and marked truncated', () async {
    final controller = MarkdownViewerController(
      path: 'a.md',
      readRange: readerOver('line\n' * 100),
      maxBytes: 64,
    );
    addTearDown(controller.dispose);

    await controller.load();

    expect(controller.loadedBytes, 64);
    expect(controller.totalSize, 500);
    expect(controller.isTruncated, isTrue);
  });

  test('a failed read keeps the error and a retry clears it', () async {
    var fail = true;
    final ok = readerOver('hi');
    final controller = MarkdownViewerController(
      path: 'a.md',
      readRange: (path, {required start, required length, serial}) => fail
          ? throw Exception('boom')
          : ok(path, start: start, length: length, serial: serial),
    );
    addTearDown(controller.dispose);

    await controller.load();
    expect(controller.error, isNotNull);
    expect(controller.isLoading, isFalse);

    fail = false;
    await controller.load();
    expect(controller.error, isNull);
    expect(controller.document.isEmpty, isFalse);
  });

  test('a read that lands after dispose changes nothing', () async {
    final gate = Completer<CodeViewerRange>();
    final controller = MarkdownViewerController(
      path: 'a.md',
      readRange: (path, {required start, required length, serial}) =>
          gate.future,
    );
    final pending = controller.load();
    controller.dispose();
    gate.complete((bytes: Uint8List.fromList(utf8.encode('x')), totalSize: 1));
    await pending;

    expect(controller.document.isEmpty, isTrue);
  });

  test('the source toggle and remote images notify once per change', () {
    final controller = MarkdownViewerController(
      path: 'a.md',
      readRange: readerOver(''),
    );
    addTearDown(controller.dispose);
    var notified = 0;
    controller.addListener(() => notified++);

    expect(controller.showSource, isFalse);
    controller.setShowSource(true);
    controller.setShowSource(true);
    expect(controller.showSource, isTrue);
    expect(notified, 1);

    expect(controller.allowedRemoteImages, isEmpty);
    controller.allowRemoteImage('https://example.com/a.png');
    controller.allowRemoteImage('https://example.com/a.png');
    expect(controller.allowedRemoteImages, {'https://example.com/a.png'});
    expect(notified, 2);
  });
}
