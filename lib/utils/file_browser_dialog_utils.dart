import 'package:autobutler/utils/autobutler_widget.dart';
import 'package:flutter/material.dart';

Future<String?> promptForFolderName(BuildContext context) async {
  final value = await _promptForText(
    context: context,
    title: 'New Folder',
    hintText: 'Folder name',
    confirmLabel: 'Create',
  );

  final normalized = value?.trim() ?? '';
  if (normalized.isEmpty) {
    return null;
  }

  return normalized.replaceAll(RegExp(r'^/+|/+$'), '');
}

Future<String?> promptForMoveRenamePath(BuildContext context) {
  return _promptForText(
    context: context,
    title: 'Move / Rename',
    hintText: 'New name or path',
    confirmLabel: 'Save',
  );
}

Future<String?> promptForSearchQuery(BuildContext context) {
  return _promptForText(
    context: context,
    title: 'Search files',
    hintText: 'Search term',
    confirmLabel: 'Search',
  );
}

Future<bool?> confirmDelete(BuildContext context, String itemName) async {
  await Future<void>.delayed(Duration.zero);
  if (!context.mounted) {
    return null;
  }

  return AutobutlerWidget.showDialog<bool>(
    context,
    useRootNavigator: true,
    builder: (dialogContext) {
      return AutobutlerWidget.alertDialog(
        title: const Text('Delete'),
        content: Text('Delete $itemName?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: const Text('Delete'),
          ),
        ],
      );
    },
  );
}

Future<String?> _promptForText({
  required BuildContext context,
  required String title,
  required String hintText,
  required String confirmLabel,
}) async {
  await Future<void>.delayed(Duration.zero);
  if (!context.mounted) {
    return null;
  }

  final textController = TextEditingController();
  final String? value;
  try {
    value = await AutobutlerWidget.showDialog(
      context,
      useRootNavigator: true,
      builder: (dialogContext) {
        return AutobutlerWidget.alertDialog(
          title: Text(title),
          content: Padding(
            padding: const EdgeInsets.only(top: 12),
            child: AutobutlerWidget.textField(
              context,
              controller: textController,
              autofocus: true,
              hintText: hintText,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) {
                Navigator.of(dialogContext).pop(textController.text.trim());
              },
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('Cancel'),
            ),
            TextButton(
              autofocus: true,
              onPressed: () {
                Navigator.of(dialogContext).pop(textController.text.trim());
              },
              child: Text(confirmLabel),
            ),
          ],
        );
      },
    );
  } finally {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      textController.dispose();
    });
  }

  final normalized = (value ?? '').trim();
  if (normalized.isEmpty) {
    return null;
  }

  return normalized;
}
