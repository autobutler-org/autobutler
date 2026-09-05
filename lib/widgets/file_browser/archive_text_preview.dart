import 'package:flutter/material.dart';

/// Read-only view of a text file previewed from inside an archive, where the
/// bytes are already in hand and there is no path on disk to open an editor on.
class ArchiveTextPreview extends StatelessWidget {
  const ArchiveTextPreview({required this.name, required this.text, super.key});

  final String name;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(name)),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: SelectableText(
          text,
          style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
        ),
      ),
    );
  }
}
