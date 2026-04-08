import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

/// Describes a creatable file type shown in the new-file dialog.
class NewFileType {
  const NewFileType({
    required this.label,
    required this.extension,
    required this.icon,
  });

  final String label;
  final String extension; // includes the dot, e.g. ".abdoc"
  final IconData icon;
}

/// The file types available for creation. Extend this list when more types
/// (spreadsheets, presentations, etc.) are added.
const _kFileTypes = [
  NewFileType(
    label: 'Document',
    extension: '.abdoc',
    icon: Icons.edit_document,
  ),
];

/// Shows the new-file dialog and returns the chosen filename (with extension),
/// or null if the user cancelled.
///
/// Usage:
/// ```dart
/// final name = await showNewFileDialog(context);
/// if (name != null) { /* create the file */ }
/// ```
Future<String?> showNewFileDialog(BuildContext context) {
  return showDialog<String>(
    context: context,
    builder: (_) => const _NewFileDialog(),
  );
}

class _NewFileDialog extends StatefulWidget {
  const _NewFileDialog();

  @override
  State<_NewFileDialog> createState() => _NewFileDialogState();
}

class _NewFileDialogState extends State<_NewFileDialog> {
  NewFileType _selected = _kFileTypes.first;
  final _nameController = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  String? _validateName(String? value) {
    final v = value?.trim() ?? '';
    if (v.isEmpty) return 'Name cannot be empty';
    if (v.contains('/') || v.contains('\\')) {
      return 'Name cannot contain slashes';
    }
    if (v.contains('\x00')) return 'Name contains invalid characters';
    return null;
  }

  void _submit() {
    if (!_formKey.currentState!.validate()) return;
    final name = _nameController.text.trim();
    Navigator.of(context).pop('$name${_selected.extension}');
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('New file'),
      contentPadding: const EdgeInsets.fromLTRB(24, 16, 24, 0),
      content: SizedBox(
        width: 360,
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // ── Type picker ─────────────────────────────────────────────
              Text(
                'File type',
                style: Theme.of(context).textTheme.labelMedium?.copyWith(
                  color: Theme.of(
                    context,
                  ).colorScheme.onSurface.withValues(alpha: 0.5),
                ),
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: _kFileTypes.map((type) {
                  final isSelected = type == _selected;
                  return _TypeCard(
                    type: type,
                    isSelected: isSelected,
                    onTap: () => setState(() => _selected = type),
                  );
                }).toList(),
              ),
              const SizedBox(height: 20),
              // ── Name input ──────────────────────────────────────────────
              Text(
                'Name',
                style: Theme.of(context).textTheme.labelMedium?.copyWith(
                  color: Theme.of(
                    context,
                  ).colorScheme.onSurface.withValues(alpha: 0.5),
                ),
              ),
              const SizedBox(height: 8),
              TextFormField(
                controller: _nameController,
                autofocus: true,
                decoration: InputDecoration(
                  hintText: 'Untitled document',
                  suffixText: _selected.extension,
                  border: const OutlineInputBorder(),
                ),
                textInputAction: TextInputAction.done,
                onFieldSubmitted: (_) => _submit(),
                validator: _validateName,
              ),
              const SizedBox(height: 4),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Create')),
      ],
    );
  }
}

class _TypeCard extends StatelessWidget {
  const _TypeCard({
    required this.type,
    required this.isSelected,
    required this.onTap,
  });

  final NewFileType type;
  final bool isSelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        width: 88,
        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
        decoration: BoxDecoration(
          color: isSelected
              ? colorScheme.primaryContainer
              : colorScheme.surfaceContainerHighest,
          border: Border.all(
            color: isSelected ? colorScheme.primary : colorScheme.outline,
            width: isSelected ? 2 : 1,
          ),
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              type.icon,
              size: 28,
              color: isSelected
                  ? colorScheme.primary
                  : colorScheme.onSurface.withValues(alpha: 0.5),
            ),
            const SizedBox(height: 6),
            Text(
              type.label,
              style: TextStyle(
                fontSize: 12,
                fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
                color: isSelected ? colorScheme.primary : colorScheme.onSurface,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}
