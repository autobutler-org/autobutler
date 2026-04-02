import 'dart:async';
import 'dart:convert';

import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_quill_to_pdf/flutter_quill_to_pdf.dart';
import 'package:http/http.dart' as http;
import 'package:printing/printing.dart';
import 'package:shared_preferences/shared_preferences.dart';

// ── Editor appearance ─────────────────────────────────────────────────────────

/// The user-selectable editor page appearance, independent of the app theme.
enum EditorPageAppearance {
  /// White page with dark text — classic word processor feel.
  light,

  /// Dark page with light text — night writing mode.
  dark,
}

extension _EditorPageAppearanceX on EditorPageAppearance {
  Color get pageColor => this == EditorPageAppearance.light
      ? Colors.white
      : const Color(0xFF1A1A2E);

  Color get textColor => this == EditorPageAppearance.light
      ? const Color(0xFF1A1A1A)
      : const Color(0xFFE8E8F0);

  Color get mutedColor => this == EditorPageAppearance.light
      ? const Color(0xFF6B7280)
      : const Color(0xFF9CA3AF);

  Color get codeBackground => this == EditorPageAppearance.light
      ? const Color(0xFFF3F4F6)
      : const Color(0xFF2D2D44);

  Color get toolbarColor => this == EditorPageAppearance.light
      ? const Color(0xFFF9FAFB)
      : const Color(0xFF151525);

  Color get dividerColor => this == EditorPageAppearance.light
      ? const Color(0xFFE5E7EB)
      : const Color(0xFF2A2A3E);

  EditorPageAppearance get toggled => this == EditorPageAppearance.light
      ? EditorPageAppearance.dark
      : EditorPageAppearance.light;

  IconData get icon => this == EditorPageAppearance.light
      ? Icons.dark_mode_outlined
      : Icons.light_mode_outlined;

  String get tooltip => this == EditorPageAppearance.light
      ? 'Switch to dark page'
      : 'Switch to light page';
}

// ── Quill styles ──────────────────────────────────────────────────────────────

