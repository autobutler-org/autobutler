import 'package:flutter/material.dart';

class FileActionsBar extends StatelessWidget {
  const FileActionsBar({
    required this.isUploading,
    required this.isCreatingFolder,
    required this.onUploadPressed,
    required this.onCreateFolderPressed,
    required this.onRefreshPressed,
    required this.isSearchMode,
    super.key,
  });

  final bool isUploading;
  final bool isCreatingFolder;
  final VoidCallback onUploadPressed;
  final VoidCallback onCreateFolderPressed;
  final VoidCallback onRefreshPressed;
  final bool isSearchMode;

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
            icon: const Icon(Icons.upload_rounded),
            label: Text(isUploading ? 'Uploading...' : 'Upload'),
          ),
          const SizedBox(width: 8),
          OutlinedButton.icon(
            onPressed: isCreatingFolder ? null : onCreateFolderPressed,
            icon: const Icon(Icons.create_new_folder_outlined),
            label: Text(isCreatingFolder ? 'Creating...' : 'New Folder'),
          ),
          const Spacer(),
          IconButton(
            onPressed: onRefreshPressed,
            icon: const Icon(Icons.refresh_rounded),
            tooltip: 'Refresh files',
          ),
        ],
      ),
    );
  }
}
