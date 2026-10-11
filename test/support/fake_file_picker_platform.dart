import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';

/// Stands in for `file_picker`'s save dialog, which a desktop save opens
/// (#3150): the user picks a file under [dir] named as offered, and the fake
/// writes the bytes it was handed there, as the Linux portal backend does.
///
/// Install it as `FilePickerPlatform.instance` in `setUp`.
class FakeFilePickerPlatform extends FilePickerPlatform {
  FakeFilePickerPlatform(this.dir);

  /// Where every save lands.
  final Directory dir;

  /// The name each save dialog offered, in order.
  final offered = <String>[];

  @override
  Future<Uri?> saveFile({
    required String fileName,
    required Uint8List bytes,
    required String mimeType,
    String? dialogTitle,
    String? initialDirectory,
    Function(FilePickerStatus)? onFileSaving,
    WindowsOptions windowsOptions = const WindowsOptions(),
    LinuxOptions linuxOptions = const LinuxOptions(),
    WebOptions webOptions = const WebOptions(),
  }) async {
    offered.add(fileName);
    final file = File('${dir.path}/$fileName');
    await file.writeAsBytes(bytes);
    return Uri.file(file.path);
  }

  /// The name and bytes of the first save, once a non-empty file has landed.
  ///
  /// The picker answers before the download starts, so the bytes arrive
  /// after it returns; this waits for them.
  Future<(String, List<int>)> saved() async {
    while (true) {
      if (offered.isNotEmpty) {
        final file = File('${dir.path}/${offered.first}');
        if (file.existsSync() && file.lengthSync() > 0) {
          return (offered.first, file.readAsBytesSync());
        }
      }
      await Future<void>.delayed(const Duration(milliseconds: 10));
    }
  }
}
