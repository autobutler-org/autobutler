import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

class FileStorageFooter extends StatelessWidget {
  const FileStorageFooter({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: AutobutlerColors.sidebar,
        border: Border(
          top: BorderSide(color: AutobutlerColors.border),
        ),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          const Icon(
            Icons.storage_rounded,
            size: 14,
            color: AutobutlerColors.mutedForeground,
          ),
          const SizedBox(width: 8),
          const Text(
            'Storage',
            style: TextStyle(
              fontSize: 12,
              color: AutobutlerColors.mutedForeground,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Container(
              height: 8,
              constraints: const BoxConstraints(maxWidth: 320),
              decoration: BoxDecoration(
                color: AutobutlerColors.input,
                border: Border.all(color: AutobutlerColors.border),
                borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
              ),
              clipBehavior: Clip.antiAlias,
              child: FractionallySizedBox(
                alignment: Alignment.centerLeft,
                widthFactor: 0.0,
                child: Container(
                  decoration: BoxDecoration(
                    color: AutobutlerColors.primary,
                    borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
                  ),
                ),
              ),
            ),
          ),
          const Spacer(),
        ],
      ),
    );
  }
}
