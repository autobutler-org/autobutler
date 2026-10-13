import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/upload_destination.dart';

/// #3191: a new folder names the device it goes to, and the device list an
/// upload starts from is not the cached one.
void main() {
  late List<Uri> requests;

  setUp(() {
    requests = [];
    StorageService.invalidateDeviceCache();
    sharedHttpClientFactory = () => MockClient((request) async {
      requests.add(request.url);
      return http.Response('{"devices":[]}', 200);
    });
  });

  tearDown(() {
    StorageService.invalidateDeviceCache();
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  test('a new folder on a USB drive sends its serial', () async {
    await FilesService.createFolder('Docs', 'New', serial: 'USB 1');

    expect(requests.single.path, '/api/v0/files/folder/Docs');
    expect(requests.single.queryParameters, {'serial': 'USB 1'});
  });

  test('a new folder on built-in storage sends no serial', () async {
    await FilesService.createFolder('', 'New');

    expect(requests.single.path, '/api/v0/files/folder/');
    expect(requests.single.hasQuery, isFalse);
  });

  test('listDevicesFresh asks the Quark past the cache', () async {
    await StorageService.listDevices();
    await StorageService.listDevices();
    expect(requests, hasLength(1), reason: 'the second call is cached');

    await listDevicesFresh();
    expect(requests, hasLength(2));
  });
}
