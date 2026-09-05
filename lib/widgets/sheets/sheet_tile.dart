import 'package:flutter/material.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark_icons/quark_icons.dart';

/// One spreadsheet in the sheets list.
class SheetTile extends StatelessWidget {
  final FileNode node;
  final VoidCallback onTap;

  const SheetTile({required this.node, required this.onTap, super.key});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

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
          color: Colors.green.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(6),
        ),
        child: Icon(
          QuarkIcons.table_chart_outlined,
          size: 18,
          color: Colors.green.shade600,
        ),
      ),
      title: Text(
        node.name.replaceAll(RegExp(r'\.qsheet$', caseSensitive: false), ''),
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
