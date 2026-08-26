import 'dart:async';
import 'dart:js_interop';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:quark/utils/upload_tree_utils.dart';
import 'package:web/web.dart' as web;

/// Browsers have supported directory selection for over a decade.
///
/// `webkitdirectory` is non-standard in name only — Chrome, Firefox, Safari
/// and Edge all implement it, and each selected file exposes
/// `webkitRelativePath` giving its path inside the chosen folder. That is
/// exactly the structure needed to rebuild `rootDir`, and it is why this goes
/// straight to the DOM instead of through file_picker, whose PlatformFile
/// does not surface the property at all.
bool get isFolderPickerSupportedPlatform => true;

Future<List<PendingUpload>> pickFolderUploadsPlatform() async {
  final input = web.HTMLInputElement()
    ..type = 'file'
    ..multiple = true
    ..style.display = 'none';
  input.setAttribute('webkitdirectory', 'true');
  web.document.body?.append(input);

  final completer = Completer<List<PendingUpload>>();
  void finish(List<PendingUpload> uploads) {
    if (!completer.isCompleted) {
      completer.complete(uploads);
    }
  }

  input.onchange = ((web.Event _) {
    finish(_uploadsFromInput(input));
  }).toJS;
  // Fired when the picker is dismissed without choosing anything. Browsers
  // that predate the cancel event simply leave the future pending until the
  // user picks something, which is the same as today's file picker.
  input.oncancel = ((web.Event _) {
    finish(const []);
  }).toJS;

  input.click();

  try {
    return await completer.future;
  } finally {
    input.remove();
  }
}

List<PendingUpload> _uploadsFromInput(web.HTMLInputElement input) {
  final files = input.files;
  if (files == null || files.length == 0) {
    return const [];
  }

  final uploads = <PendingUpload>[];
  for (var i = 0; i < files.length; i++) {
    if (uploads.length >= kMaxUploadFiles) {
      break;
    }
    final file = files.item(i);
    if (file == null) {
      continue;
    }

    // webkitRelativePath is 'chosenFolder/sub/file.txt'. It is empty when the
    // browser gave us a plain file selection instead of a directory one, in
    // which case the file belongs at the upload root.
    final relativePath = file.webkitRelativePath.isNotEmpty
        ? file.webkitRelativePath
        : file.name;
    final relativeDir = sanitizeRelativeDir(relativeDirOf(relativePath));
    if (relativeDir == null) {
      continue;
    }

    final name = file.name.trim();
    if (name.isEmpty) {
      continue;
    }

    uploads.add(
      PendingUpload(
        relativeDir: relativeDir,
        name: name,
        build: () async {
          try {
            final buffer = await file.arrayBuffer().toDart;
            return http.MultipartFile.fromBytes(
              'files',
              buffer.toDart.asUint8List(),
              filename: name,
            );
          } catch (e) {
            debugPrint('[folder_picker_web.dart] Failed to read $name: $e');
            return null;
          }
        },
      ),
    );
  }

  return uploads;
}
