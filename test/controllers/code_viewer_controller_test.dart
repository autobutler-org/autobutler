import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/code_viewer_controller.dart';
import 'package:quark/services/code_viewer_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The code viewer's state (#2991): a file read a page at a time, split into
/// lines across page boundaries, and searched as far as it has been read.
void main() {
  /// A reader over [content] that honors the range it is asked for, and
  /// records every start it was asked for in [starts].
  ReadCodeRangeFn readerOver(
    String content, {
    List<int>? starts,
    bool reportsSize = true,
  }) {
    final bytes = Uint8List.fromList(utf8.encode(content));
    return (path, {required start, required length, serial}) async {
      starts?.add(start);
      final from = start.clamp(0, bytes.length);
      final to = (start + length).clamp(0, bytes.length);
      return (
        bytes: Uint8List.sublistView(bytes, from, to),
        totalSize: reportsSize ? bytes.length : null,
      );
    };
  }

  CodeViewerController build(
    ReadCodeRangeFn readRange, {
    String path = 'src/main.go',
    int pageBytes = 16,
    int maxLineBytes = 64,
  }) {
    final controller = CodeViewerController(
      path: path,
      readRange: readRange,
      pageBytes: pageBytes,
      maxLineBytes: maxLineBytes,
    );
    addTearDown(controller.dispose);
    return controller;
  }

  test('a file smaller than a page loads whole, in one request', () async {
    final starts = <int>[];
    final controller = build(readerOver('a\nb\n', starts: starts));
    expect(controller.isLoading, isTrue);

    await controller.load();

    expect(controller.isLoading, isFalse);
    expect(controller.lines, ['a', 'b']);
    expect(controller.hasMore, isFalse);
    expect(controller.totalSize, 4);
    expect(starts, [0]);
  });

  test('an empty file is no lines and nothing more', () async {
    final controller = build(readerOver(''));
    await controller.load();
    expect(controller.lines, isEmpty);
    expect(controller.hasMore, isFalse);
    expect(controller.error, isNull);
  });

  test('only the first page is read until more is asked for', () async {
    final starts = <int>[];
    final content = List.generate(20, (i) => 'line $i').join('\n');
    final controller = build(readerOver(content, starts: starts));

    await controller.load();

    expect(starts, [0]);
    expect(controller.hasMore, isTrue);
    expect(controller.loadedBytes, 16);
    // "line 2" straddles the page boundary, so it is held back.
    expect(controller.lines, ['line 0', 'line 1']);
  });

  test('pages join into the file\'s lines, whatever the page size', () async {
    final content = 'é ö first\r\nsecond ✓ line\n\nlast without newline';
    for (final pageBytes in [1, 2, 3, 5, 7, 64, 1024]) {
      final controller = build(readerOver(content), pageBytes: pageBytes);
      await controller.load();
      while (controller.hasMore) {
        await controller.loadMore();
      }
      expect(controller.lines, [
        'é ö first',
        'second ✓ line',
        '',
        'last without newline',
      ], reason: 'pageBytes $pageBytes');
    }
  });

  test('a Quark that does not report a size still pages to the end', () async {
    final content = List.generate(10, (i) => 'row $i').join('\n');
    final controller = build(readerOver(content, reportsSize: false));
    await controller.load();
    expect(controller.hasMore, isTrue);
    while (controller.hasMore) {
      await controller.loadMore();
    }
    expect(controller.lines.length, 10);
    expect(controller.lines.last, 'row 9');
  });

  test('a run with no newline is cut into lines of bounded length', () async {
    final controller = build(
      readerOver('x' * 100),
      pageBytes: 16,
      maxLineBytes: 32,
    );
    await controller.load();
    while (controller.hasMore) {
      await controller.loadMore();
    }
    expect(controller.lines.join(), 'x' * 100);
    expect(controller.lines.every((line) => line.length <= 32), isTrue);
    expect(controller.lines.length, greaterThan(1));
  });

  test('a failed first page is an error, and load tries again', () async {
    var fail = true;
    final good = readerOver('ok\n');
    final controller = build((path, {required start, required length, serial}) {
      if (fail) throw const ApiException(500, 'boom');
      return good(path, start: start, length: length, serial: serial);
    });

    await controller.load();
    expect(controller.error, isA<ApiException>());
    expect(controller.lines, isEmpty);

    fail = false;
    await controller.load();
    expect(controller.error, isNull);
    expect(controller.lines, ['ok']);
  });

  test('a failed later page keeps what was read and can be retried', () async {
    var fail = false;
    final content = List.generate(20, (i) => 'line $i').join('\n');
    final good = readerOver(content);
    final controller = build((path, {required start, required length, serial}) {
      if (fail) throw const ApiException(500, 'boom');
      return good(path, start: start, length: length, serial: serial);
    });
    await controller.load();
    final before = controller.lines;

    fail = true;
    await controller.loadMore();
    expect(controller.loadMoreError, isA<ApiException>());
    expect(controller.lines, before);
    expect(controller.hasMore, isTrue);
    expect(controller.error, isNull);

    fail = false;
    await controller.loadMore();
    expect(controller.loadMoreError, isNull);
    expect(controller.lines.length, greaterThan(before.length));
  });

  test('a second loadMore while one is out does not read twice', () async {
    final starts = <int>[];
    final content = List.generate(20, (i) => 'line $i').join('\n');
    final controller = build(readerOver(content, starts: starts));
    await controller.load();

    final first = controller.loadMore();
    final second = controller.loadMore();
    expect(controller.isLoadingMore, isTrue);
    await Future.wait([first, second]);

    expect(starts, [0, 16]);
  });

  test('the language comes from the name, or a shebang', () async {
    final go = build(readerOver('package main\n'));
    await go.load();
    expect(go.language, CodeLanguage.go);

    final script = build(
      readerOver('#!/usr/bin/env python3\nprint(1)\n'),
      path: 'bin/tool',
      pageBytes: 1024,
    );
    await script.load();
    expect(script.language, CodeLanguage.python);
  });

  test('search finds hits, steps through them and wraps around', () async {
    final controller = build(
      readerOver('alpha\nbeta Alpha\ngamma\n'),
      pageBytes: 1024,
    );
    await controller.load();

    controller.setSearchQuery('alpha');
    expect(controller.matches, const [
      CodeSearchMatch(line: 0, start: 0, end: 5),
      CodeSearchMatch(line: 1, start: 5, end: 10),
    ]);
    expect(controller.activeMatch, 0);

    controller.nextMatch();
    expect(controller.activeMatch, 1);
    controller.nextMatch();
    expect(controller.activeMatch, 0);
    controller.previousMatch();
    expect(controller.activeMatch, 1);

    controller.setSearchQuery('nothing here');
    expect(controller.matches, isEmpty);
    expect(controller.activeMatch, isNull);
    controller.nextMatch();
    expect(controller.activeMatch, isNull);
  });

  test('a page that lands adds its hits and keeps the one in view', () async {
    final content = List.generate(20, (i) => 'hit $i').join('\n');
    final controller = build(readerOver(content));
    await controller.load();
    controller.setSearchQuery('hit');
    controller.nextMatch();
    final found = controller.matches.length;

    await controller.loadMore();

    expect(controller.matches.length, greaterThan(found));
    expect(controller.activeMatch, 1);
  });

  test('wrap is off until turned on, and notifies once per change', () {
    final controller = build(readerOver(''));
    var notified = 0;
    controller.addListener(() => notified++);
    expect(controller.wrap, isFalse);
    controller.setWrap(true);
    controller.setWrap(true);
    expect(controller.wrap, isTrue);
    expect(notified, 1);
  });

  test('a page that lands after dispose is dropped', () async {
    final gate = Completer<CodeViewerRange>();
    final controller = CodeViewerController(
      path: 'a.log',
      readRange: (path, {required start, required length, serial}) =>
          gate.future,
    );
    final loading = controller.load();
    controller.dispose();
    gate.complete((
      bytes: Uint8List.fromList(utf8.encode('x\n')),
      totalSize: 2,
    ));
    await loading;
    expect(controller.lines, isEmpty);
  });
}
