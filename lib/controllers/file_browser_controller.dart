import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:quark/models/file_node.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/services/file_browser_actions.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/file_browser_dialog_utils.dart';
import 'package:quark/utils/folder_picker.dart';
import 'package:quark/utils/file_browser_path_utils.dart';
import 'package:quark/utils/quark_widget.dart';
import 'package:quark/utils/upload_tree_utils.dart';
import 'package:quark/widgets/file_browser/file_browser_view.dart';

class FileMenuActionOutcome {
  const FileMenuActionOutcome({
    required this.message,
    this.shouldRefresh = false,
  });

  final String message;
  final bool shouldRefresh;
}

class FileBrowserController {
  const FileBrowserController();

  Future<List<FileNode>> fetchFiles(
    String currentPath, {
    List<String>? serials,
  }) {
    return FilesService.getFiles(currentPath, serials: serials);
  }

  /// Picks one or more files for upload.
  ///
  /// On web and desktop, multiple file selection is supported.
  /// On mobile (iOS/Android), the platform picker typically supports
  /// multi-select — enabled via [allowMultiple: true]. If the platform
  /// returns only a single file, the list will have one entry.
  ///
  /// Selects files only. For folders see [pickUploadFolder], which needs a
  /// different picker on every platform and so cannot share this entry point.
  ///
  /// Returns an empty list if the user cancelled.
  Future<List<http.MultipartFile>> pickUploadFiles() async {
    final result = await FilePicker.pickFiles(
      withData: true,
      allowMultiple: true,
    );
    if (result == null || result.files.isEmpty) {
      return [];
    }

    final files = <http.MultipartFile>[];
    for (final platformFile in result.files) {
      final f = await multipartFileFromPlatformFile(platformFile);
      if (f != null) files.add(f);
    }
    return files;
  }

  /// Whether this platform can offer folder selection at all.
  ///
  /// Web and desktop can; mobile has no meaningful folder picker, so callers
  /// hide the affordance rather than offering one that cannot work.
  bool get isFolderUploadSupported => isFolderPickerSupported;

  /// Picks a folder and returns its files, each carrying the directory it sat
  /// in relative to the chosen folder.
  ///
  /// Returns an empty list if the user cancelled or the folder holds no files.
  Future<List<PendingUpload>> pickUploadFolder() {
    return pickFolderUploads();
  }

  /// @deprecated Use [pickUploadFiles] instead.
  Future<http.MultipartFile?> pickUploadFile() async {
    final files = await pickUploadFiles();
    return files.isEmpty ? null : files.first;
  }

  Future<http.MultipartFile?> multipartFileFromPlatformFile(
    PlatformFile selected,
  ) async {
    final bytes = selected.bytes;

    // Web cannot use MultipartFile.fromPath because it depends on dart:io.
    if (kIsWeb) {
      if (bytes == null) {
        return null;
      }
      return multipartFileFromBytes(bytes: bytes, filename: selected.name);
    }

    final path = selected.path;
    if (path != null && path.isNotEmpty) {
      return http.MultipartFile.fromPath(
        'files',
        path,
        filename: selected.name,
      );
    }

    if (bytes == null) {
      return null;
    }
    return multipartFileFromBytes(bytes: bytes, filename: selected.name);
  }

  http.MultipartFile multipartFileFromBytes({
    required Uint8List bytes,
    required String filename,
  }) {
    return http.MultipartFile.fromBytes('files', bytes, filename: filename);
  }

  Future<void> uploadFile({
    required String currentPath,
    required http.MultipartFile selectedFile,
    String? serial,
  }) {
    return uploadFiles(
      currentPath: currentPath,
      selectedFiles: [selectedFile],
      serial: serial,
    );
  }

  Future<void> uploadFiles({
    required String currentPath,
    required List<http.MultipartFile> selectedFiles,
    String? serial,
  }) {
    return uploadMultipartFilesToCurrentPath(
      currentPath: currentPath,
      selectedFiles: selectedFiles,
      serial: serial,
    );
  }

  Future<String?> promptFolderName(BuildContext context) {
    return promptForFolderName(context);
  }

  Future<void> createFolder({
    required String currentPath,
    required String folderName,
  }) {
    return createFolderAtCurrentPath(
      currentPath: currentPath,
      folderName: folderName,
    );
  }

  /// Delete a single node. Caller is responsible for confirmation and
  /// any optimistic UI updates.
  Future<void> deleteNode({required FileNode node}) {
    final rootDir = toRootDir(parentPath(node.apiPath));
    return FilesService.deleteFile(
      rootDir,
      trimTrailingSlashes(node.name),
      deviceSerial: serialOrNull(node.deviceSerial),
    );
  }

