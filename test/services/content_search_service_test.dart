import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:quark/controllers/file_type_listing_cache.dart';
import 'package:quark/services/content_search_service.dart';
import 'package:quark/services/shared_http_client.dart';
import 'package:quark/utils/content_search_config.dart';

/// A client that answers from the test rather than the network.
class _StubClient extends http.BaseClient {
  _StubClient(this._respond);

  final Future<http.StreamedResponse> Function(http.BaseRequest request)
  _respond;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) =>
      _respond(request);
}

http.StreamedResponse _jsonResponse(String body) =>
    http.StreamedResponse(Stream.value(utf8.encode(body)), 200);

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('ContentSearchResult.fromJson', () {
    // The exact body returned by GET /api/v0/files/search/content. The
    // handler's payload is written directly by WrapApiRoute, so this is a bare
    // array with a `serial` key — not a {"data": …} envelope, and not
    // `deviceSerial`.
    const realResponse =
        '[{"serial":"","relPath":"something.txt.qdoc",'
        '"snippet":"\\u003cb\\u003e hello world\\u003c/b\\u003e"}]';

    test('parses the backend response shape', () {
      final decoded = jsonDecode(realResponse) as List;
      final results = decoded
          .whereType<Map<String, dynamic>>()
          .map(ContentSearchResult.fromJson)
          .toList();

      expect(results, hasLength(1));
      expect(results.single.relPath, 'something.txt.qdoc');
      expect(results.single.deviceSerial, '');
      expect(results.single.snippet, '<b> hello world</b>');
    });

    test('exposes filename and tag-stripped snippet for the result tile', () {
      final decoded = jsonDecode(realResponse) as List;
      final result = ContentSearchResult.fromJson(
        decoded.first as Map<String, dynamic>,
      );

      expect(result.filename, 'something.txt.qdoc');
      expect(result.plainSnippet, ' hello world');
    });

    test('reads nested paths down to the filename', () {
      final result = ContentSearchResult.fromJson({
        'serial': 'ABC123',
        'relPath': 'notes/2026/meeting.qdoc',
        'snippet': 'x',
      });

      expect(result.filename, 'meeting.qdoc');
      expect(result.deviceSerial, 'ABC123');
    });

    test('tolerates missing keys', () {
      final result = ContentSearchResult.fromJson({});

      expect(result.deviceSerial, '');
      expect(result.relPath, '');
      expect(result.snippet, '');
    });
  });

  /// #1780: retyping a query re-ran the FTS scan on the Quark. Recent answers
  /// are memoized, identical queries in flight share one request, and both
  /// are forgotten whenever the by-type listings are.
  group('memo', () {
    const body = '[{"serial":"","relPath":"a.qdoc","snippet":"<b>hi</b>"}]';
    var requests = 0;

    setUp(() {
      requests = 0;
      ContentSearchService.clearRecent();
      contentSearchHttpClientFactory = () => _StubClient((_) async {
        requests++;
        return _jsonResponse(body);
      });
    });
    tearDown(() {
      ContentSearchService.clearRecent();
      contentSearchHttpClientFactory = () => sharedHttpClient;
    });

    test('a repeated query is answered without a second request', () async {
      final first = await ContentSearchService.search('hello');
      final second = await ContentSearchService.search('hello');

      expect(requests, 1);
      expect(second, same(first));
      expect(second.single.relPath, 'a.qdoc');
    });

    test('a query is matched after trimming', () async {
      await ContentSearchService.search('hello');
      await ContentSearchService.search('  hello ');

      expect(requests, 1);
      expect(ContentSearchService.recent(' hello'), isNotNull);
    });

    test('a different query is a miss', () async {
      await ContentSearchService.search('hello');
      await ContentSearchService.search('world');

      expect(requests, 2);
    });

    test('an empty query is answered without a request', () async {
      expect(await ContentSearchService.search('   '), isEmpty);
      expect(requests, 0);
    });

    test('two identical queries in flight share one request', () async {
      final release = Completer<void>();
      contentSearchHttpClientFactory = () => _StubClient((_) async {
        requests++;
        await release.future;
        return _jsonResponse(body);
      });

      final first = ContentSearchService.search('hello');
      final second = ContentSearchService.search('hello');
      release.complete();
      final results = await Future.wait([first, second]);

      expect(requests, 1);
      expect(results.first, same(results.last));
    });

    test('clearRecent forgets the answer', () async {
      await ContentSearchService.search('hello');

      ContentSearchService.clearRecent();
      expect(ContentSearchService.recent('hello'), isNull);

      await ContentSearchService.search('hello');
      expect(requests, 2);
    });

    test('clearing the listing cache forgets the answer too', () async {
      await ContentSearchService.search('hello');

      FileTypeListingCache.instance.clear();
      await ContentSearchService.search('hello');

      expect(requests, 2);
    });

    test('the memo holds only the most recent queries', () async {
      const limit = ContentSearchConfig.recentQueryLimit;
      for (var i = 0; i <= limit; i++) {
        await ContentSearchService.search('q$i');
      }

      expect(ContentSearchService.recent('q0'), isNull);
      expect(ContentSearchService.recent('q1'), isNotNull);
      expect(ContentSearchService.recent('q$limit'), isNotNull);

      await ContentSearchService.search('q0');
      expect(requests, limit + 2);
    });

    test('a hit counts as recent use and survives eviction', () async {
      const limit = ContentSearchConfig.recentQueryLimit;
      for (var i = 0; i < limit; i++) {
        await ContentSearchService.search('q$i');
      }
      await ContentSearchService.search('q0');
      await ContentSearchService.search('new');

      expect(ContentSearchService.recent('q0'), isNotNull);
      expect(ContentSearchService.recent('q1'), isNull);
    });

    test('a failed search is not memoized', () async {
      contentSearchHttpClientFactory = () => _StubClient((_) async {
        requests++;
        return _jsonResponse('<html>not the api</html>');
      });

      expect(await ContentSearchService.search('hello'), isEmpty);
      expect(ContentSearchService.recent('hello'), isNull);

      await ContentSearchService.search('hello');
      expect(requests, 2);
    });
  });
}
