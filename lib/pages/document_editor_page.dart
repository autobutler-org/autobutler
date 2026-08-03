import 'dart:async';
import 'dart:convert';

import 'package:autobutler/router.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:autobutler_icons/autobutler_icons.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_quill_to_pdf/flutter_quill_to_pdf.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:printing/printing.dart';
import 'package:autobutler/theme/autobutler_theme.dart';
import 'package:autobutler/widgets/editor/editor_sidebar_toolbar.dart';
import 'package:autobutler/widgets/layout/theme_toggle_button.dart';
import 'package:shared_preferences/shared_preferences.dart';

// ── Quill styles ──────────────────────────────────────────────────────────────

DefaultStyles _quillStyles(ColorScheme cs) {
  final fg = cs.onSurface;
  final muted = cs.onSurface.withValues(alpha: 0.5);
  final codeBg = cs.surfaceContainerHighest;
  final codeColor = cs.secondary;
  final outline = cs.outline;

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
        border: Border(left: BorderSide(color: cs.primary, width: 3)),
      ),
    ),
    inlineCode: InlineCodeStyle(
      style: TextStyle(
        fontFamily: 'monospace',
        fontSize: 13,
        color: codeColor,
        backgroundColor: codeBg,
      ),
      backgroundColor: codeBg,
      radius: const Radius.circular(4),
    ),
    code: DefaultTextBlockStyle(
      TextStyle(fontFamily: 'monospace', fontSize: 13, color: codeColor),
      const HorizontalSpacing(16, 16),
      const VerticalSpacing(8, 8),
      VerticalSpacing.zero,
      BoxDecoration(
        color: codeBg,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: outline),
      ),
    ),
    color: fg,
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

class DocumentEditorPage extends StatefulWidget {
  final String filePath;
  final String deviceSerial;
  final String? overlayTargetRoute;
  final String? overlayCloseRoute;

  const DocumentEditorPage({
    required this.filePath,
    this.deviceSerial = '',
    this.overlayTargetRoute,
    this.overlayCloseRoute,
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
  bool _routeMovedExternally = false;

  // Read-only / edit mode (#939)
  bool _isReadOnly = true;

  // Edit button glow hint (#940)
  bool _hintEditButton = false;
  Timer? _hintTimer;

  // Dark page mode (#938)
  static const _prefKeyDarkPage = 'document_editor_dark_page';
  bool _editorDarkPage = false;

  // Toolbar mode (#1327)
  static const _prefKeyToolbarMode = 'document_editor_toolbar_mode';
  EditorToolbarMode _toolbarMode = EditorToolbarMode.top;

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
    _controller = QuillController.basic()..readOnly = _isReadOnly;
    if (widget.overlayTargetRoute != null) {
      router.routeInformationProvider.addListener(_handleOverlayRouteChange);
    }
    _loadPrefs();
    _loadDocument();
  }

  @override
  void dispose() {
    _autoSaveTimer?.cancel();
    _hintTimer?.cancel();
    if (widget.overlayTargetRoute != null) {
      router.routeInformationProvider.removeListener(_handleOverlayRouteChange);
    }
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

  String _currentRoute() =>
      router.routeInformationProvider.value.uri.toString();

  Future<void> _handleOverlayRouteChange() async {
    final targetRoute = widget.overlayTargetRoute;
    if (targetRoute == null || !mounted || _currentRoute() == targetRoute) {
      return;
    }

    _routeMovedExternally = true;
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (!mounted) {
        return;
      }
      await Navigator.of(context).maybePop();
    });
  }

