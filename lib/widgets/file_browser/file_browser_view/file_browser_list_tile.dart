import 'package:flutter/material.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/widgets/file_browser/file_browser_view.dart';
import 'package:quark/widgets/file_browser/file_browser_view/file_list_leading.dart';
import 'package:quark/widgets/file_browser/file_browser_view/file_node_display.dart';
import 'package:quark_icons/quark_icons.dart';

/// Runs one of the row's menu entries. The [BuildContext] is the one the menu
/// item was built with, which is what the caller checks for mounting before it
/// touches the tree.
typedef FileMenuActionDispatch =
    void Function(BuildContext context, FileNode item, FileMenuAction action);

/// One file or folder in the list view.
class FileBrowserListTile extends StatelessWidget {
  const FileBrowserListTile({
    required this.item,
    required this.isSelected,
    required this.extractingPaths,
    required this.showFileSizeAndMenu,
    required this.inArchive,
    required this.isSearchMode,
    required this.selectionMode,
    required this.onDispatchMenuAction,
    required this.onOpenDirectory,
    this.onNavigateToFolder,
    this.onSelectionChanged,
    super.key,
  });

  final FileNode item;
  final bool isSelected;

  /// `FileNode.apiPath` values with an extraction in flight; the owner mutates
  /// this set and rebuilds, so it is read rather than copied.
  final Set<String> extractingPaths;
  final bool showFileSizeAndMenu;
  final bool inArchive;
  final bool isSearchMode;
  final bool selectionMode;
  final FileMenuActionDispatch onDispatchMenuAction;
  final void Function(FileNode) onOpenDirectory;
  final void Function(FileNode)? onNavigateToFolder;
  final void Function(FileNode node, {required bool enterSelectionMode})?
  onSelectionChanged;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Material(
      color: isSelected
          ? colors.primaryContainer.withValues(alpha: 0.35)
          : Colors.transparent,
      child: ListTile(
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 2),
        leading: selectionMode
            ? Checkbox(
                value: isSelected,
                onChanged: (_) =>
                    onSelectionChanged?.call(item, enterSelectionMode: false),
              )
            : FileListLeading(item: item),
        title: Row(
          children: [
            Expanded(
              flex: 5,
              child: Text(
                item.name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            Expanded(
              flex: 2,
              child: Text(
                item.deviceName,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: colors.onSurfaceVariant),
              ),
            ),
            if (showFileSizeAndMenu)
              Expanded(
                flex: 2,
                child: Text(
                  formatFileSize(
                    item.size,
                    item.isDir,
                    compressedSize: item.compressedSize,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: colors.onSurfaceVariant),
                ),
              ),
          ],
        ),
        trailing: showFileSizeAndMenu
            ? PopupMenuButton<FileMenuAction>(
                icon: const Icon(QuarkIcons.more_vert),
                itemBuilder: (context) => [
                  PopupMenuItem<FileMenuAction>(
                    value: FileMenuAction.download,
                    onTap: () => onDispatchMenuAction(
                      context,
                      item,
                      FileMenuAction.download,
                    ),
                    child: const Text('Download'),
                  ),
                  if (!inArchive)
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.moveRename,
                      onTap: () => onDispatchMenuAction(
                        context,
                        item,
                        FileMenuAction.moveRename,
                      ),
                      child: const Text('Move/Rename'),
                    ),
                  if (!inArchive)
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.delete,
                      onTap: () => onDispatchMenuAction(
                        context,
                        item,
                        FileMenuAction.delete,
                      ),
                      child: const Text('Delete'),
                    ),
                  if (!inArchive && isArchiveNode(item))
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.extractHere,
                      enabled: !extractingPaths.contains(item.apiPath),
                      onTap: () => onDispatchMenuAction(
                        context,
                        item,
                        FileMenuAction.extractHere,
                      ),
                      child: extractingPaths.contains(item.apiPath)
                          ? const Row(
                              children: [
                                SizedBox(
                                  width: 16,
                                  height: 16,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                ),
                                SizedBox(width: 8),
                                Text('Extracting...'),
                              ],
                            )
                          : const Text('Extract here'),
                    ),
                  if (isSearchMode && onNavigateToFolder != null)
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.navigateToFolder,
                      onTap: () => onNavigateToFolder!(item),
                      child: const Text('Navigate to folder'),
                    ),
                ],
              )
            : null,
        onTap: selectionMode
            ? () => onSelectionChanged?.call(item, enterSelectionMode: false)
            : () => onOpenDirectory(item),
        onLongPress: inArchive || selectionMode
            ? null
            : () => onSelectionChanged?.call(item, enterSelectionMode: true),
      ),
    );
  }
}
