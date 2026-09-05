import 'package:flutter/material.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark_icons/quark_icons.dart';

/// One document in the docs list.
class DocTile extends StatelessWidget {
  final FileNode node;
  final VoidCallback onTap;

  const DocTile({required this.node, required this.onTap, super.key});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    // Strip the filename from dirPath to get the folder path for the subtitle.
    final folder = node.dirPath.contains('/')
        ? node.dirPath.substring(0, node.dirPath.lastIndexOf('/'))
        : '';
    final subtitle = [
      if (node.deviceName.isNotEmpty) node.deviceName,
      if (folder.isNotEmpty) folder,
    ].join(' · ');

    return ListTile(
      leading: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: colorScheme.primary.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(6),
        ),
        child: Icon(
          QuarkIcons.description_outlined,
          size: 18,
          color: colorScheme.primary,
        ),
      ),
      title: Text(
        node.name.replaceAll(RegExp(r'\.qdoc$', caseSensitive: false), ''),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: subtitle.isNotEmpty
          ? Text(
              subtitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 12,
                color: colorScheme.onSurface.withValues(alpha: 0.55),
              ),
            )
          : null,
      onTap: onTap,
    );
  }
}
