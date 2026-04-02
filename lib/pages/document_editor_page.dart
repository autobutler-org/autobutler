import 'dart:async';
import 'dart:convert';

import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

/// Full-screen rich text editor for AutoButler native documents (.abdoc).
///
/// Documents are stored as Quill Delta JSON:
///   {"ops": [{"insert": "Hello\n"}]}
///
/// Load: GET /api/v1/cirrus/download?filePath={path}
/// Save: POST /api/v1/cirrus/upload/{parentDir}  (multipart, replaces file)
class DocumentEditorPage extends StatefulWidget {
  /// The Cirrus API path to the .abdoc file, e.g. "my-docs/notes.abdoc".
  final String filePath;

  /// USB serial of the device holding the file; empty for internal devices.
  final String deviceSerial;

  const DocumentEditorPage({
    required this.filePath,
    this.deviceSerial = '',
    super.key,
  });

  @override
  State<DocumentEditorPage> createState() => _DocumentEditorPageState();
}

class _DocumentEditorPageState extends State<DocumentEditorPage> {
  late QuillController _controller;
  final FocusNode _editorFocus = FocusNode();
  final ScrollController _scrollController = ScrollController();

  bool _loading = true;
  bool _saving = false;
  bool _dirty = false;
  String? _error;

  // ── Auto-save ─────────────────────────────────────────────────────────────
  static const _prefKeyAutoSave = 'document_editor_auto_save';
  static const _autoSaveDelay = Duration(seconds: 2);

  bool _autoSaveEnabled = true;
  Timer? _autoSaveTimer;

  /// Display name shown in the app bar (filename without extension).
  late String _displayName;

  @override
  void initState() {
    super.initState();
    _displayName = _nameFromPath(widget.filePath);
    _controller = QuillController.basic();
    _loadAutoSavePref();
    _loadDocument();
  }

  @override
  void dispose() {
    _autoSaveTimer?.cancel();
    _controller.dispose();
    _editorFocus.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  // ── Persistence ───────────────────────────────────────────────────────────

  String _nameFromPath(String path) {
    final name = path.split('/').last;
    return name.endsWith('.abdoc') ? name.substring(0, name.length - 6) : name;
  }

  Future<void> _loadAutoSavePref() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    setState(() {
      _autoSaveEnabled = prefs.getBool(_prefKeyAutoSave) ?? true;
    });
  }

  Future<void> _setAutoSaveEnabled(bool value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_prefKeyAutoSave, value);
    if (!mounted) return;
    setState(() => _autoSaveEnabled = value);
    if (!value) {
      _autoSaveTimer?.cancel();
      _autoSaveTimer = null;
    }
  }

  Future<void> _loadDocument() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final bytes = await CirrusService.downloadFileBytes(
        widget.filePath,
        serial: serialOrNull(widget.deviceSerial),
      );

      if (!mounted) return;

      if (bytes == null || bytes.isEmpty) {
        setState(() {
          _loading = false;
          _dirty = false;
        });
        _controller.addListener(_onDocumentChanged);
        return;
      }

      final jsonString = utf8.decode(bytes);
      final jsonData = jsonDecode(jsonString) as Map<String, dynamic>?;

      Document doc;
      if (jsonData != null && jsonData['ops'] is List) {
        doc = Document.fromJson(jsonData['ops'] as List);
      } else {
        doc = Document();
      }

      if (!mounted) return;
      setState(() {
        _controller = QuillController(
          document: doc,
          selection: const TextSelection.collapsed(offset: 0),
        );
        _loading = false;
        _dirty = false;
      });
      _controller.addListener(_onDocumentChanged);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = 'Failed to load document: $e';
      });
    }
  }

  void _onDocumentChanged() {
    if (!mounted) return;
    if (!_dirty) setState(() => _dirty = true);

    // (Re)start the auto-save debounce timer on every change.
    if (_autoSaveEnabled) {
      _autoSaveTimer?.cancel();
      _autoSaveTimer = Timer(_autoSaveDelay, _autoSaveSilently);
    }
  }

  /// Auto-save: saves silently (no snackbar on success, subtle indicator on
  /// failure so it doesn't interrupt the writing flow).
  Future<void> _autoSaveSilently() async {
    if (!_dirty || _saving || !mounted) return;
    try {
      await _doSave();
    } catch (e) {
      if (!mounted) return;
      // Non-blocking — just show a brief indicator.
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Auto-save failed: $e'),
          duration: const Duration(seconds: 3),
        ),
      );
    }
  }

  Future<void> _saveDocument() async {
    _autoSaveTimer?.cancel();
    try {
      await _doSave();
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Saved')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Save failed: $e')));
    }
  }

  Future<void> _doSave() async {
    if (_saving) return;
    setState(() => _saving = true);

    try {
      final ops = _controller.document.toDelta().toJson();
      final jsonString = jsonEncode({'ops': ops});
      final bytes = utf8.encode(jsonString);

      final fileName = '$_displayName.abdoc';
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
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);
      rethrow;
    }
  }

  // ── Build ─────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: !_dirty,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final leave = await _confirmDiscard(context);
        if (leave && context.mounted) {
          Navigator.of(context).pop();
        }
      },
      child: Scaffold(
        appBar: AppBar(
          title: Row(
            children: [
              Expanded(
                child: Text(
                  _dirty ? '$_displayName •' : _displayName,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          actions: [
            // Auto-save toggle
            IconButton(
              icon: Icon(
                _autoSaveEnabled
                    ? Icons.cloud_sync_outlined
                    : Icons.cloud_off_outlined,
              ),
              tooltip: _autoSaveEnabled
                  ? 'Auto-save on — tap to disable'
                  : 'Auto-save off — tap to enable',
              onPressed: () => _setAutoSaveEnabled(!_autoSaveEnabled),
            ),
            if (_saving)
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 16),
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
                onPressed: _dirty ? _saveDocument : null,
              ),
          ],
        ),
        body: _buildBody(context),
      ),
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
            const Icon(Icons.error_outline, size: 48),
            const SizedBox(height: 12),
            Text(_error!, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            ElevatedButton(
              onPressed: _loadDocument,
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    return Column(
      children: [
        QuillSimpleToolbar(
          controller: _controller,
          config: const QuillSimpleToolbarConfig(
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
        const Divider(height: 1),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
            child: QuillEditor.basic(
              controller: _controller,
              focusNode: _editorFocus,
              scrollController: _scrollController,
              config: const QuillEditorConfig(
                autoFocus: true,
                expands: false,
                padding: EdgeInsets.zero,
                placeholder: 'Start writing…',
              ),
            ),
          ),
        ),
      ],
    );
  }

  Future<bool> _confirmDiscard(BuildContext context) async {
    final result = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Unsaved changes'),
        content: const Text('You have unsaved changes. Leave without saving?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Stay'),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Leave'),
          ),
        ],
      ),
    );
    return result ?? false;
  }
}
