import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';
import 'package:quark/services/content_search_service.dart';
import 'package:quark_icons/quark_icons.dart';

/// One full-text search hit in the docs list.
class DocContentResultTile extends StatelessWidget {
  final ContentSearchResult result;

  const DocContentResultTile({required this.result, super.key});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return ListTile(
      leading: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: colorScheme.tertiaryContainer,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Icon(
          QuarkIcons.search_rounded,
          size: 18,
          color: colorScheme.onTertiaryContainer,
        ),
      ),
      title: Text(
        result.filename,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        result.plainSnippet,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(
          color: colorScheme.onSurface.withValues(alpha: 0.6),
          fontSize: 12,
        ),
      ),
      onTap: () => context.push(
        AppRoutes.docFile(result.relPath, serial: result.deviceSerial),
      ),
    );
  }
}
