import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:quark/utils/upload_tree_utils.dart';

/// Desktop can open a directory chooser; mobile cannot.
///
/// [FilePicker.getDirectoryPath] is implemented for Linux, macOS and Windows.
/// On Android it returns protected or unusable paths and on iOS there is no
/// equivalent at all, so folder upload is not offered there.
bool get isFolderPickerSupportedPlatform =>
    Platform.isLinux || Platform.isMacOS || Platform.isWindows;

Future<List<PendingUpload>> pickFolderUploadsPlatform() async {
  if (!isFolderPickerSupportedPlatform) {
    return const [];
  }

  final rootPath = await FilePicker.getDirectoryPath(
    dialogTitle: 'Select folder to upload',
  );
  if (rootPath == null || rootPath.trim().isEmpty) {
    return const [];
  }

  final root = Directory(rootPath);
  if (!root.existsSync()) {
    return const [];
  }

  final uploads = <PendingUpload>[];
  // followLinks: false — a symlink out of the chosen folder is exactly the
  // traversal this feature must not perform, and sanitizeRelativeDir cannot
  // see it because the resulting relative path looks ordinary.
  await for (final entity in root.list(recursive: true, followLinks: false)) {
    if (entity is! File) {
      continue;
    }
    if (uploads.length >= kMaxUploadFiles) {
      break;
    }

    final relativePath = _relativeTo(root.path, entity.path);
    final relativeDir = sanitizeRelativeDir(relativeDirOf(relativePath));
    if (relativeDir == null) {
      continue;
    }

    final name = _basename(relativePath).trim();
    if (name.isEmpty) {
      continue;
    }

    uploads.add(
      PendingUpload(
        relativeDir: relativeDir,
        name: name,
        // fromPath streams off disk, so a large folder never lands in memory
        // all at once.
        build: () async {
          try {
            return await http.MultipartFile.fromPath(
              'files',
              entity.path,
              filename: name,
            );
          } catch (e) {
            debugPrint('[folder_picker_io.dart] Failed to read $name: $e');
            return null;
          }
        },
      ),
    );
  }

  return uploads;
}

/// [filePath] expressed relative to [rootPath], with `/` separators.
///
/// Directory.list yields absolute paths under the directory it was given, so
/// stripping that prefix is enough — no path package needed for a job this
/// narrow.
String _relativeTo(String rootPath, String filePath) {
  var relative = filePath;
  if (relative.startsWith(rootPath)) {
    relative = relative.substring(rootPath.length);
  }
  return relative.replaceAll(r'\', '/').replaceAll(RegExp(r'^/+'), '');
}

String _basename(String relativePath) {
  final lastSlash = relativePath.lastIndexOf('/');
  return lastSlash < 0 ? relativePath : relativePath.substring(lastSlash + 1);
}
