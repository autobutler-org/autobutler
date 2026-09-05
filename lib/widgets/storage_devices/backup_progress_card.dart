import 'package:flutter/material.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark_icons/quark_icons.dart';

/// Progress of the running (or just-finished) snapshot backup job.
class BackupProgressCard extends StatelessWidget {
  final BackupJobStatus status;
  const BackupProgressCard({required this.status, super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final pct = (status.progress * 100).toStringAsFixed(0);

    String statusText;
    switch (status.status) {
      case 'PENDING':
        statusText = 'Preparing backup...';
      case 'SCANNING':
        statusText = 'Scanning files...';
      case 'COPYING':
        statusText = 'Copying files ($pct%)';
      case 'COMPLETED':
        statusText = 'Backup complete';
      case 'FAILED':
        statusText = 'Backup failed';
      default:
        statusText = status.status;
    }

    return Card(
      color: status.isFailed
          ? theme.colorScheme.errorContainer
          : theme.colorScheme.primaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  status.isRunning
                      ? QuarkIcons.backup_outlined
                      : status.isComplete
                      ? QuarkIcons.check_circle_outline
                      : QuarkIcons.error_outline,
                  size: 20,
                ),
                const SizedBox(width: 8),
                Text(statusText, style: theme.textTheme.titleSmall),
              ],
            ),
            if (status.isRunning) ...[
              const SizedBox(height: 12),
              LinearProgressIndicator(value: status.progress),
              const SizedBox(height: 8),
              Text(
                '${status.filesCopied} / ${status.totalFiles} files  ·  '
                '${StorageDevice.formatBytes(status.bytesCopied)} / '
                '${StorageDevice.formatBytes(status.totalBytes)}',
                style: theme.textTheme.bodySmall,
              ),
            ],
            if (status.isComplete) ...[
              const SizedBox(height: 8),
              Text(
                '${status.filesCopied} files copied, '
                '${status.filesSkipped} skipped  ·  '
                '${StorageDevice.formatBytes(status.bytesCopied)}',
                style: theme.textTheme.bodySmall,
              ),
            ],
            if (status.isFailed && status.errorMsg != null) ...[
              const SizedBox(height: 8),
              Text(
                status.errorMsg!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
