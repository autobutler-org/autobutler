import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/content_search_service.dart';

void main() {
  group('ContentSearchResult.fromJson', () {
    // The exact body returned by GET /api/v0/cirrus/search/content. The
    // handler's payload is written directly by WrapApiRoute, so this is a bare
    // array with a `serial` key — not a {"data": …} envelope, and not
    // `deviceSerial`.
    const realResponse =
        '[{"serial":"","relPath":"something.txt.abdoc",'
        '"snippet":"\\u003cb\\u003e hello world\\u003c/b\\u003e"}]';

    test('parses the backend response shape', () {
      final decoded = jsonDecode(realResponse) as List;
      final results = decoded
          .whereType<Map<String, dynamic>>()
          .map(ContentSearchResult.fromJson)
          .toList();

      expect(results, hasLength(1));
      expect(results.single.relPath, 'something.txt.abdoc');
      expect(results.single.deviceSerial, '');
      expect(results.single.snippet, '<b> hello world</b>');
    });

    test('exposes filename and tag-stripped snippet for the result tile', () {
      final decoded = jsonDecode(realResponse) as List;
      final result = ContentSearchResult.fromJson(
        decoded.first as Map<String, dynamic>,
      );

      expect(result.filename, 'something.txt.abdoc');
      expect(result.plainSnippet, ' hello world');
    });

    test('reads nested paths down to the filename', () {
      final result = ContentSearchResult.fromJson({
        'serial': 'ABC123',
        'relPath': 'notes/2026/meeting.abdoc',
        'snippet': 'x',
      });

      expect(result.filename, 'meeting.abdoc');
      expect(result.deviceSerial, 'ABC123');
    });

    test('tolerates missing keys', () {
      final result = ContentSearchResult.fromJson({});

      expect(result.deviceSerial, '');
      expect(result.relPath, '');
      expect(result.snippet, '');
    });
  });
}
