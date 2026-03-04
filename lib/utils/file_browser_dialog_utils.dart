import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/utils/autobutler_widget.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:autobutler/widgets/file_browser/file_breadcrumb_bar.dart';
import 'package:autobutler/controllers/file_browser_controller.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

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

// startPath should be the current folder of the item being moved; returns a path
// relative to startPath (e.g. "sub/folder/newname") or a simple name like "newname".
Future<String?> promptForMoveRenamePath(
  BuildContext context, {
  String startPath = '',
  String? initialName,
}) async {
  await Future<void>.delayed(Duration.zero);
  if (!context.mounted) return null;

  final controller = FileBrowserController();
  String currentAbsolutePath = normalizePath(
    startPath,
  ); // normalized with leading slash or empty
  if (currentAbsolutePath.startsWith('/')) {
    // normalizePath returns leading slash for non-empty; keep it
  } else {
    currentAbsolutePath = currentAbsolutePath; // keep empty
  }

  final nameController = TextEditingController(
    text: initialName?.trim().replaceAll('/', '') ?? '',
  );

  final result = await AutobutlerWidget.showDialog<String?>(
    context,
    useRootNavigator: true,
    builder: (dialogContext) {
      bool hasInvalidChar = nameController.text.contains('/');
      return StatefulBuilder(
        builder: (context, setState) {
          Future<List<CirrusFileNode>> filesFuture() {
            return controller.fetchFiles(currentAbsolutePath);
          }

          void openDirectory(CirrusFileNode node) {
            if (!node.isDir) return;
            // Prevent opening the folder that's being moved into itself
            if (initialName != null && initialName.trim().isNotEmpty) {
              final targetOfNode = normalizePath(
                joinPath(startPath, initialName),
              );
              final candidate = normalizePath(
                joinPath(currentAbsolutePath, node.name),
              );
              if (candidate == targetOfNode) {
                // Do nothing to prevent selecting the node itself as a destination
                return;
              }
            }
            setState(() {
              currentAbsolutePath = joinPath(currentAbsolutePath, node.name);
            });
          }

          void goUp() {
            setState(() {
              currentAbsolutePath = parentPath(currentAbsolutePath);
            });
          }

          String relativeToStart() {
            final normStart = normalizePath(startPath);
            final normCurrent = normalizePath(currentAbsolutePath);
            if (normStart.isEmpty) {
              // root start
              return normCurrent.startsWith('/')
                  ? normCurrent.substring(1)
                  : normCurrent;
            }
            if (normCurrent == normStart) return '';
            if (normCurrent.startsWith('$normStart/')) {
              return normCurrent.substring(normStart.length + 1);
            }
            // fallback to absolute with leading slash so callers treat it as absolute
            return normCurrent.startsWith('/') ? normCurrent : '/$normCurrent';
          }

          return AutobutlerWidget.alertDialog(
            title: const Text('Move / Rename'),
            scrollable: true,
            content: SizedBox(
              width: double.maxFinite,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  FileBreadcrumbBar(
                    currentPath: currentAbsolutePath,
                    onGoHome: () {
                      setState(() {
                        currentAbsolutePath = '';
                      });
                    },
                    onGoUp: goUp,
                    onPathSelected: (path) {
                      setState(() {
                        currentAbsolutePath = path;
                      });
                    },
                    isSearchMode: false,
                  ),
                  SizedBox(
                    height: 300,
                    child: FileBrowserView(
                      filesFuture: filesFuture(),
                      onFileMenuAction: (node, action) async {},
                      onOpenDirectory: openDirectory,
                      isGridView: false,
                      showFileSizeAndMenu: false,
                    ),
                  ),
                  const SizedBox(height: 8),
                  // Text field that prevents typing '/' and shows validation
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      AutobutlerWidget.textField(
                        context,
                        controller: nameController,
                        hintText: 'New file name',
                        autofocus: true,
                        textInputAction: TextInputAction.done,
                        onSubmitted: (_) {},
                        onChanged: (v) {
                          setState(() {
                            hasInvalidChar = v.contains('/');
                          });
                        },
                        inputFormatters: [
                          FilteringTextInputFormatter.deny(RegExp(r'/')),
                        ],
                      ),
                      if (hasInvalidChar)
                        Padding(
                          padding: const EdgeInsets.only(top: 6.0),
                          child: Text(
                            'The file name cannot contain "/"',
                            style: TextStyle(
                              color: Theme.of(context).colorScheme.error,
                              fontSize: 12,
                            ),
                          ),
                        ),
                    ],
                  ),
                ],
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
                  final name = nameController.text.trim();
                  if (name.isEmpty || hasInvalidChar) {
                    Navigator.of(dialogContext).pop(null);
                    return;
                  }
                  final rel = relativeToStart();
                  String out;
                  if (rel.isEmpty) {
                    out = name;
                  } else {
                    out = '$rel/$name';
                  }
                  Navigator.of(dialogContext).pop(out);
                },
                child: const Text('Save'),
              ),
            ],
          );
        },
      );
    },
  );

  WidgetsBinding.instance.addPostFrameCallback((_) {
    nameController.dispose();
  });

  final normalized = (result ?? '').trim();
  if (normalized.isEmpty) return null;
  // Collapse duplicate slashes and remove trailing slashes, preserving a single leading
  // slash for absolute targets (e.g. "/parent/file").
  final collapsed = normalized.replaceAll(RegExp(r'/+'), '/');
  return collapsed.replaceAll(RegExp(r'/+$'), '');
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
