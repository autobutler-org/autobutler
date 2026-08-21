import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_quill_to_pdf/flutter_quill_to_pdf.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:printing/printing.dart';
import 'package:quark/router.dart';
import 'package:quark/services/cirrus_service.dart';
import 'package:quark/theme/quark_theme.dart';
import 'package:quark/utils/file_browser_path_utils.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:quark_icons/quark_icons.dart';
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

// ── Find shortcut ─────────────────────────────────────────────────────────────

/// Swallows Ctrl/Cmd+F inside a [QuillEditor] and runs [onToggle] instead.
///
/// flutter_quill binds Ctrl/Cmd+F to its own modal search dialog
/// (`OpenSearchIntent`), which would open on top of our inline find bar. Its
/// `customShortcuts` hook can't override that — the package merges its defaults
/// *over* the caller's map — so this hangs off `onKeyPressed`, which runs on the
/// editor's own focus node, below the `Shortcuts` widget that dispatches the
/// intent. Returning a non-null result stops the event there (#1046).
///
/// Returns null for anything else so flutter_quill handles keys as usual.
KeyEventResult? quillFindKeyInterceptor(KeyEvent event, VoidCallback onToggle) {
  if (event.logicalKey != LogicalKeyboardKey.keyF) return null;
  final modified =
      HardwareKeyboard.instance.isControlPressed ||
      HardwareKeyboard.instance.isMetaPressed;
  if (!modified) return null;
  if (event is KeyDownEvent) onToggle();
  return KeyEventResult.handled;
}

/// Inline find bar for the document editor.
///
/// [QuillToolbarSearchDialog] owns the search itself (matching, hit list,
/// selection), but its default chrome is a [Dialog] whose close button calls
/// `Navigator.pop()` — which pops the editor route when the widget is rendered
/// inline instead of in a dialog. `childBuilder` swaps that chrome for this
/// strip, so closing the bar closes the bar (#1046).
class DocumentFindBar extends StatefulWidget {
  final QuillController controller;
  final VoidCallback onClose;

  const DocumentFindBar({
    required this.controller,
    required this.onClose,
    super.key,
  });

  @override
  State<DocumentFindBar> createState() => _DocumentFindBarState();
}

class _DocumentFindBarState extends State<DocumentFindBar> {
  final FocusNode _fieldFocus = FocusNode(debugLabel: 'find field');