  void _restoreOverlayCloseRoute() {
    final targetRoute = widget.overlayTargetRoute;
    final closeRoute = widget.overlayCloseRoute;
    if (targetRoute == null || closeRoute == null) {
      return;
    }

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!_routeMovedExternally && _currentRoute() == targetRoute) {
        router.go(closeRoute);
      }
    });
  }

  Future<void> _loadPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    setState(() {
      _autoSaveEnabled = prefs.getBool(_prefKeyAutoSave) ?? true;
      _editorDarkPage = prefs.getBool(_prefKeyDarkPage) ?? false;
      _toolbarMode = (prefs.getString(_prefKeyToolbarMode) == 'sidebar')
          ? EditorToolbarMode.sidebar
          : EditorToolbarMode.top;
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

  Future<void> _setToolbarMode(EditorToolbarMode mode) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(
      _prefKeyToolbarMode,
      mode == EditorToolbarMode.sidebar ? 'sidebar' : 'top',
    );
    if (!mounted) return;
    setState(() => _toolbarMode = mode);
  }

  // ── Read-only / edit toggle (#939) ────────────────────────────────────────

  void _enterEditMode() {
    _controller.readOnly = false;
    setState(() => _isReadOnly = false);
    _focusEditor();
  }

  Future<void> _exitEditMode() async {
    _controller.readOnly = true;
    setState(() => _isReadOnly = true);
    _editorFocus.unfocus();
    if (_dirty) {
      _autoSaveTimer?.cancel();
      await _autoSaveSilently();
    }
  }

  // ── Edit button glow hint (#940) ──────────────────────────────────────────

  void _onEditorTappedInReadOnly() {
    if (!_isReadOnly) return;
    _hintTimer?.cancel();
    setState(() => _hintEditButton = true);
    _hintTimer = Timer(const Duration(milliseconds: 1500), () {
      if (mounted) setState(() => _hintEditButton = false);
    });
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
      controller.readOnly = _isReadOnly;
      setState(() {
        _controller = controller;
        _loading = false;
        _dirty = false;
        _wordCount = _countWords(doc.toPlainText());
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
    // Do not mark dirty or schedule auto-save in read-only mode
    if (_isReadOnly) return;
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
        if (didPop) {
          _restoreOverlayCloseRoute();
          return;
        }
        final leave = await _confirmDiscard(context);
        if (!context.mounted) {
          return;
        }
        if (leave) {
          Navigator.of(context).pop();
          return;
        }
        if (_routeMovedExternally && widget.overlayTargetRoute != null) {
          _routeMovedExternally = false;
          router.go(widget.overlayTargetRoute!);
        }
      },
      child: Scaffold(
        appBar: AppBar(
          title: Text(_dirty ? '$_displayName •' : _displayName),
          actions: _buildAppBarActions(context),
        ),
        body: _buildBody(context),
      ),
    );
  }

  List<Widget> _buildAppBarActions(BuildContext context) {
    return [
      // In-document search (TODO: wire to find-in-doc, see #1046)
      IconButton(
        icon: const Icon(AutobutlerIcons.search_rounded),
        tooltip: 'Search in document',
        onPressed: () {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('In-document search coming soon'),
              duration: Duration(seconds: 2),
            ),
          );
        },
      ),
      // Settings shortcut
      IconButton(
        icon: const Icon(AutobutlerIcons.settings_outlined),
        tooltip: 'Settings',
        onPressed: () => context.go(AppRoutes.settings),
      ),
      const ThemeToggleButton(),
      // Auto-save toggle (only relevant in edit mode)
      if (!_isReadOnly)
        IconButton(
          icon: Icon(
            _autoSaveEnabled
                ? AutobutlerIcons.cloud_sync_outlined
                : AutobutlerIcons.cloud_off_outlined,
          ),
          tooltip: _autoSaveEnabled ? 'Auto-save on' : 'Auto-save off',
          onPressed: () => _setAutoSaveEnabled(!_autoSaveEnabled),
        ),
      // Overflow menu
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
        IconButton(
          icon: const Icon(AutobutlerIcons.more_horiz),
          tooltip: 'More options',
          onPressed: () => _showOverflowMenu(context),
        ),
      // Save button (only in edit mode)
      if (!_isReadOnly)
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
          Padding(
            padding: const EdgeInsets.only(right: 4),
            child: FilledButton.icon(
              onPressed: _dirty ? _saveDocument : null,
              icon: const Icon(AutobutlerIcons.save_outlined, size: 16),
              label: const Text('Save'),
            ),
          ),
      // Edit / Done toggle button (#939) with glow hint (#940)
      Padding(
        padding: const EdgeInsets.only(right: 8),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          decoration: _hintEditButton
              ? BoxDecoration(
                  borderRadius: BorderRadius.circular(8),
                  boxShadow: [
                    BoxShadow(
                      color: Theme.of(
                        context,
                      ).colorScheme.primary.withValues(alpha: 0.6),
                      blurRadius: 12,
                      spreadRadius: 2,
                    ),
                  ],
                )
              : null,
          child: _isReadOnly
              ? FilledButton.icon(
                  onPressed: _enterEditMode,
                  icon: const Icon(AutobutlerIcons.edit_outlined, size: 16),
                  label: const Text('Edit'),
                )
              : OutlinedButton.icon(
                  onPressed: _exitEditMode,
                  icon: const Icon(AutobutlerIcons.check, size: 16),
                  label: const Text('Done'),
                ),
        ),
      ),
    ];
  }

  void _showOverflowMenu(BuildContext context) {
    showModalBottomSheet<void>(
      context: context,
      builder: (_) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const SizedBox(height: 8),
          Container(
            width: 36,
            height: 4,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.outline,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          const SizedBox(height: 16),
          ListTile(
            leading: const Icon(AutobutlerIcons.picture_as_pdf_outlined),
            title: const Text('Export as PDF'),
            onTap: () {
              Navigator.pop(context);
              _exportPdf();
            },
          ),
          ListTile(
            leading: const Icon(AutobutlerIcons.print_outlined),
            title: const Text('Print'),
            onTap: () {
              Navigator.pop(context);
              _printDocument();
            },
          ),
          const Divider(),
          ListTile(
            leading: Icon(
              _toolbarMode == EditorToolbarMode.sidebar
                  ? Icons.view_sidebar
                  : Icons.view_stream,
            ),
            title: Text(
              _toolbarMode == EditorToolbarMode.sidebar
                  ? 'Switch to top toolbar'
                  : 'Switch to sidebar toolbar',
            ),
            onTap: () {
              Navigator.pop(context);
              _setToolbarMode(
                _toolbarMode == EditorToolbarMode.sidebar
                    ? EditorToolbarMode.top
                    : EditorToolbarMode.sidebar,
              );
            },
          ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(AutobutlerIcons.error_outline, size: 48, color: cs.error),
            const SizedBox(height: 12),
            Text(_error!, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton(onPressed: _loadDocument, child: const Text('Retry')),
          ],
        ),
      );
    }

    final editorBody = Expanded(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 900),
          child: Column(
            children: [
              Expanded(child: _buildPageFrame(cs)),
              _buildStatusBar(cs),
            ],
          ),
        ),
      ),
    );

    if (_isReadOnly) {
      return Column(children: [editorBody]);
    }

    return EditorToolbarLayout(
      mode: _toolbarMode,
      toolbar: _buildToolbar(theme),
      sidebarToolbar: _buildSidebarToolbar(theme),
      child: Column(children: [editorBody]),
    );
  }

  Widget _buildToolbar(ThemeData theme) {
    final cs = theme.colorScheme;
    final tt = _toolbarTheme(theme);
    return Theme(
      data: tt,
      child: Container(
        decoration: BoxDecoration(
          color: cs.surfaceContainer,
          border: Border(bottom: BorderSide(color: cs.outline)),
        ),
        child: QuillSimpleToolbar(
          controller: _controller,
          config: _toolbarConfig(cs),
        ),
      ),
    );
  }

  ThemeData _toolbarTheme(ThemeData theme) {
    final cs = theme.colorScheme;
    return theme.copyWith(
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
  }

  QuillSimpleToolbarConfig _toolbarConfig(ColorScheme cs) {
    return QuillSimpleToolbarConfig(
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
          customOnPressedCallback: _pickBackgroundColor,
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
    );
  }

  Widget _buildSidebarToolbar(ThemeData theme) {
    final tt = _toolbarTheme(theme);
    return EditorSidebarToolbar(
      controller: _controller,
      toolbarTheme: tt,
      config: _toolbarConfig(theme.colorScheme),
    );
  }

  Widget _buildPageFrame(ColorScheme cs) {
    // When dark page mode is active, use the app's dark theme ColorScheme;
    // when light, use the app's light theme ColorScheme. This keeps the
    // editor page consistent with the rest of the app's design language
    // while allowing the user to choose page brightness independently of
    // the global theme toggle.
    final pageCs = _editorDarkPage
        ? AutobutlerTheme.dark().colorScheme
        : AutobutlerTheme.light().colorScheme;

    return Padding(
      padding: const EdgeInsets.fromLTRB(24, 16, 24, 0),
      child: Container(
        decoration: BoxDecoration(
          color: pageCs.surface,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: pageCs.outline),
        ),
        padding: const EdgeInsets.fromLTRB(40, 24, 40, 24),
        child: GestureDetector(
          onTap: _onEditorTappedInReadOnly,
          behavior: HitTestBehavior.translucent,
          child: QuillEditor.basic(
            controller: _controller,
            focusNode: _editorFocus,
            scrollController: _scrollController,
            config: QuillEditorConfig(
              autoFocus: false,
              expands: false,
              padding: EdgeInsets.zero,
              placeholder: 'Start writing…',
              customStyles: _quillStyles(pageCs),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildStatusBar(ColorScheme cs) {
    final muted = cs.onSurface.withValues(alpha: 0.5);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
      child: Row(
        children: [
          // Page brightness toggle (#938) — bottom-left, near the page
          IconButton(
            icon: Icon(
              _editorDarkPage
                  ? AutobutlerIcons.light_mode_outlined
                  : AutobutlerIcons.dark_mode_outlined,
              size: 14,
            ),
            tooltip: _editorDarkPage
                ? 'Switch to light page'
                : 'Switch to dark page',
            style: IconButton.styleFrom(
              padding: EdgeInsets.zero,
              minimumSize: const Size(24, 24),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
            color: muted,
            onPressed: () async {
              final prefs = await SharedPreferences.getInstance();
              setState(() => _editorDarkPage = !_editorDarkPage);
              await prefs.setBool(_prefKeyDarkPage, _editorDarkPage);
            },
          ),
          const SizedBox(width: 8),
          _statusItem(
            icon: AutobutlerIcons.edit_note,
            label: '$_wordCount words',
            color: muted,
          ),
          const SizedBox(width: 16),
          _statusItem(
            icon: AutobutlerIcons.lock_outline,
            label: 'Private',
            color: muted,
          ),
          const Spacer(),
          if (_isReadOnly)
            _statusItem(
              icon: AutobutlerIcons.visibility_outlined,
              label: 'Read-only',
              color: muted,
            )
          else if (_dirty)
            _statusItem(
              icon: AutobutlerIcons.circle,
              label: 'Unsaved',
              color: const Color(0xFFF59E0B),
            )
          else
            _statusItem(
              icon: AutobutlerIcons.check_circle_outline,
              label: 'Saved',
              color: const Color(0xFF10B981),
            ),
        ],
      ),
    );
  }

  Widget _statusItem({
    required IconData icon,
    required String label,
    required Color color,
  }) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: color),
        const SizedBox(width: 5),
        Text(
          label,
          style: TextStyle(
            fontSize: 12,
            color: Theme.of(
              context,
            ).colorScheme.onSurface.withValues(alpha: 0.5),
          ),
        ),
      ],
    );
  }

  Future<void> _pickBackgroundColor(
    QuillController controller,
    bool isBackground,
  ) async {
    if (!mounted) return;
    // Save selection before the dialog steals focus (web loses it otherwise).
    final saved = controller.selection;

    final picked = await showDialog<Color?>(
      context: context,
      builder: (_) => const _HighlightPickerDialog(),
    );

    if (!mounted) return;
    if (picked == null) return; // cancelled

    // Restore selection in case web dropped it while the dialog was open.
    if (saved.isValid) {
      controller.updateSelection(saved, ChangeSource.local);
    }

    // Always clear first so we never visually stack two background colors.
    controller.formatSelection(const BackgroundAttribute(null));
    if (picked != Colors.transparent) {
      controller.formatSelection(BackgroundAttribute(_colorToHex(picked)));
    }
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

// Produces "#AARRGGBB" matching flutter_quill's BackgroundAttribute storage format.
String _colorToHex(Color color) {
  int ch(double v) => (v * 255).round() & 0xFF;
  return '#'
          '${ch(color.a).toRadixString(16).padLeft(2, '0')}'
          '${ch(color.r).toRadixString(16).padLeft(2, '0')}'
          '${ch(color.g).toRadixString(16).padLeft(2, '0')}'
          '${ch(color.b).toRadixString(16).padLeft(2, '0')}'
      .toUpperCase();
}

class _HighlightPickerDialog extends StatelessWidget {
  const _HighlightPickerDialog();

  static const _colors = <Color>[
    Color(0xFFFFEB3B), // Yellow
    Color(0xFF8BC34A), // Green
    Color(0xFF4FC3F7), // Blue
    Color(0xFFF48FB1), // Pink
    Color(0xFFCE93D8), // Lavender
    Color(0xFFFFCC80), // Orange
    Color(0xFFEF9A9A), // Red
    Color(0xFF80DEEA), // Cyan
  ];

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Highlight color'),
      content: Wrap(
        spacing: 10,
        runSpacing: 10,
        children: _colors.map((c) {
          return GestureDetector(
            onTap: () => Navigator.of(context).pop(c),
            child: Container(
              width: 36,
              height: 36,
              decoration: BoxDecoration(
                color: c,
                shape: BoxShape.circle,
                border: Border.all(color: Colors.black26, width: 1.5),
              ),
            ),
          );
        }).toList(),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(Colors.transparent),
          child: const Text('Clear'),
        ),
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
      ],
    );
  }
}
