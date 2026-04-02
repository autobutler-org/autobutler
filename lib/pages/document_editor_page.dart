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

// ── Design tokens (from Banani spec) ─────────────────────────────────────────

class _EditorColors {
  _EditorColors._();

  // Page background gradient stops
  static const gradientBase = Color(0xFF0A0E1A);

  // Surfaces
  static const toolbarBg = Color(0xFF080C18); // rgba(8,12,24,0.9)
  static const pageBg = Color(0xFF08090E); // rgba(8,10,18,0.85) — page frame
  static const groupBg = Color(0x0AFFFFFF); // rgba(255,255,255,0.04)

  // Borders
  static const border = Color(0x0FFFFFFF); // rgba(255,255,255,0.06)
  static const pageBorder = Color(0x0AFFFFFF); // rgba(255,255,255,0.04)

  // Text
  static const foreground = Color(0xFFE2E8F0); // card-foreground
  static const muted = Color(0xFF64748B); // muted-foreground
  static const secondary = Color(0xFF94A3B8); // secondary-foreground

  // Accents
  static const primary = Color(0xFF0EA5E9);
  static const primaryFg = Color(0xFFFFFFFF);
  // Ghost button
  static const ghostBg = Color(0x08FFFFFF); // rgba(255,255,255,0.03)
  static const ghostBorder = Color(0x14FFFFFF); // rgba(255,255,255,0.08)
}

// ── Quill styles ──────────────────────────────────────────────────────────────

