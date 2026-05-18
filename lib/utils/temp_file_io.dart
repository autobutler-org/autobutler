import 'dart:io';
import 'dart:typed_data';

/// Writes [bytes] to a temp file named [fileName] and returns its path.
Future<String?> writeTempFile(Uint8List bytes, String fileName) async {
  final tmpPath = '${Directory.systemTemp.path}/$fileName';
  await File(tmpPath).writeAsBytes(bytes);
  return tmpPath;
}
