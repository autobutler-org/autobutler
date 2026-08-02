import 'dart:convert';

import 'package:autobutler/router.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:autobutler/widgets/layout/theme_toggle_button.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;

/// A simple plaintext editor for text-like files (txt, md, json, yaml, etc.)
///
/// For files with recognisable code extensions the editor shows a **Preview**
/// toggle in the app bar. In preview mode the content is rendered as a
/// read-only, line-numbered code view using a monospace font. In edit mode
/// the user gets a plain [TextField]. Both modes share the same underlying
/// text controller so switching is lossless.
class PlaintextEditorPage extends StatefulWidget {
  final String filePath;
  final String deviceSerial;

  const PlaintextEditorPage({
    required this.filePath,
    this.deviceSerial = '',
    super.key,
  });

  @override
  State<PlaintextEditorPage> createState() => _PlaintextEditorPageState();
}

// Extensions that get the line-numbered code preview toggle.
const _codeExtensions = {
  '.json',
  '.yaml',
  '.yml',
  '.xml',
  '.html',
  '.htm',
  '.css',
  '.js',
  '.ts',
  '.go',
  '.py',
  '.sh',
  '.bash',
  '.zsh',
  '.dart',
  '.swift',
  '.kt',
  '.java',
  '.c',
  '.cpp',
  '.h',
  '.rs',
  '.rb',
  '.php',
  '.toml',
  '.ini',
  '.cfg',
  '.conf',
  '.env',
  '.log',
  '.md',
  '.csv',
};

class _PlaintextEditorPageState extends State<PlaintextEditorPage> {
  final TextEditingController _textController = TextEditingController();

  bool _loading = true;
  bool _saving = false;
  bool _dirty = false;
  bool _previewMode = false;
  String? _error;

  late String _displayName;
  late bool _supportsPreview;

  @override
  void initState() {
    super.initState();
    _displayName = widget.filePath.split('/').last;
    final ext = _ext(_displayName);
    _supportsPreview = _codeExtensions.contains(ext);
    // Default to preview for files that support it.
    _previewMode = _supportsPreview;
    _textController.addListener(_onTextChanged);
    _loadFile();
  }

  @override
  void dispose() {
    _textController.dispose();
    super.dispose();
  }

  static String _ext(String name) {
    final idx = name.lastIndexOf('.');
    if (idx < 0 || idx == name.length - 1) return '';
    return name.substring(idx).toLowerCase();
  }

  void _onTextChanged() {
    if (!_dirty && mounted) {
      setState(() => _dirty = true);
    }
  }

  Future<void> _loadFile() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final serial = serialOrNull(widget.deviceSerial);
      final bytes = await CirrusService.downloadFileBytes(
        widget.filePath,
        serial: serial,
      );
      if (!mounted) return;
      final text = bytes != null && bytes.isNotEmpty
          ? utf8.decode(bytes, allowMalformed: true)
          : '';
      _textController.removeListener(_onTextChanged);
      _textController.text = text;
      _textController.addListener(_onTextChanged);
      setState(() {
        _loading = false;
        _dirty = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = 'Failed to load file: $e';
      });
    }
  }

  Future<void> _saveFile() async {
    if (_saving) return;
    setState(() => _saving = true);
    try {
      final bytes = utf8.encode(_textController.text);
      final fileName = _displayName;
      final parentDir = parentPath(widget.filePath);
      final serial = serialOrNull(widget.deviceSerial);
      final file = http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: fileName,
      );
      await CirrusService.uploadFilesFromFormData(
        parentDir,
        [file],
        serial: serial,
        overwrite: true,
      );
      if (!mounted) return;
      setState(() {
        _saving = false;
        _dirty = false;
      });
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Saved')));
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Save failed: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final title = _dirty ? '$_displayName •' : _displayName;

    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          tooltip: 'Back',
          onPressed: () {
            if (context.canPop()) {
              context.pop();
            } else {
              context.go(AppRoutes.cirrus);
            }
          },
        ),
        actions: [
          if (_supportsPreview)
            Tooltip(
              message: _previewMode
                  ? 'Switch to edit mode'
                  : 'Switch to preview',
              child: IconButton(
                icon: Icon(
                  _previewMode ? Icons.edit_outlined : Icons.preview_outlined,
                ),
                onPressed: () => setState(() => _previewMode = !_previewMode),
              ),
            ),
          if (_saving)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 12),
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          else
            IconButton(
              icon: const Icon(Icons.save_outlined),
              tooltip: 'Save',
              onPressed: _dirty ? _saveFile : null,
            ),
          const ThemeToggleButton(),
        ],
      ),
      body: _buildBody(context),
    );
  }

  Widget _buildBody(BuildContext context) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(_error!, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton(onPressed: _loadFile, child: const Text('Retry')),
          ],
        ),
      );
    }

    if (_previewMode) {
      return _CodePreview(text: _textController.text);
    }

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: TextField(
        controller: _textController,
        maxLines: null,
        keyboardType: TextInputType.multiline,
        style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
        decoration: const InputDecoration(
          border: InputBorder.none,
          hintText: 'Empty file',
          isCollapsed: true,
        ),
      ),
    );
  }
}

/// A read-only, line-numbered code view.
///
/// Renders each line of [text] in a `ListView.builder` with:
/// - A fixed-width gutter showing the 1-based line number
/// - The line content in a monospace font
/// - Horizontal scrolling per-line via [SingleChildScrollView]
///
/// Uses [SelectableText] so users can copy code snippets.
class _CodePreview extends StatelessWidget {
  final String text;

  const _CodePreview({required this.text});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final lines = text.isEmpty ? const [''] : text.split('\n');
    final gutterWidth = _gutterWidth(lines.length);
    final codeBg = theme.brightness == Brightness.dark
        ? theme.colorScheme.surfaceContainerHighest
        : theme.colorScheme.surfaceContainerLowest;
    final gutterColor = theme.brightness == Brightness.dark
        ? theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.6)
        : theme.colorScheme.surfaceContainer;
    final lineNumColor = theme.colorScheme.onSurface.withValues(alpha: 0.4);
    const codeStyle = TextStyle(
      fontFamily: 'monospace',
      fontSize: 13,
      height: 1.5,
    );

    return Container(
      color: codeBg,
      child: ListView.builder(
        itemCount: lines.length,
        itemExtent: 20, // 13px font * 1.5 line-height ≈ 20px
        itemBuilder: (context, i) {
          final lineNum = (i + 1).toString().padLeft(gutterWidth);
          return Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Gutter
              Container(
                width: gutterWidth * 9.0 + 16, // ~9px per char + padding
                color: gutterColor,
                padding: const EdgeInsets.symmetric(horizontal: 8),
                alignment: Alignment.centerRight,
                child: Text(
                  lineNum,
                  style: codeStyle.copyWith(color: lineNumColor),
                ),
              ),
              // Code line
              Expanded(
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  child: SelectableText(
                    lines[i],
                    style: codeStyle.copyWith(
                      color: theme.colorScheme.onSurface,
                    ),
                    maxLines: 1,
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  /// Returns the number of digits in [lineCount] for gutter sizing.
  static int _gutterWidth(int lineCount) {
    if (lineCount < 10) return 1;
    if (lineCount < 100) return 2;
    if (lineCount < 1000) return 3;
    if (lineCount < 10000) return 4;
    return 5;
  }
}
