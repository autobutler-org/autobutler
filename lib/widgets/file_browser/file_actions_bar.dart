import 'package:flutter/material.dart';

class FileActionsBar extends StatelessWidget {
  const FileActionsBar({
    required this.isUploading,
    required this.isCreatingFolder,
    required this.onUploadPressed,
    required this.onCreateFolderPressed,
    required this.isSearchMode,
    this.uploadTotal = 0,
    this.uploadCompleted = 0,
    super.key,
  });

  final bool isUploading;
  final bool isCreatingFolder;
  final VoidCallback onUploadPressed;
  final VoidCallback onCreateFolderPressed;
  final bool isSearchMode;
  /// Total number of files in the current upload batch. 0 when not uploading.
  final int uploadTotal;
  /// Number of files completed so far in the current batch.
  final int uploadCompleted;

  String get _uploadLabel {
    if (!isUploading) return 'Upload';
    if (uploadTotal > 1) return 'Uploading $uploadCompleted of $uploadTotal...';
    return 'Uploading...';
  }

  @override
  Widget build(BuildContext context) {
    if (isSearchMode) {
      return const SizedBox.shrink();
    }
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
      child: Row(
        children: [
          FilledButton.tonalIcon(
            onPressed: isUploading ? null : onUploadPressed,
            icon: isUploading
                ? SizedBox.square(
                    dimension: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      value: uploadTotal > 1
                          ? uploadCompleted / uploadTotal
                          : null,
                    ),
                  )
                : const Icon(Icons.upload_rounded),
            label: Text(_uploadLabel),
          ),
          const SizedBox(width: 8),
          OutlinedButton.icon(
            onPressed: isCreatingFolder ? null : onCreateFolderPressed,
            icon: const Icon(Icons.create_new_folder_outlined),
            label: Text(isCreatingFolder ? 'Creating...' : 'New Folder'),
          ),
        ],
      ),
    );
  }
}
