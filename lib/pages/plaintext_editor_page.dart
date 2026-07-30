import 'dart:convert';

import 'package:autobutler/router.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:autobutler/widgets/layout/theme_toggle_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_highlight/flutter_highlight.dart';
import 'package:flutter_highlight/themes/atom-one-dark.dart';
import 'package:flutter_highlight/themes/atom-one-light.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;

// ---------------------------------------------------------------------------
// Language detection
// ---------------------------------------------------------------------------

/// Maps a lowercase file extension to a flutter_highlight language name.
/// Falls back to 'plaintext' for unknown extensions.
String _langForExtension(String ext) {
  const map = {
    '.dart': 'dart',
    '.go': 'go',
    '.py': 'python',
    '.js': 'javascript',
    '.ts': 'typescript',
    '.jsx': 'javascript',
    '.tsx': 'typescript',
    '.json': 'json',
    '.yaml': 'yaml',
    '.yml': 'yaml',
    '.xml': 'xml',
    '.html': 'html',
    '.htm': 'html',
    '.css': 'css',
    '.scss': 'css',
    '.sh': 'bash',
    '.bash': 'bash',
    '.zsh': 'bash',
    '.swift': 'swift',
    '.kt': 'kotlin',
    '.java': 'java',
    '.c': 'c',
    '.h': 'c',
    '.cpp': 'cpp',
    '.cc': 'cpp',
    '.cs': 'cs',
    '.rs': 'rust',
    '.rb': 'ruby',
    '.php': 'php',
    '.sql': 'sql',
    '.md': 'markdown',
    '.toml': 'ini',
    '.ini': 'ini',
    '.conf': 'nginx',
    '.dockerfile': 'dockerfile',
    '.proto': 'protobuf',
    '.graphql': 'graphql',
    '.txt': 'plaintext',
  };
  return map[ext.toLowerCase()] ?? 'plaintext';
}

/// Returns true when the extension is a code/markup type that benefits from
/// syntax highlighting. Plain .txt files default to the editable view.
bool _isCodeFile(String ext) =>
    _langForExtension(ext.toLowerCase()) != 'plaintext';

// ---------------------------------------------------------------------------
// Widget
// ---------------------------------------------------------------------------

/// A combined plaintext editor + syntax-highlighted code preview.
///
/// For recognised code/markup extensions the toolbar shows a toggle between
/// ✏️ (edit) and 🎨 (highlight) mode. For plain .txt files only the editor
/// mode is shown (syntax highlighting adds nothing).
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

class _PlaintextEditorPageState extends State<PlaintextEditorPage> {
  final TextEditingController _textController = TextEditingController();

  bool _loading = true;
  bool _saving = false;
  bool _dirty = false;
  bool _highlightMode =
      false; // toggled per-session; defaults to highlight for code files
  String? _error;

  late String _displayName;
  late String _ext;

  @override
  void initState() {
    super.initState();
    _displayName = widget.filePath.split('/').last;
    _ext = _displayName.contains('.')
        ? '.${_displayName.split('.').last.toLowerCase()}'
        : '';
    // Auto-enable syntax highlight for code files.
    _highlightMode = _isCodeFile(_ext);
    _textController.addListener(_onTextChanged);
    _loadFile();
  }

  @override
  void dispose() {
    _textController.dispose();
    super.dispose();
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
    final isCode = _isCodeFile(_ext);

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
          // Highlight / edit mode toggle — only for code files.
          if (isCode && !_loading && _error == null)
            IconButton(
              icon: Icon(
                _highlightMode
                    ? Icons.edit_outlined
                    : Icons.color_lens_outlined,
              ),
              tooltip: _highlightMode
                  ? 'Switch to edit mode'
                  : 'Switch to highlight mode',
              onPressed: () => setState(() {
                _highlightMode = !_highlightMode;
                // Leaving highlight mode resets the dirty flag so the user
                // doesn't see a spurious "unsaved changes" prompt.
                if (!_highlightMode) _dirty = false;
              }),
            ),
          // Save button — only in edit mode.
          if (!_highlightMode)
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

    if (_highlightMode) {
      return _buildHighlightView(context);
    }

    return _buildEditorView();
  }

  // -------------------------------------------------------------------------
  // Syntax-highlighted read-only view
  // -------------------------------------------------------------------------

  Widget _buildHighlightView(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final theme = isDark ? atomOneDarkTheme : atomOneLightTheme;
    final lang = _langForExtension(_ext);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(12),
      child: HighlightView(
        _textController.text.isEmpty ? ' ' : _textController.text,
        language: lang,
        theme: theme,
        padding: const EdgeInsets.all(12),
        textStyle: const TextStyle(
          fontFamily: 'monospace',
          fontSize: 13,
          height: 1.5,
        ),
      ),
    );
  }

  // -------------------------------------------------------------------------
  // Editable plain-text view
  // -------------------------------------------------------------------------

  Widget _buildEditorView() {
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
