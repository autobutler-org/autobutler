import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

class FileUploadZone extends StatelessWidget {
  const FileUploadZone({
    required this.isUploading,
    required this.isDragging,
    required this.onUploadPressed,
    required this.onDownloadPressed,
    super.key,
  });

  final bool isUploading;
  final bool isDragging;
  final VoidCallback onUploadPressed;
  final VoidCallback? onDownloadPressed;

  @override
  Widget build(BuildContext context) {
    return AnimatedContainer(
      duration: const Duration(milliseconds: 150),
      margin: const EdgeInsets.symmetric(horizontal: 0),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        border: Border.all(
          color: isDragging ? AutobutlerColors.primary : AutobutlerColors.border,
          width: isDragging ? 2 : 1,
          strokeAlign: BorderSide.strokeAlignInside,
        ),
        borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
        color: isDragging
            ? AutobutlerColors.primary.withValues(alpha: 0.05)
            : AutobutlerColors.card,
      ),
      child: Row(
        children: [
          GestureDetector(
            onTap: isUploading ? null : onUploadPressed,
            child: Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                border: Border.all(color: AutobutlerColors.border),
                borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
                color: AutobutlerColors.input,
              ),
              child: Icon(
                isUploading ? Icons.hourglass_top_rounded : Icons.upload_rounded,
                size: 18,
                color: AutobutlerColors.secondaryForeground,
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  isDragging ? 'Drop files here' : (isUploading ? 'Uploading...' : 'Drag & drop to upload'),
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                    color: AutobutlerColors.cardForeground,
                  ),
                ),
                const Text(
                  'Or click to choose files',
                  style: TextStyle(
                    fontSize: 12,
                    color: AutobutlerColors.mutedForeground,
                  ),
                ),
              ],
            ),
          ),
          if (onDownloadPressed != null) ...[
            _actionChip(
              icon: Icons.download_rounded,
              label: 'Download',
              onTap: onDownloadPressed!,
            ),
            const SizedBox(width: 8),
          ],
        ],
      ),
    );
  }

  Widget _actionChip({
    required IconData icon,
    required String label,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          border: Border.all(color: AutobutlerColors.border),
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          color: AutobutlerColors.input,
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 16, color: AutobutlerColors.secondaryForeground),
            const SizedBox(width: 6),
            Text(
              label,
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w500,
                color: AutobutlerColors.secondaryForeground,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
