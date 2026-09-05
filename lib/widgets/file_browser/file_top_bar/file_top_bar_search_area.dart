import 'package:flutter/material.dart';
import 'package:quark/widgets/file_browser/file_top_bar/file_top_bar_search_field.dart';
import 'package:quark/widgets/file_browser/file_top_bar/top_bar_icon_button.dart';
import 'package:quark_icons/quark_icons.dart';

/// Inline search. When collapsed shows just the search icon pinned to the
/// right; when expanded the field fills all available width.
///
/// This widget lives inside an [Expanded] in the top row, so it always has
/// bounded horizontal constraints — no double.infinity needed.
class FileTopBarSearchArea extends StatelessWidget {
  const FileTopBarSearchArea({
    required this.expanded,
    required this.controller,
    required this.focusNode,
    required this.onOpen,
    required this.onChanged,
    required this.onClose,
    super.key,
  });

  final bool expanded;
  final TextEditingController controller;
  final FocusNode focusNode;
  final VoidCallback onOpen;
  final ValueChanged<String> onChanged;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    if (expanded) {
      // Fill all available space with the text field.
      return FileTopBarSearchField(
        controller: controller,
        focusNode: focusNode,
        onChanged: onChanged,
        onClose: onClose,
      );
    }

    // Collapsed: push the search icon to the trailing edge.
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        TopBarIconButton(
          icon: QuarkIcons.search_rounded,
          onTap: onOpen,
          tooltip: 'Search',
        ),
      ],
    );
  }
}
