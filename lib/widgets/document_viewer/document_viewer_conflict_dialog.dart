import 'package:flutter/material.dart';

/// What the reader chose when "Open as Quark doc" found a `.qdoc` of the
/// same name already there.
enum DocumentConflictChoice {
  /// Leave both files alone.
  cancel,

  /// Convert again, replacing the `.qdoc`.
  replace,

  /// Open the `.qdoc` that is already there.
  openExisting,
}

/// Asks what to do when "Open as Quark doc" would replace [qdocName], a
/// `.qdoc` the reader may have edited since, with a fresh conversion of
/// [documentName]. Pops a [DocumentConflictChoice].
///
/// Keys: `document_conflict_cancel`, `document_conflict_replace` and
/// `document_conflict_open_existing` on its buttons.
class DocumentViewerConflictDialog extends StatelessWidget {
  /// The name of the `.qdoc` already there.
  final String qdocName;

  /// The name of the document being converted.
  final String documentName;

  const DocumentViewerConflictDialog({
    required this.qdocName,
    required this.documentName,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    void choose(DocumentConflictChoice choice) =>
        Navigator.of(context).pop(choice);
    return AlertDialog(
      title: const Text('Document already exists'),
      content: Text(
        '"$qdocName" is already here. Open it, or replace it with a fresh '
        'conversion of "$documentName"?',
      ),
      actions: [
        TextButton(
          key: const ValueKey('document_conflict_cancel'),
          onPressed: () => choose(DocumentConflictChoice.cancel),
          child: const Text('Cancel'),
        ),
        TextButton(
          key: const ValueKey('document_conflict_replace'),
          onPressed: () => choose(DocumentConflictChoice.replace),
          child: const Text('Replace'),
        ),
        FilledButton(
          key: const ValueKey('document_conflict_open_existing'),
          onPressed: () => choose(DocumentConflictChoice.openExisting),
          child: const Text('Open existing'),
        ),
      ],
    );
  }
}
