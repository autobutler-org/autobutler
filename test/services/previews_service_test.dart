import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The previews API as the document viewer calls it (#2987).
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final requests = <http.Request>[];

  void answer(int status, List<int> body) {
    resetSharedHttpClient();
    sharedHttpClientFactory = () => MockClient((request) async {
      requests.add(request);
      return http.Response.bytes(body, status);
    });
  }

  setUp(requests.clear);
  tearDown(() {
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  test('reads the block stream as UTF-8 though it names no charset', () async {
    answer(
      200,
      utf8.encode(
        '{"type":"meta","version":1,"format":"odt"}\n'
        '{"type":"heading","level":1,"runs":[{"text":"Café ß"}]}\n'
        '{"type":"end","truncated":false,"words":2,"characters":6,'
        '"paragraphs":1}\n',
      ),
    );

    final document = await PreviewsService.document(
      'work/menu.odt',
      serial: 'usb1',
    );

    final request = requests.single;
    expect(request.method, 'GET');
    expect(request.url.path, '/api/v0/previews/document');
    expect(request.url.queryParameters, {
      'filePath': 'work/menu.odt',
      'serial': 'usb1',
    });
    final heading = document.blocks.single as QuarkDocumentHeading;
    expect(heading.runs.single.text, 'Café ß');
  });

  test('a stream cut off before its end line is refused', () async {
    answer(200, utf8.encode('{"type":"meta","version":1,"format":"rtf"}\n'));
    await expectLater(
      PreviewsService.document('a.rtf'),
      throwsA(isA<FormatException>()),
    );
  });

  test('a refused preview throws its status', () async {
    answer(400, utf8.encode('{"error":"not a document"}'));
    await expectLater(
      PreviewsService.document('fake.docx'),
      throwsA(isA<ApiException>().having((e) => e.statusCode, 'status', 400)),
    );
    expect(requests.single.url.queryParameters.containsKey('serial'), isFalse);
  });

  test('the part address names the document, the part and the device', () {
    final url = PreviewsService.partUrl(
      'work/q3.docx',
      'word/media/image1.png',
      serial: 'usb1',
    );
    expect(url.path, '/api/v0/previews/part');
    expect(url.queryParameters, {
      'filePath': 'work/q3.docx',
      'part': 'word/media/image1.png',
      'serial': 'usb1',
    });
  });

  test('Open as Quark doc posts and returns the path and warnings', () async {
    answer(
      200,
      utf8.encode(
        jsonEncode({
          'path': 'work/q3.qdoc',
          'truncated': false,
          'words': 10,
          'warnings': ['2 images left out.'],
        }),
      ),
    );

    final converted = await PreviewsService.convertToQdoc(
      'work/q3.docx',
      overwrite: true,
    );

    final request = requests.single;
    expect(request.method, 'POST');
    expect(request.url.path, '/api/v0/files/convert/document');
    expect(request.url.queryParameters, {
      'filePath': 'work/q3.docx',
      'overwrite': 'true',
    });
    expect(converted.path, 'work/q3.qdoc');
    expect(converted.warnings, ['2 images left out.']);
  });

  test('an existing .qdoc comes back as a 409', () async {
    answer(409, utf8.encode('{"error":"exists"}'));
    await expectLater(
      PreviewsService.convertToQdoc('work/q3.docx'),
      throwsA(isA<ApiException>().having((e) => e.statusCode, 'status', 409)),
    );
    expect(
      requests.single.url.queryParameters.containsKey('overwrite'),
      isFalse,
    );
  });
}
