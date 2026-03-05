import 'dart:html' as html;
import 'dart:typed_data';

Future<String?> saveBytesForDownload(Uint8List data, String fileName) async {
  final blob = html.Blob(<dynamic>[data]);
  final url = html.Url.createObjectUrlFromBlob(blob);
  final anchor = html.AnchorElement(href: url)
    ..download = fileName
    ..style.display = 'none';

  html.document.body?.append(anchor);
  anchor.click();
  anchor.remove();
  html.Url.revokeObjectUrl(url);

  return fileName;
}
