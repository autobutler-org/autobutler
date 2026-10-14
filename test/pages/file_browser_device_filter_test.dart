import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/files_service.dart';

/// The Files device chips send their selection as `serial=` values; the
/// internal drive's is blank and must survive into the query (#3201).
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late List<Uri> requests;

  setUp(() {
    requests = [];
    resetSharedHttpClient();
    sharedHttpClientFactory = () => MockClient((request) async {
      requests.add(request.url);
      return http.Response('[]', 200);
    });
  });

  tearDown(() {
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  final calls = <String, Future<void> Function(List<String>?)>{
    'getFiles': (s) => FilesService.getFiles('', serials: s),
    'searchFiles': (s) => FilesService.searchFiles('a', serials: s),
    'getFilesByType': (s) => FilesService.getFilesByType('qdoc', serials: s),
    'getRecentFiles': (s) => FilesService.getRecentFiles(serials: s),
  };

  for (final MapEntry(key: name, value: call) in calls.entries) {
    group(name, () {
      test('sends the internal drive as a blank serial', () async {
        await call(['', 'USB1']);
        expect(requests.single.queryParametersAll['serial'], ['', 'USB1']);
      });

      test('built-in storage alone is still a filter', () async {
        await call(['']);
        expect(requests.single.queryParametersAll['serial'], ['']);
      });

      test('no selection sends no filter', () async {
        await call(null);
        await call(const []);
        for (final uri in requests) {
          expect(uri.queryParametersAll.containsKey('serial'), isFalse);
        }
      });
    });
  }
}
