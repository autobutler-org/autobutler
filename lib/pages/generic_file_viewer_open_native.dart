import 'dart:io' show File;
import 'dart:typed_data';

import 'package:open_filex/open_filex.dart';
import 'package:path_provider/path_provider.dart';

Future<String> openFileWithSystem(Uint8List bytes, String fileName) async {
  final tempDir = await getTemporaryDirectory();
  final tempFile = File('${tempDir.path}/$fileName');
  await tempFile.writeAsBytes(bytes);
  final result = await OpenFilex.open(tempFile.path);
  if (result.type != ResultType.done) {
    return result.message;
  }
  return '';
}
