import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/download_destination.dart';
import 'package:quark/services/file_save_io.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/utils/error_text.dart';

// The desktop save (#3150): the destination is asked for before anything is
// downloaded, and the download is then moved there, never held in memory.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late Directory dir;
  late int downloads;

  setUp(() {
    dir = Directory.systemTemp.createTempSync('file_save_io_test');
    downloads = 0;
    resetSharedHttpClient();
    sharedHttpClientFactory = () => MockClient((request) async {
      downloads++;
      return http.Response('report body', 200);
    });
  });
  tearDown(() {
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
    dir.deleteSync(recursive: true);
  });

  Future<DownloadedFile> download() => FilesService.instance
      .authenticatedDownload(Uri.parse('http://quark.test/report.pdf'));

  test('canceling downloads nothing and writes nothing', () async {
    String? asked;
    final saved = await saveDownloadToPickedPath(
      fileName: 'report.pdf',
      download: download,
      pickPath: (name) async {
        asked = name;
        return null;
      },
    );

    expect(saved, isNull);
    expect(asked, 'report.pdf');
    expect(downloads, 0, reason: 'the dialog is answered before downloading');
    expect(dir.listSync(), isEmpty);
  });

  test('streams the download to the chosen path', () async {
    final destination = '${dir.path}/chosen name.pdf';
    late String tempPath;
    final saved = await saveDownloadToPickedPath(
      fileName: 'report.pdf',
      download: () async {
        final downloaded = await download();
        tempPath = downloaded.path;
        return downloaded;
      },
      pickPath: (_) async => destination,
    );

    expect(saved, destination);
    expect(await File(destination).readAsString(), 'report body');
    expect(File(tempPath).existsSync(), isFalse);
  });

  test('replaces the placeholder the picker left at the path', () async {
    final destination = '${dir.path}/report.pdf';
    File(destination).writeAsStringSync('');

    await saveDownloadToPickedPath(
      fileName: 'report.pdf',
      download: download,
      pickPath: (_) async => destination,
    );

    expect(await File(destination).readAsString(), 'report body');
  });

  test('a failed download leaves no empty file behind', () async {
    final destination = '${dir.path}/report.pdf';

    await expectLater(
      saveDownloadToPickedPath(
        fileName: 'report.pdf',
        download: () async {
          File(destination).writeAsStringSync('');
          throw const HttpException('connection reset');
        },
        pickPath: (_) async => destination,
      ),
      throwsA(isA<HttpException>()),
    );
    expect(File(destination).existsSync(), isFalse);
  });

  test('bytes land at the chosen path, and nowhere when canceled', () async {
    final destination = '${dir.path}/notes.txt';
    final bytes = Uint8List.fromList('hello'.codeUnits);

    expect(
      await saveBytesToPickedPath(
        bytes,
        fileName: 'notes.txt',
        pickPath: (_) async => null,
      ),
      isNull,
    );
    expect(dir.listSync(), isEmpty);

    expect(
      await saveBytesToPickedPath(
        bytes,
        fileName: 'notes.txt',
        pickPath: (_) async => destination,
      ),
      destination,
    );
    expect(await File(destination).readAsString(), 'hello');
  });

  // The mobile save (#3334): the photo library for pictures and videos, the
  // save dialog for the rest and for anything the library cannot take.
  group('saveToPhotoLibraryOrFiles', () {
    late List<String> calls;
    late List<String> notices;

    setUp(() {
      calls = [];
      notices = [];
    });

    Future<String?> save(
      DownloadDestination destination, {
      bool allowed = true,
      bool libraryRefuses = false,
      String? filesResult = '/files/saved',
    }) => saveToPhotoLibraryOrFiles(
      destination: destination,
      requestAccess: () async {
        calls.add('ask');
        return allowed;
      },
      saveToLibrary: (destination) async {
        calls.add('library:${destination.name}');
        if (libraryRefuses) throw Exception('unsupported format');
        return 'asset-1';
      },
      saveToFiles: () async {
        calls.add('files');
        return filesResult;
      },
      notify: notices.add,
    );

    test('a picture goes to the library', () async {
      expect(await save(DownloadDestination.photoLibraryImage), 'asset-1');
      expect(calls, ['ask', 'library:photoLibraryImage']);
      expect(notices, isEmpty);
    });

    test('a video goes to the library', () async {
      expect(await save(DownloadDestination.photoLibraryVideo), 'asset-1');
      expect(calls, ['ask', 'library:photoLibraryVideo']);
    });

    test(
      'another file goes to Files without asking for photo access',
      () async {
        expect(await save(DownloadDestination.files), '/files/saved');
        expect(calls, ['files']);
        expect(notices, isEmpty);
      },
    );

    test('declined access still saves to Files and says why', () async {
      expect(
        await save(DownloadDestination.photoLibraryImage, allowed: false),
        '/files/saved',
      );
      expect(calls, ['ask', 'files']);
      expect(notices, [Errors.photoLibraryAccessDenied]);
    });

    test('declined access then a canceled dialog says nothing', () async {
      expect(
        await save(
          DownloadDestination.photoLibraryImage,
          allowed: false,
          filesResult: null,
        ),
        isNull,
      );
      expect(notices, isEmpty);
    });

    test('a file the library refuses goes to Files', () async {
      expect(
        await save(DownloadDestination.photoLibraryImage, libraryRefuses: true),
        '/files/saved',
      );
      expect(calls, ['ask', 'library:photoLibraryImage', 'files']);
      expect(notices, isEmpty);
    });
  });
}
