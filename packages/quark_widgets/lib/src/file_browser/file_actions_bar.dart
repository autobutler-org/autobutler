import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

import '../theme/quark_tokens.dart';

/// The upload and new-folder row above a file listing.
///
/// It renders nothing at all in search mode, where the results are the whole
/// point of the view and these actions do not apply. Upload progress is an
/// input: the bar shows a determinate spinner and a running count when the app
/// says a batch is in flight.
///
/// Key prefixes: `file_actions_upload` and `file_actions_new_folder`.
///
/// ```dart
/// FileActionsBar(
///   isUploading: controller.isUploading,
///   isCreatingFolder: controller.isCreatingFolder,
///   isSearchMode: controller.isSearching,
///   onUploadPressed: controller.pickAndUpload,
///   onCreateFolderPressed: controller.createFolder,
/// );
/// ```
class FileActionsBar extends StatelessWidget {
  /// Creates the actions row for a file listing.
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

  /// Whether an upload is in flight. Disables the upload button and swaps its
  /// icon for progress.
  final bool isUploading;

  /// Whether a folder is being created. Disables the new-folder button.
  final bool isCreatingFolder;

  /// Starts the upload flow, usually opening the platform file picker.
  final VoidCallback onUploadPressed;

  /// Starts the new-folder flow, usually prompting for a name.
  final VoidCallback onCreateFolderPressed;

  /// Whether the listing is showing search results, which hides the bar.
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
    final tokens = QuarkTokens.of(context);

    return Padding(
      padding: EdgeInsets.fromLTRB(
        tokens.spacingMd,
        tokens.spacingMd,
        tokens.spacingMd,
        tokens.spacingSm + tokens.spacingXs,
      ),
      // Both buttons are Flexible so their labels are clipped on a narrow phone
      // rather than overflowing the row (#1599).
      child: Row(
        children: [
          Flexible(
            child: FilledButton.tonalIcon(
              key: const ValueKey('file_actions_upload'),
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
                  : const Icon(QuarkIcons.upload_rounded),
              label: Text(_uploadLabel, overflow: TextOverflow.ellipsis),
            ),
          ),
          SizedBox(width: tokens.spacingSm),
          Flexible(
            child: OutlinedButton.icon(
              key: const ValueKey('file_actions_new_folder'),
              onPressed: isCreatingFolder ? null : onCreateFolderPressed,
              icon: const Icon(QuarkIcons.create_new_folder_outlined),
              label: Text(
                isCreatingFolder ? 'Creating...' : 'New Folder',
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
