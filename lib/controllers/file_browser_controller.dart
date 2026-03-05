import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/services/file_browser_actions.dart';
import 'package:autobutler/utils/autobutler_widget.dart';
import 'package:autobutler/utils/file_browser_dialog_utils.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

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

  Future<List<CirrusFileNode>> fetchFiles(String currentPath) {
    return CirrusService.getFiles(currentPath);
  }

  Future<http.MultipartFile?> pickUploadFile() async {
    final result = await FilePicker.platform.pickFiles(withData: true);
    if (result == null || result.files.isEmpty) {
      return null;
    }

    final selected = result.files.first;
    final bytes = selected.bytes;

    // Web cannot use MultipartFile.fromPath because it depends on dart:io.
    if (kIsWeb) {
      if (bytes == null) {
        return null;
      }
      return http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: selected.name,
      );
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
    return http.MultipartFile.fromBytes(
      'files',
      bytes,
      filename: selected.name,
    );
  }

  Future<void> uploadFile({
    required String currentPath,
    required http.MultipartFile selectedFile,
  }) {
    return uploadMultipartFileToCurrentPath(
      currentPath: currentPath,
      selectedFile: selectedFile,
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

  Future<FileMenuActionOutcome?> handleFileAction({
    required String currentPath,
    required CirrusFileNode node,
    required FileMenuAction action,
    required BuildContext context,
  }) async {
    switch (action) {
      case FileMenuAction.download:
        final savedPath = await downloadNode(
          currentPath: currentPath,
          node: node,
        );
        if (savedPath == null) {
          return const FileMenuActionOutcome(message: 'Download canceled');
        }
        return FileMenuActionOutcome(message: downloadedMessage(node));
      case FileMenuAction.moveRename:
        final targetInput = await promptForMoveRenamePath(
          context,
          startPath: currentPath,
          initialName: node.name,
        );
        if (!context.mounted) {
          return null;
        }
        if (targetInput == null) {
          return null;
        }
        final targetPath = resolveMoveRenameTargetPath(
          currentPath: currentPath,
          nodeName: node.name,
          targetInput: targetInput,
        );
        if (targetPath == null) {
          return null;
        }

        // Prevent moving a directory into itself or its own subtree
        if (node.isDir) {
          final oldPath = joinPath(currentPath, trimTrailingSlashes(node.name));
          final normalizedOld = normalizePath(oldPath);
          final normalizedTarget = normalizePath(targetPath);
          if (normalizedTarget == normalizedOld ||
              normalizedTarget.startsWith('$normalizedOld/')) {
            // show an error dialog
            await AutobutlerWidget.showDialog<void>(
              context,
              useRootNavigator: true,
              builder: (dialogContext) => AutobutlerWidget.alertDialog(
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
          currentPath: currentPath,
          node: node,
          targetInput: targetInput,
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
        await deleteNode(currentPath: currentPath, node: node);
        return const FileMenuActionOutcome(
          message: 'Deleted',
          shouldRefresh: true,
        );
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
    }
  }

  String? resolveMoveRenameTargetPath({
    required String currentPath,
    required String nodeName,
    required String targetInput,
  }) {
    final oldPath = joinPath(currentPath, trimTrailingSlashes(nodeName));
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
    required CirrusFileNode node,
  }) {
    return joinPath(currentPath, node.name);
  }

  String nextPathForGoUp(String currentPath) {
    return parentPath(currentPath);
  }

  String downloadedMessage(CirrusFileNode node) {
    return 'Downloaded ${trimTrailingSlashes(node.name)}';
  }
}