DefaultStyles _quillStylesForAppearance(
  EditorPageAppearance appearance,
  ThemeData theme,
) {
  final fg = appearance.textColor;
  final muted = appearance.mutedColor;
  final primary = theme.colorScheme.primary;
  final codeBg = appearance.codeBackground;
  final textTheme = theme.textTheme;

  TextStyle base(TextStyle? s) => (s ?? const TextStyle()).copyWith(color: fg);

  return DefaultStyles(
    paragraph: DefaultTextBlockStyle(
      base(textTheme.bodyMedium),
      HorizontalSpacing.zero,
      VerticalSpacing.zero,
      VerticalSpacing.zero,
      null,
    ),
    h1: DefaultTextBlockStyle(
      base(textTheme.headlineLarge).copyWith(fontWeight: FontWeight.bold),
      HorizontalSpacing.zero,
      const VerticalSpacing(16, 4),
      VerticalSpacing.zero,
      null,
    ),
    h2: DefaultTextBlockStyle(
      base(textTheme.headlineMedium).copyWith(fontWeight: FontWeight.bold),
      HorizontalSpacing.zero,
      const VerticalSpacing(12, 4),
      VerticalSpacing.zero,
      null,
    ),
    h3: DefaultTextBlockStyle(
      base(textTheme.headlineSmall).copyWith(fontWeight: FontWeight.bold),
      HorizontalSpacing.zero,
      const VerticalSpacing(8, 4),
      VerticalSpacing.zero,
      null,
    ),
    placeHolder: DefaultTextBlockStyle(
      (textTheme.bodyMedium ?? const TextStyle()).copyWith(color: muted),
      HorizontalSpacing.zero,
      VerticalSpacing.zero,
      VerticalSpacing.zero,
      null,
    ),
    quote: DefaultTextBlockStyle(
      base(
        textTheme.bodyMedium,
      ).copyWith(color: muted, fontStyle: FontStyle.italic),
      const HorizontalSpacing(16, 0),
      const VerticalSpacing(6, 6),
      VerticalSpacing.zero,
      BoxDecoration(
        border: Border(left: BorderSide(color: primary, width: 4)),
      ),
    ),
    inlineCode: InlineCodeStyle(
      style: TextStyle(
        fontFamily: 'monospace',
        fontSize: 13,
        color: muted,
        backgroundColor: codeBg,
      ),
      backgroundColor: codeBg,
      radius: const Radius.circular(4),
    ),
    code: DefaultTextBlockStyle(
      TextStyle(fontFamily: 'monospace', fontSize: 13, color: muted),
      const HorizontalSpacing(12, 12),
      const VerticalSpacing(8, 8),
      VerticalSpacing.zero,
      BoxDecoration(color: codeBg, borderRadius: BorderRadius.circular(8)),
    ),
    color: fg,
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

/// Full-screen rich text editor for AutoButler native documents (.abdoc).
///
/// Documents are stored as Quill Delta JSON:
///   {"ops": [{"insert": "Hello\n"}]}
///
/// Load: GET /api/v1/cirrus/download?filePath={path}
/// Save: POST /api/v1/cirrus/upload/{parentDir}  (multipart, replaces file)
class DocumentEditorPage extends StatefulWidget {
  final String filePath;
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
  bool _exporting = false;
  String? _error;

  // ── Read-only / edit mode ──────────────────────────────────────────────────
  /// Documents open in read-only mode by default (#939).
  bool _isEditing = false;

  // ── Page appearance ────────────────────────────────────────────────────────
  static const _prefKeyAppearance = 'document_editor_appearance';
  EditorPageAppearance _appearance = EditorPageAppearance.light;

  // ── Auto-save ──────────────────────────────────────────────────────────────
  static const _prefKeyAutoSave = 'document_editor_auto_save';
  static const _autoSaveDelay = Duration(seconds: 2);
  bool _autoSaveEnabled = true;
  Timer? _autoSaveTimer;

  late String _displayName;

  @override
  void initState() {
    super.initState();
    _displayName = _nameFromPath(widget.filePath);
    _controller = QuillController.basic();
    _loadPrefs();
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

  // ── Prefs ──────────────────────────────────────────────────────────────────

  String _nameFromPath(String path) {
    final name = path.split('/').last;
    return name.endsWith('.abdoc') ? name.substring(0, name.length - 6) : name;
  }

  Future<void> _loadPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    final appearanceStr = prefs.getString(_prefKeyAppearance);
    setState(() {
      _autoSaveEnabled = prefs.getBool(_prefKeyAutoSave) ?? true;
      _appearance = appearanceStr == 'dark'
          ? EditorPageAppearance.dark
          : EditorPageAppearance.light;
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

  Future<void> _toggleAppearance() async {
    final next = _appearance.toggled;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefKeyAppearance, next.name);
    if (!mounted) return;
    setState(() => _appearance = next);
  }

  // ── Document load/save ─────────────────────────────────────────────────────

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
        // New document — open straight into edit mode.
        setState(() {
          _loading = false;
          _dirty = false;
          _isEditing = true;
        });
        _controller.addListener(_onDocumentChanged);
        _focusEditor();
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
      final controller = QuillController(
        document: doc,
        selection: const TextSelection.collapsed(offset: 0),
        readOnly: true, // Existing documents open read-only (#939).
      );
      setState(() {
        _controller = controller;
        _loading = false;
        _dirty = false;
        _isEditing = false;
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

  void _focusEditor() {
    // On web, the browser needs a frame to recognize the focusable element
    // after switching from read-only. Unfocus first, then re-request on the
    // next frame to ensure the browser hands focus to the editor correctly.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      FocusScope.of(context).unfocus();
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) _editorFocus.requestFocus();
      });
    });
  }

  void _enterEditMode() {
    _controller.readOnly = false;
    setState(() => _isEditing = true);
    _focusEditor();
  }

  void _exitEditMode() {
    _controller.readOnly = true;
    setState(() => _isEditing = false);
  }

  void _onDocumentChanged() {
    if (!mounted) return;
    if (!_dirty) setState(() => _dirty = true);
    if (_autoSaveEnabled) {
      _autoSaveTimer?.cancel();
      _autoSaveTimer = Timer(_autoSaveDelay, _autoSaveSilently);
    }
  }

  Future<void> _autoSaveSilently() async {
    if (!_dirty || _saving || !mounted) return;
    try {
      await _doSave();
    } catch (e) {
      if (!mounted) return;
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

  // ── PDF / Print ────────────────────────────────────────────────────────────

  Future<Uint8List?> _buildPdfBytes() async {
    final converter = PDFConverter(
      document: _controller.document.toDelta(),
      pageFormat: PDFPageFormat.a4,
      fallbacks: const [],
      // ignore: experimental_member_use
      isWeb: kIsWeb,
    );
    final doc = await converter.createDocument();
    return doc?.save();
  }

  Future<void> _exportPdf() async {
    if (_exporting) return;
    setState(() => _exporting = true);
    try {
      final bytes = await _buildPdfBytes();
      if (!mounted) return;
      if (bytes == null) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('Failed to generate PDF')));
        return;
      }
      await Printing.sharePdf(bytes: bytes, filename: '$_displayName.pdf');
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Export failed: $e')));
    } finally {
      if (mounted) setState(() => _exporting = false);
    }
  }

  Future<void> _printDocument() async {
    if (_exporting) return;
    setState(() => _exporting = true);
    try {
      final bytes = await _buildPdfBytes();
      if (!mounted) return;
      if (bytes == null) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Failed to generate PDF for printing')),
        );
        return;
      }
      await Printing.layoutPdf(onLayout: (_) => bytes);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Print failed: $e')));
    } finally {
      if (mounted) setState(() => _exporting = false);
    }
  }

  // ── Build ──────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final appearance = _appearance;
    return PopScope(
      canPop: !_dirty,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final leave = await _confirmDiscard(context);
        if (leave && context.mounted) Navigator.of(context).pop();
      },
      child: Scaffold(
        backgroundColor: appearance.pageColor,
        appBar: _buildAppBar(appearance),
        body: _buildBody(context, appearance),
      ),
    );
  }

  PreferredSizeWidget _buildAppBar(EditorPageAppearance appearance) {
    final titleText = _dirty ? '$_displayName •' : _displayName;
    return AppBar(
      backgroundColor: appearance.toolbarColor,
      foregroundColor: appearance.textColor,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      title: Text(titleText, overflow: TextOverflow.ellipsis),
      actions: [
        // Page appearance toggle (sun/moon) — #938
        IconButton(
          icon: Icon(appearance.icon),
          tooltip: appearance.tooltip,
          onPressed: _toggleAppearance,
        ),
        // Edit / Done toggle
        if (_isEditing) ...[
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
          TextButton(
            onPressed: _exitEditMode,
            child: Text('Done', style: TextStyle(color: appearance.textColor)),
          ),
        ] else
          TextButton.icon(
            icon: Icon(
              Icons.edit_outlined,
              size: 18,
              color: appearance.textColor,
            ),
            label: Text('Edit', style: TextStyle(color: appearance.textColor)),
            onPressed: _enterEditMode,
          ),
        // Overflow menu (export / print)
        if (_exporting)
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 12),
            child: SizedBox(
              width: 20,
              height: 20,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
          )
        else
          PopupMenuButton<_DocAction>(
            icon: Icon(Icons.more_vert, color: appearance.textColor),
            onSelected: (action) {
              switch (action) {
                case _DocAction.exportPdf:
                  _exportPdf();
                case _DocAction.print:
                  _printDocument();
              }
            },
            itemBuilder: (_) => const [
              PopupMenuItem(
                value: _DocAction.exportPdf,
                child: ListTile(
                  leading: Icon(Icons.picture_as_pdf_outlined),
                  title: Text('Export as PDF'),
                  contentPadding: EdgeInsets.zero,
                ),
              ),
              PopupMenuItem(
                value: _DocAction.print,
                child: ListTile(
                  leading: Icon(Icons.print_outlined),
                  title: Text('Print'),
                  contentPadding: EdgeInsets.zero,
                ),
              ),
            ],
          ),
      ],
    );
  }

  Widget _buildToolbar(
    BuildContext context,
    EditorPageAppearance appearance,
    ThemeData theme,
  ) {
    // Override the ambient theme so ALL toolbar children (icon buttons,
    // dropdown labels, undo/redo, dividers) pick up the right colors.
    final toolbarTheme = theme.copyWith(
      colorScheme: theme.colorScheme.copyWith(
        onSurface: appearance.textColor,
        onSurfaceVariant: appearance.mutedColor,
        surface: appearance.toolbarColor,
        surfaceContainerLow: appearance.toolbarColor,
        surfaceContainer: appearance.toolbarColor,
      ),
      iconTheme: IconThemeData(color: appearance.textColor, size: 20),
      // Dropdown / DropdownButton text color
      textTheme: theme.textTheme.apply(bodyColor: appearance.textColor),
    );

    return Theme(
      data: toolbarTheme,
      child: ColoredBox(
        color: appearance.toolbarColor,
        child: QuillSimpleToolbar(
          controller: _controller,
          config: QuillSimpleToolbarConfig(
            toolbarIconAlignment: WrapAlignment.start,
            buttonOptions: QuillSimpleToolbarButtonOptions(
              base: QuillToolbarBaseButtonOptions(
                iconTheme: QuillIconTheme(
                  iconButtonUnselectedData: IconButtonData(
                    color: appearance.textColor,
                  ),
                  iconButtonSelectedData: IconButtonData(
                    style: IconButton.styleFrom(
                      foregroundColor: theme.colorScheme.onPrimary,
                      backgroundColor: theme.colorScheme.primary,
                    ),
                  ),
                ),
              ),
              // Explicitly style the header dropdown text
              selectHeaderStyleDropdownButton:
                  QuillToolbarSelectHeaderStyleDropdownButtonOptions(
                    textStyle: TextStyle(
                      color: appearance.textColor,
                      fontSize: 13,
                    ),
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

  Widget _buildBody(BuildContext context, EditorPageAppearance appearance) {
    if (_loading) {
      return Center(
        child: CircularProgressIndicator(color: appearance.textColor),
      );
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline, size: 48, color: appearance.mutedColor),
            const SizedBox(height: 12),
            Text(
              _error!,
              textAlign: TextAlign.center,
              style: TextStyle(color: appearance.textColor),
            ),
            const SizedBox(height: 12),
            ElevatedButton(
              onPressed: _loadDocument,
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    final theme = Theme.of(context);
    final styles = _quillStylesForAppearance(appearance, theme);

    return Column(
      children: [
        // Toolbar — only visible in edit mode
        if (_isEditing) ...[
          _buildToolbar(context, appearance, theme),
          Divider(height: 1, color: appearance.dividerColor),
        ],
        // Document body — "paper" feel with centred max-width column
        Expanded(
          child: ColoredBox(
            color: appearance.pageColor,
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 760),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 32,
                    vertical: 40,
                  ),
                  child: QuillEditor.basic(
                    controller: _controller,
                    focusNode: _editorFocus,
                    scrollController: _scrollController,
                    config: QuillEditorConfig(
                      autoFocus: false,
                      expands: false,
                      padding: EdgeInsets.zero,
                      placeholder: _isEditing ? 'Start writing…' : '',
                      customStyles: styles,
                    ),
                  ),
                ),
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

enum _DocAction { exportPdf, print }