  /// Deletes [nodes] in a single batch request per device group.
  Future<void> deleteNodes({required List<FileNode> nodes}) async {
    if (nodes.isEmpty) return;
    // Group by device serial so each batch request stays on one device.
    final bySerial = <String, List<FileNode>>{};
    for (final n in nodes) {
      final key = serialOrNull(n.deviceSerial) ?? '';
      (bySerial[key] ??= []).add(n);
    }
    for (final entry in bySerial.entries) {
      final serial = entry.key.isEmpty ? null : entry.key;
      final paths = entry.value
          .map((n) => trimTrailingSlashes(n.name))
          .toList();
      // Use the parent dir of the first node as rootDir — they should all
      // share the same directory within a batch, but the backend ignores
      // rootDir when explicit filePaths are supplied.
      final rootDir = toRootDir(parentPath(entry.value.first.apiPath));
      await FilesService.deleteFiles(
        paths,
        rootDir: rootDir,
        deviceSerial: serial,
      );
    }
  }

  Future<FileMenuActionOutcome?> handleFileAction({
    required FileNode node,
    required FileMenuAction action,
    required BuildContext context,
  }) async {
    switch (action) {
      case FileMenuAction.download:
        final savedPath = await downloadNode(node: node);
        if (savedPath == null) {
          return const FileMenuActionOutcome(message: 'Download canceled');
        }
        return FileMenuActionOutcome(message: downloadedMessage(node));
      case FileMenuAction.moveRename:
        final startPath = parentPath(node.apiPath);
        // Fetch devices for cross-device move support.
        // Use ALL devices (not just isEnabled) so the picker shows even
        // when mount state is stale — the user can still select a drive
        // that they know is mounted.
        List<StorageDevice> allDevices = [];
        try {
          allDevices = await StorageService.listDevices();
        } catch (_) {
          // Fall through with empty list — dialog will skip device picker
        }
        if (!context.mounted) return null;
        final moveResult = await promptForMoveRenamePath(
          context,
          startPath: startPath,
          initialName: node.name,
          devices: allDevices,
        );
        if (!context.mounted) {
          return null;
        }
        if (moveResult == null) {
          return null;
        }
        final targetInput = moveResult.targetInput;
        final targetPath = resolveMoveRenameTargetPath(
          currentPath: startPath,
          nodeApiPath: node.apiPath,
          targetInput: targetInput,
        );
        if (targetPath == null) {
          return null;
        }

        // Prevent moving a directory into itself or its own subtree
        if (node.isDir) {
          final normalizedOld = normalizePath(node.apiPath);
          final normalizedTarget = normalizePath(targetPath);
          if (normalizedTarget == normalizedOld ||
              normalizedTarget.startsWith('$normalizedOld/')) {
            // show an error dialog
            await QuarkWidget.showDialog<void>(
              context,
              useRootNavigator: true,
              builder: (dialogContext) => QuarkWidget.alertDialog(
                title: const Text('Invalid target'),
                content: const Text(
                  'Cannot move a folder into itself or one of its subfolders.',
                ),
                actions: [
                  TextButton(
                    onPressed: () => Navigator.of(dialogContext).pop(),
                    child: const Text('OK'),
                  ),
                ],
              ),
            );
            return null;
          }
        }

        await moveRenameNode(
          node: node,
          targetInput: targetInput,
          newDeviceSerial: moveResult.deviceSerial,
        );
        return const FileMenuActionOutcome(
          message: 'Move/Rename complete',
          shouldRefresh: true,
        );
      case FileMenuAction.delete:
        final shouldDelete = await confirmDelete(
          context,
          trimTrailingSlashes(node.name),
        );
        if (shouldDelete != true) {
          return null;
        }
        await deleteNode(node: node);
        return const FileMenuActionOutcome(
          message: 'Deleted',
          shouldRefresh: true,
        );
      case FileMenuAction.extractHere:
        await extractNode(node: node);
        return const FileMenuActionOutcome(
          message: 'Extraction complete',
          shouldRefresh: true,
        );
      case FileMenuAction.navigateToFolder:
        // Handled via the onNavigateToFolder callback in FileBrowserView;
        // should never reach handleFileAction.
        return null;
    }
  }

  String failureMessage(FileMenuAction action) {
    switch (action) {
      case FileMenuAction.download:
        return 'Download failed';
      case FileMenuAction.moveRename:
        return 'Move/Rename failed';
      case FileMenuAction.delete:
        return 'Delete failed';
      case FileMenuAction.extractHere:
        return 'Extraction failed';
      case FileMenuAction.navigateToFolder:
        return 'Navigation failed';
    }
  }

  String? resolveMoveRenameTargetPath({
    required String currentPath,
    required String nodeApiPath,
    required String targetInput,
  }) {
    final oldPath = normalizePath(nodeApiPath);
    final targetPath = targetInput.startsWith('/')
        ? normalizePath(targetInput)
        : joinPath(currentPath, targetInput);

    if (targetPath.isEmpty || targetPath == oldPath) {
      return null;
    }

    return targetPath;
  }

  String nextPathForOpenDirectory({
    required String currentPath,
    required FileNode node,
  }) {
    return joinPath(currentPath, node.name);
  }

  String nextPathForGoUp(String currentPath) {
    return parentPath(currentPath);
  }

  String downloadedMessage(FileNode node) {
    return 'Downloaded ${trimTrailingSlashes(node.name)}';
  }
}