DefaultStyles _quillStyles(ThemeData theme) {
  const fg = _EditorColors.foreground;
  const muted = _EditorColors.muted;
  const codeBg = Color(0xFF111827);
  final primary = theme.colorScheme.primary;

  TextStyle base([double size = 14]) =>
      TextStyle(color: fg, fontSize: size, height: 1.7);

  return DefaultStyles(
    paragraph: DefaultTextBlockStyle(
      base(),
      HorizontalSpacing.zero,
      VerticalSpacing.zero,
      VerticalSpacing.zero,
      null,
    ),
    h1: DefaultTextBlockStyle(
      base(26).copyWith(fontWeight: FontWeight.w600, color: fg),
      HorizontalSpacing.zero,
      const VerticalSpacing(20, 6),
      VerticalSpacing.zero,
      null,
    ),
    h2: DefaultTextBlockStyle(
      base(20).copyWith(fontWeight: FontWeight.w600, color: fg),
      HorizontalSpacing.zero,
      const VerticalSpacing(16, 4),
      VerticalSpacing.zero,
      null,
    ),
    h3: DefaultTextBlockStyle(
      base(16).copyWith(fontWeight: FontWeight.w600, color: fg),
      HorizontalSpacing.zero,
      const VerticalSpacing(12, 4),
      VerticalSpacing.zero,
      null,
    ),
    placeHolder: DefaultTextBlockStyle(
      base().copyWith(color: muted),
      HorizontalSpacing.zero,
      VerticalSpacing.zero,
      VerticalSpacing.zero,
      null,
    ),
    quote: DefaultTextBlockStyle(
      base().copyWith(color: muted, fontStyle: FontStyle.italic),
      const HorizontalSpacing(16, 0),
      const VerticalSpacing(6, 6),
      VerticalSpacing.zero,
      BoxDecoration(
        border: Border(left: BorderSide(color: primary, width: 3)),
      ),
    ),
    inlineCode: InlineCodeStyle(
      style: TextStyle(
        fontFamily: 'monospace',
        fontSize: 13,
        color: const Color(0xFFA78BFA),
        backgroundColor: codeBg,
      ),
      backgroundColor: codeBg,
      radius: const Radius.circular(4),
    ),
    code: DefaultTextBlockStyle(
      const TextStyle(
        fontFamily: 'monospace',
        fontSize: 13,
        color: Color(0xFFA78BFA),
      ),
      const HorizontalSpacing(16, 16),
      const VerticalSpacing(8, 8),
      VerticalSpacing.zero,
      BoxDecoration(
        color: codeBg,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _EditorColors.border),
      ),
    ),
    color: fg,
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

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

  // Auto-save
  static const _prefKeyAutoSave = 'document_editor_auto_save';
  static const _autoSaveDelay = Duration(seconds: 2);
  bool _autoSaveEnabled = true;
  Timer? _autoSaveTimer;

  // Word/char count (updated on doc change)
  int _wordCount = 0;

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

  // ── Helpers ────────────────────────────────────────────────────────────────

  String _nameFromPath(String path) {
    final name = path.split('/').last;
    return name.endsWith('.abdoc') ? name.substring(0, name.length - 6) : name;
  }

  Future<void> _loadPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    setState(() => _autoSaveEnabled = prefs.getBool(_prefKeyAutoSave) ?? true);
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

  // ── Document ───────────────────────────────────────────────────────────────

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
      );
      setState(() {
        _controller = controller;
        _loading = false;
        _dirty = false;
        _wordCount = _countWords(doc.toPlainText());
      });
      _controller.addListener(_onDocumentChanged);
      _focusEditor();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = 'Failed to load document: $e';
      });
    }
  }

  void _focusEditor() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      FocusScope.of(context).unfocus();
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) _editorFocus.requestFocus();
      });
    });
  }

  int _countWords(String text) {
    final trimmed = text.trim();
    if (trimmed.isEmpty) return 0;
    return trimmed.split(RegExp(r'\s+')).length;
  }

  void _onDocumentChanged() {
    if (!mounted) return;
    final wc = _countWords(_controller.document.toPlainText());
    if (!_dirty || wc != _wordCount) {
      setState(() {
        _dirty = true;
        _wordCount = wc;
      });
    }
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
      final bytes = utf8.encode(jsonEncode({'ops': ops}));
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
    return PopScope(
      canPop: !_dirty,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final leave = await _confirmDiscard(context);
        if (leave && context.mounted) Navigator.of(context).pop();
      },
      child: Scaffold(
        backgroundColor: _EditorColors.gradientBase,
        appBar: _buildAppBar(),
        body: _buildBody(context),
      ),
    );
  }

  PreferredSizeWidget _buildAppBar() {
    final saveLabel = _dirty ? '$_displayName •' : _displayName;
    final savedLabel = _dirty ? 'Unsaved changes' : 'Auto-saved just now';

    return PreferredSize(
      preferredSize: const Size.fromHeight(64),
      child: Container(
        decoration: const BoxDecoration(
          color: _EditorColors.toolbarBg,
          border: Border(bottom: BorderSide(color: _EditorColors.border)),
        ),
        child: SafeArea(
          bottom: false,
          child: SizedBox(
            height: 64,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Row(
                children: [
                  // Back
                  IconButton(
                    icon: const Icon(
                      Icons.arrow_back,
                      color: _EditorColors.secondary,
                      size: 20,
                    ),
                    onPressed: () => Navigator.of(context).maybePop(),
                    tooltip: 'Back',
                  ),
                  const SizedBox(width: 8),
                  // Doc title + saved status
                  Expanded(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          saveLabel,
                          style: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w500,
                            color: _EditorColors.foreground,
                          ),
                          overflow: TextOverflow.ellipsis,
                        ),
                        Text(
                          savedLabel,
                          style: const TextStyle(
                            fontSize: 12,
                            color: _EditorColors.muted,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  // Auto-save toggle
                  _GhostButton(
                    icon: _autoSaveEnabled
                        ? Icons.cloud_sync_outlined
                        : Icons.cloud_off_outlined,
                    tooltip: _autoSaveEnabled
                        ? 'Auto-save on'
                        : 'Auto-save off',
                    onTap: () => _setAutoSaveEnabled(!_autoSaveEnabled),
                  ),
                  const SizedBox(width: 6),
                  // Overflow menu
                  if (_exporting)
                    const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: _EditorColors.secondary,
                      ),
                    )
                  else
                    _GhostButton(
                      icon: Icons.more_horiz,
                      tooltip: 'More options',
                      onTap: () => _showOverflowMenu(context),
                    ),
                  const SizedBox(width: 6),
                  // Save button
                  if (_saving)
                    const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: _EditorColors.secondary,
                      ),
                    )
                  else
                    _PrimaryButton(
                      icon: Icons.save_outlined,
                      label: 'Save',
                      enabled: _dirty,
                      onTap: _saveDocument,
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  void _showOverflowMenu(BuildContext context) {
    showModalBottomSheet<void>(
      context: context,
      backgroundColor: const Color(0xFF0F1629),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
        side: BorderSide(color: _EditorColors.border),
      ),
      builder: (_) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const SizedBox(height: 8),
          Container(
            width: 36,
            height: 4,
            decoration: BoxDecoration(
              color: _EditorColors.border,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          const SizedBox(height: 16),
          ListTile(
            leading: const Icon(
              Icons.picture_as_pdf_outlined,
              color: _EditorColors.secondary,
            ),
            title: const Text(
              'Export as PDF',
              style: TextStyle(color: _EditorColors.foreground),
            ),
            onTap: () {
              Navigator.pop(context);
              _exportPdf();
            },
          ),
          ListTile(
            leading: const Icon(
              Icons.print_outlined,
              color: _EditorColors.secondary,
            ),
            title: const Text(
              'Print',
              style: TextStyle(color: _EditorColors.foreground),
            ),
            onTap: () {
              Navigator.pop(context);
              _printDocument();
            },
          ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context) {
    if (_loading) {
      return const Center(
        child: CircularProgressIndicator(color: _EditorColors.secondary),
      );
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(
              Icons.error_outline,
              size: 48,
              color: _EditorColors.muted,
            ),
            const SizedBox(height: 12),
            Text(
              _error!,
              textAlign: TextAlign.center,
              style: const TextStyle(color: _EditorColors.foreground),
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

    return Column(
      children: [
        // ── Quill toolbar ──
        _buildToolbar(theme),
        // ── Editor body ──
        Expanded(
          child: Container(
            // Subtle purple+blue gradient to match Banani background
            decoration: const BoxDecoration(
              gradient: RadialGradient(
                center: Alignment(-0.8, -0.8),
                radius: 1.5,
                colors: [Color(0x2E7854FF), _EditorColors.gradientBase],
              ),
            ),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 900),
                child: Column(
                  children: [
                    Expanded(child: _buildPageFrame(theme)),
                    _buildStatusBar(),
                  ],
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildToolbar(ThemeData theme) {
    final toolbarTheme = theme.copyWith(
      colorScheme: theme.colorScheme.copyWith(
        onSurface: _EditorColors.secondary,
        onSurfaceVariant: _EditorColors.muted,
        surface: _EditorColors.toolbarBg,
        surfaceContainerLow: _EditorColors.toolbarBg,
        surfaceContainer: _EditorColors.toolbarBg,
      ),
      iconTheme: const IconThemeData(color: _EditorColors.secondary, size: 16),
      textTheme: theme.textTheme.apply(
        bodyColor: _EditorColors.secondary,
        displayColor: _EditorColors.secondary,
      ),
    );

    return Theme(
      data: toolbarTheme,
      child: Container(
        decoration: const BoxDecoration(
          color: _EditorColors.toolbarBg,
          border: Border(bottom: BorderSide(color: _EditorColors.border)),
        ),
        child: QuillSimpleToolbar(
          controller: _controller,
          config: QuillSimpleToolbarConfig(
            toolbarIconAlignment: WrapAlignment.center,
            buttonOptions: QuillSimpleToolbarButtonOptions(
              base: QuillToolbarBaseButtonOptions(
                iconTheme: QuillIconTheme(
                  iconButtonUnselectedData: IconButtonData(
                    color: _EditorColors.secondary,
                    style: IconButton.styleFrom(
                      backgroundColor: _EditorColors.groupBg,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6),
                      ),
                    ),
                  ),
                  iconButtonSelectedData: IconButtonData(
                    style: IconButton.styleFrom(
                      foregroundColor: theme.colorScheme.onPrimary,
                      backgroundColor: theme.colorScheme.primary,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6),
                      ),
                    ),
                  ),
                ),
              ),
              selectHeaderStyleDropdownButton:
                  QuillToolbarSelectHeaderStyleDropdownButtonOptions(
                    textStyle: const TextStyle(
                      color: _EditorColors.secondary,
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

  Widget _buildPageFrame(ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(24, 16, 24, 0),
      child: Container(
        decoration: BoxDecoration(
          color: _EditorColors.pageBg,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: _EditorColors.pageBorder),
        ),
        padding: const EdgeInsets.fromLTRB(40, 24, 40, 24),
        child: QuillEditor.basic(
          controller: _controller,
          focusNode: _editorFocus,
          scrollController: _scrollController,
          config: QuillEditorConfig(
            autoFocus: true,
            expands: false,
            padding: EdgeInsets.zero,
            placeholder: 'Start writing…',
            customStyles: _quillStyles(theme),
          ),
        ),
      ),
    );
  }

  Widget _buildStatusBar() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
      child: Row(
        children: [
          _StatusItem(icon: Icons.edit_note, label: '$_wordCount words'),
          const SizedBox(width: 16),
          _StatusItem(icon: Icons.lock_outline, label: 'Private'),
          const Spacer(),
          if (_dirty)
            const _StatusItem(
              icon: Icons.circle,
              label: 'Unsaved',
              iconColor: Color(0xFFF59E0B),
            )
          else
            const _StatusItem(
              icon: Icons.check_circle_outline,
              label: 'Saved',
              iconColor: Color(0xFF10B981),
            ),
        ],
      ),
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

// ── Small reusable widgets ─────────────────────────────────────────────────────

class _GhostButton extends StatelessWidget {
  const _GhostButton({
    required this.icon,
    required this.tooltip,
    required this.onTap,
  });

  final IconData icon;
  final String tooltip;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
          decoration: BoxDecoration(
            color: _EditorColors.ghostBg,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: _EditorColors.ghostBorder),
          ),
          child: Icon(icon, size: 18, color: _EditorColors.secondary),
        ),
      ),
    );
  }
}

class _PrimaryButton extends StatelessWidget {
  const _PrimaryButton({
    required this.icon,
    required this.label,
    required this.onTap,
    this.enabled = true,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return Opacity(
      opacity: enabled ? 1.0 : 0.4,
      child: InkWell(
        onTap: enabled ? onTap : null,
        borderRadius: BorderRadius.circular(10),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          decoration: BoxDecoration(
            color: _EditorColors.primary,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: const Color(0x1AFFFFFF)),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 16, color: _EditorColors.primaryFg),
              const SizedBox(width: 6),
              Text(
                label,
                style: const TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: _EditorColors.primaryFg,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _StatusItem extends StatelessWidget {
  const _StatusItem({
    required this.icon,
    required this.label,
    this.iconColor = _EditorColors.muted,
  });

  final IconData icon;
  final String label;
  final Color iconColor;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: iconColor),
        const SizedBox(width: 5),
        Text(
          label,
          style: const TextStyle(fontSize: 12, color: _EditorColors.muted),
        ),
      ],
    );
  }
}
