import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';

/// The formatting toolbar shown above the document in edit mode.
class DocumentEditorToolbar extends StatelessWidget {
  final QuillController controller;

  /// Replaces Quill's own color picker for the background/highlight button.
  final QuillToolbarColorPickerOnPressedCallback onPickBackgroundColor;

  const DocumentEditorToolbar({
    required this.controller,
    required this.onPickBackgroundColor,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final toolbarTheme = theme.copyWith(
      colorScheme: cs.copyWith(
        onSurface: cs.onSurface,
        surface: cs.surfaceContainer,
        surfaceContainerLow: cs.surfaceContainer,
        surfaceContainer: cs.surfaceContainer,
      ),
      iconTheme: IconThemeData(color: cs.onSurface, size: 16),
      textTheme: theme.textTheme.apply(
        bodyColor: cs.onSurface,
        displayColor: cs.onSurface,
      ),
    );

    return Theme(
      data: toolbarTheme,
      child: Container(
        decoration: BoxDecoration(
          color: cs.surfaceContainer,
          border: Border(bottom: BorderSide(color: cs.outline)),
        ),
        child: QuillSimpleToolbar(
          controller: controller,
          config: QuillSimpleToolbarConfig(
            toolbarIconAlignment: WrapAlignment.center,
            buttonOptions: QuillSimpleToolbarButtonOptions(
              base: QuillToolbarBaseButtonOptions(
                iconTheme: QuillIconTheme(
                  iconButtonUnselectedData: IconButtonData(
                    color: cs.onSurface,
                    style: IconButton.styleFrom(
                      backgroundColor: cs.onSurface.withValues(alpha: 0.05),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6),
                      ),
                    ),
                  ),
                  iconButtonSelectedData: IconButtonData(
                    style: IconButton.styleFrom(
                      foregroundColor: cs.onPrimary,
                      backgroundColor: cs.primary,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6),
                      ),
                    ),
                  ),
                ),
              ),
              selectHeaderStyleDropdownButton:
                  QuillToolbarSelectHeaderStyleDropdownButtonOptions(
                    textStyle: TextStyle(color: cs.onSurface, fontSize: 13),
                  ),
              backgroundColor: QuillToolbarColorButtonOptions(
                customOnPressedCallback: onPickBackgroundColor,
              ),
            ),
            showFontFamily: false,
            showFontSize: false,
            showInlineCode: true,
            showCodeBlock: true,
            showQuote: true,
            showLink: false,
            showSearchButton: false,
            showSubscript: false,
            showSuperscript: false,
          ),
        ),
      ),
    );
  }
}
