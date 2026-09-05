import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The expanded inline search field. Height is explicit so the field fits the
/// 56-px navbar without clipping.
class FileTopBarSearchField extends StatelessWidget {
  const FileTopBarSearchField({
    required this.controller,
    required this.focusNode,
    required this.onChanged,
    required this.onClose,
    super.key,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final ValueChanged<String> onChanged;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return SizedBox(
      height: 36,
      child: Focus(
        // Handle ESC to close without needing a separate KeyboardListener
        // (which requires a managed FocusNode that would outlive rebuilds).
        onKeyEvent: (_, event) {
          if (event is KeyDownEvent &&
              event.logicalKey == LogicalKeyboardKey.escape) {
            onClose();
            return KeyEventResult.handled;
          }
          return KeyEventResult.ignored;
        },
        child: TextField(
          controller: controller,
          focusNode: focusNode,
          autofocus: false,
          decoration: InputDecoration(
            hintText: 'Search files…',
            isDense: true,
            contentPadding: const EdgeInsets.symmetric(vertical: 8),
            prefixIcon: Icon(
              QuarkIcons.search_rounded,
              size: 18,
              color: colorScheme.onSurfaceVariant,
            ),
            suffixIcon: IconButton(
              icon: Icon(
                QuarkIcons.close_rounded,
                size: 16,
                color: colorScheme.onSurfaceVariant,
              ),
              tooltip: 'Close search',
              onPressed: onClose,
            ),
            filled: true,
            fillColor: colorScheme.surfaceContainerHighest,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
              borderSide: BorderSide(color: colorScheme.outline),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
              borderSide: BorderSide(color: colorScheme.outline),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
              borderSide: BorderSide(color: colorScheme.primary),
            ),
          ),
          onChanged: onChanged,
        ),
      ),
    );
  }
}
