import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_file_dialog/flutter_file_dialog.dart';
import 'package:quark/services/authenticated_service.dart';

/// Asks the user where to save a file named [fileName], and returns the path
/// they chose, or null when they canceled.
typedef SavePathPicker = Future<String?> Function(String fileName);

/// `flutter_file_dialog` implements only Android and iOS; every other native
/// platform saves through `file_picker` instead (#3150).
bool get _usesMobileDialog => Platform.isAndroid || Platform.isIOS;

/// The native save of a download: the mobile dialog on Android and iOS, the
/// desktop one everywhere else. See `saveDownloadThroughDialog`.
Future<String?> saveDownloadPlatform({
  required String Function(Map<String, String> headers) fileNameFor,
  required Future<DownloadedFile> Function() download,
}) async {
  if (!_usesMobileDialog) {
    return saveDownloadToPickedPath(
      fileName: fileNameFor(const {}),
      download: download,
      pickPath: _pickDesktopSavePath,
    );
  }
  final downloaded = await download();
  try {
    // Awaited, not just returned: the finally below deletes the temp file, and
    // an unawaited future would let that race the dialog reading it.
    return await FlutterFileDialog.saveFile(
      params: SaveFileDialogParams(
        sourceFilePath: downloaded.path,
        fileName: fileNameFor(downloaded.headers),
      ),
    );
  } finally {
    await downloaded.delete();
  }
}

/// The native save of [bytes]: the mobile dialog on Android and iOS, the
/// desktop one everywhere else.
Future<String?> saveBytesPlatform(Uint8List bytes, String fileName) {
  if (!_usesMobileDialog) {
    return saveBytesToPickedPath(
      bytes,
      fileName: fileName,
      pickPath: _pickDesktopSavePath,
    );
  }
  return FlutterFileDialog.saveFile(
    params: SaveFileDialogParams(data: bytes, fileName: fileName),
  );
}

/// The desktop save: asks [pickPath] for a destination first, and only then
/// streams [download] and moves it there. Canceling the dialog downloads and
/// writes nothing. Returns the destination, or null when canceled.
@visibleForTesting
Future<String?> saveDownloadToPickedPath({
  required String fileName,
  required Future<DownloadedFile> Function() download,
  required SavePathPicker pickPath,
}) async {
  final destination = await pickPath(fileName);
  if (destination == null) return null;
  try {
    final downloaded = await download();
    try {
      final source = File(downloaded.path);
      try {
        await source.rename(destination);
      } on FileSystemException {
        // A rename cannot cross filesystems, and the temp directory is often
        // on its own. A copy streams, so it costs no more memory than a move.
        await source.copy(destination);
      }
    } finally {
      await downloaded.delete();
    }
  } catch (_) {
    await _removePlaceholder(destination);
    rethrow;
  }
  return destination;
}

/// The desktop save of [bytes]: asks [pickPath] for a destination and writes
/// them there. Returns the destination, or null when canceled.
@visibleForTesting
Future<String?> saveBytesToPickedPath(
  Uint8List bytes, {
  required String fileName,
  required SavePathPicker pickPath,
}) async {
  final destination = await pickPath(fileName);
  if (destination == null) return null;
  await File(destination).writeAsBytes(bytes, flush: true);
  return destination;
}

/// The Linux picker writes the bytes it is handed to the chosen path before
/// returning. It is handed none, so a failed download would otherwise leave an
/// empty file behind where the user expected theirs.
Future<void> _removePlaceholder(String path) async {
  try {
    final file = File(path);
    if (await file.length() == 0) await file.delete();
  } catch (_) {
    // Nothing there, or nothing we may touch: either way, nothing to clean.
  }
}

/// Opens `file_picker`'s save dialog. On Linux that is the XDG desktop portal's
/// FileChooser over D-Bus, so the session needs `xdg-desktop-portal` and a
/// backend for it (GTK, KDE, GNOME, ...); there is no command-line dialog
/// fallback.
///
/// `file_picker` demands bytes to write, but the download has not happened
/// yet: it gets none, and the caller streams the file over what it wrote.
Future<String?> _pickDesktopSavePath(String fileName) async {
  final uri = await FilePicker.saveFile(
    dialogTitle: 'Save file',
    fileName: fileName,
    bytes: Uint8List(0),
  );
  return uri?.toFilePath();
}
