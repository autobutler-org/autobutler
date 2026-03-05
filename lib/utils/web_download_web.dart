import 'dart:convert';
import 'dart:typed_data';

import 'package:web/web.dart' as web;

Future<String?> saveBytesForDownload(Uint8List data, String fileName) async {
  final encoded = base64Encode(data);
  final anchor = web.HTMLAnchorElement()
    ..href = 'data:application/octet-stream;base64,$encoded'
    ..download = fileName
    ..style.display = 'none';

  web.document.body?.append(anchor);
  anchor.click();
  anchor.remove();

  return fileName;
}