  @override
  void initState() {
    super.initState();
    // `autofocus` is a no-op when something in the scope already holds focus,
    // which is the case when the bar is opened with Ctrl/Cmd+F from inside the
    // editor. Take focus outright once the field is in the tree.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _fieldFocus.requestFocus();
    });
  }

  @override
  void dispose() {
    _fieldFocus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return Container(
      color: cs.surfaceContainer,
      padding: const EdgeInsets.fromLTRB(12, 6, 6, 6),
      child: QuillToolbarSearchDialog(
        controller: widget.controller,
        childBuilder: (options) {
          final hits = options.offsets?.length ?? 0;
          final hasHits = hits > 0;
          return Row(
            children: [
              Expanded(
                child: TextField(
                  controller: options.textEditingController,
                  focusNode: _fieldFocus,
                  autofocus: true,
                  onChanged: options.onTextChanged,
                  onEditingComplete: options.onEditingComplete,
                  style: const TextStyle(fontSize: 13),
                  decoration: InputDecoration(
                    isDense: true,
                    hintText: 'Find in document',
                    border: const OutlineInputBorder(),
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 10,
                      vertical: 8,
                    ),
                    suffixText: options.text.isEmpty
                        ? null
                        : '${hasHits ? options.index + 1 : 0}/$hits',
                    suffixStyle: TextStyle(
                      fontSize: 12,
                      color: cs.onSurface.withValues(alpha: 0.6),
                    ),
                  ),
                ),
              ),
              IconButton(
                icon: const Icon(QuarkIcons.keyboard_arrow_up_rounded),
                tooltip: 'Previous match',
                onPressed: hasHits ? options.moveToPrevious : null,
              ),
              IconButton(
                icon: const Icon(QuarkIcons.expand_more_rounded),
                tooltip: 'Next match',
                onPressed: hasHits ? options.moveToNext : null,
              ),
              IconButton(
                icon: const Icon(QuarkIcons.close_rounded),
                tooltip: 'Close find bar (Esc)',
                onPressed: widget.onClose,
              ),
            ],
          );
        },
      ),
    );
  }
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

  // Auto-save
  static const _prefKeyAutoSave = 'document_editor_auto_save';
  static const _autoSaveDelay = Duration(seconds: 2);
  bool _autoSaveEnabled = true;
  Timer? _autoSaveTimer;

  // Word/char count (updated on doc change)
  int _wordCount = 0;

  // In-document find bar (#1046)
  bool _showFindBar = false;

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
      child: CallbackShortcuts(
        bindings: {
          // Ctrl+F / Cmd+F — toggle find bar (#1046). Only fires when focus is
          // outside the editor; the editor handles it in _handleEditorKey.
          const SingleActivator(LogicalKeyboardKey.keyF, control: true):
              _toggleFindBar,
          const SingleActivator(LogicalKeyboardKey.keyF, meta: true):
              _toggleFindBar,
          // Escape — close find bar when open
          const SingleActivator(LogicalKeyboardKey.escape): () {
            if (_showFindBar) setState(() => _showFindBar = false);
          },
        },
        child: Focus(
          // Take focus on open so Ctrl+F reaches CallbackShortcuts before the
          // user has clicked into the editor (which never autofocuses).
          autofocus: true,
          child: Scaffold(
            appBar: AppBar(
              title: Text(_dirty ? '$_displayName •' : _displayName),
              actions: _buildAppBarActions(context),
            ),
            body: _buildBody(context),
          ),
        ),
      ),
    );
  }

  List<Widget> _buildAppBarActions(BuildContext context) {
    return [
      // In-document find bar (#1046)
      IconButton(
        icon: Icon(_showFindBar ? Icons.close : QuarkIcons.search_rounded),
        tooltip: _showFindBar ? 'Close find bar' : 'Find in document (Ctrl+F)',
        onPressed: _toggleFindBar,
      ),
      // Settings shortcut
      IconButton(
        icon: const Icon(QuarkIcons.settings_outlined),
        tooltip: 'Settings',
        onPressed: () => context.go(AppRoutes.settings),
      ),
      const ThemeToggleButton(),
      // Auto-save toggle (only relevant in edit mode)
      if (!_isReadOnly)
        IconButton(
          icon: Icon(
            _autoSaveEnabled
                ? QuarkIcons.cloud_sync_outlined
                : QuarkIcons.cloud_off_outlined,
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
          icon: const Icon(QuarkIcons.more_horiz),
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
              icon: const Icon(QuarkIcons.save_outlined, size: 16),
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
                  icon: const Icon(QuarkIcons.edit_outlined, size: 16),
                  label: const Text('Edit'),
                )
              : OutlinedButton.icon(
                  onPressed: _exitEditMode,
                  icon: const Icon(QuarkIcons.check, size: 16),
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
            leading: const Icon(QuarkIcons.picture_as_pdf_outlined),
            title: const Text('Export as PDF'),
            onTap: () {
              Navigator.pop(context);
              _exportPdf();
            },
          ),
          ListTile(
            leading: const Icon(QuarkIcons.print_outlined),
            title: const Text('Print'),
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
            Icon(QuarkIcons.error_outline, size: 48, color: cs.error),
            const SizedBox(height: 12),
            Text(_error!, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton(onPressed: _loadDocument, child: const Text('Retry')),
          ],
        ),
      );
    }

    return Column(
      children: [
        if (!_isReadOnly) _buildToolbar(theme),
        // Find works in view mode too — the toolbar above is edit-only, the
        // find bar is not.
        if (_showFindBar) _buildFindBar(cs),
        Expanded(
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
        ),
      ],
    );
  }

  void _toggleFindBar() => setState(() => _showFindBar = !_showFindBar);

  KeyEventResult? _handleEditorKey(KeyEvent event, Node? node) =>
      quillFindKeyInterceptor(event, _toggleFindBar);

  Widget _buildFindBar(ColorScheme cs) =>
      DocumentFindBar(controller: _controller, onClose: _toggleFindBar);

  Widget _buildToolbar(ThemeData theme) {
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
          controller: _controller,
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
          ),
        ),
      ),
    );
  }

  Widget _buildPageFrame(ColorScheme cs) {
    // When dark page mode is active, use the app's dark theme ColorScheme;
    // when light, use the app's light theme ColorScheme. This keeps the
    // editor page consistent with the rest of the app's design language
    // while allowing the user to choose page brightness independently of
    // the global theme toggle.
    final pageCs = _editorDarkPage
        ? QuarkTheme.dark().colorScheme
        : QuarkTheme.light().colorScheme;

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
              // Keeps Quill's built-in search dialog from opening on top of the
              // inline find bar — see [quillFindKeyInterceptor].
              // ignore: experimental_member_use
              onKeyPressed: _handleEditorKey,
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
                  ? QuarkIcons.light_mode_outlined
                  : QuarkIcons.dark_mode_outlined,
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
            icon: QuarkIcons.edit_note,
            label: '$_wordCount words',
            color: muted,
          ),
          const SizedBox(width: 16),
          _statusItem(
            icon: QuarkIcons.lock_outline,
            label: 'Private',
            color: muted,
          ),
          const Spacer(),
          if (_isReadOnly)
            _statusItem(
              icon: QuarkIcons.visibility_outlined,
              label: 'Read-only',
              color: muted,
            )
          else if (_dirty)
            _statusItem(
              icon: QuarkIcons.circle,
              label: 'Unsaved',
              color: const Color(0xFFF59E0B),
            )
          else
            _statusItem(
              icon: QuarkIcons.check_circle_outline,
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
