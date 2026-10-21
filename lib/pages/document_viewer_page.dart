import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/controllers/document_viewer_controller.dart';
import 'package:quark/router.dart';
import 'package:quark/utils/document_viewer_files.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/file_browser_path_utils.dart';
import 'package:quark/utils/file_viewer_actions.dart';
import 'package:quark/widgets/document_viewer/document_viewer_body.dart';
import 'package:quark/widgets/document_viewer/document_viewer_conflict_dialog.dart';
import 'package:quark/widgets/layout/chrome_app_bar.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';
import 'package:url_launcher/url_launcher.dart';

/// The read-only document viewer at `/document/<path>` (#2987): a `.docx`,
/// `.docm`, `.odt` or `.rtf` read on the Quark and drawn as headings,
/// paragraphs, lists, tables and images.
///
/// Nothing the document holds is run or fetched from outside it. A link
/// opens only when it is `http`, `https` or `mailto`, and only after a tap.
/// "Open as Quark doc" writes the document beside itself as a `.qdoc` and
/// opens that in the editor.
///
/// Keys: `document_viewer_open_as_qdoc` and `document_viewer_download` on
/// the bar's actions.
class DocumentViewerPage extends StatefulWidget {
  /// The document's files-relative path.
  final String filePath;

  /// The device the document is on, empty for the internal drive.
  final String deviceSerial;

  /// The viewer's state; a test passes one with fake service calls.
  final DocumentViewerController? controller;

  /// Opens a link outside the Quark; a test passes a recorder.
  final void Function(Uri uri) openExternal;

  const DocumentViewerPage({
    required this.filePath,
    this.deviceSerial = '',
    this.controller,
    this.openExternal = _launchExternal,
    super.key,
  });

  static void _launchExternal(Uri uri) =>
      launchUrl(uri, mode: LaunchMode.externalApplication);

  @override
  State<DocumentViewerPage> createState() => _DocumentViewerPageState();
}

class _DocumentViewerPageState extends State<DocumentViewerPage> {
  late final DocumentViewerController _controller =
      widget.controller ??
      DocumentViewerController(
        path: widget.filePath,
        serial: serialOrNull(widget.deviceSerial),
      );

  String get _name => widget.filePath.split('/').last;

  @override
  void initState() {
    super.initState();
    _controller.load();
  }

  @override
  void dispose() {
    if (widget.controller == null) _controller.dispose();
    super.dispose();
  }

  void _openLink(String href) {
    final uri = documentLinkUri(href);
    if (uri != null) widget.openExternal(uri);
  }

  void _showMessage(String message) => ScaffoldMessenger.maybeOf(
    context,
  )?.showSnackBar(SnackBar(content: Text(message)));

  void _openQdoc(String path) => context.go(
    AppRoutes.docFile(
      path,
      serial: widget.deviceSerial,
      // Back from the editor returns here (#3186).
      from: GoRouterState.of(context).uri.toString(),
    ),
  );

  Future<void> _openAsQuarkDoc({bool overwrite = false}) async {
    try {
      final converted = await _controller.openAsQuarkDoc(overwrite: overwrite);
      if (!mounted) return;
      if (converted.warnings.isNotEmpty) {
        _showMessage(
          'Some of "$_name" is not in the Quark doc: '
          '${converted.warnings.join(' ')}',
        );
      }
      _openQdoc(converted.path);
    } on ApiException catch (e) {
      if (!mounted) return;
      if (e.statusCode == 409 && !overwrite) return _resolveConflict();
      _showMessage(Errors.message(e, 'convert the document'));
    } catch (e) {
      if (mounted) _showMessage(Errors.message(e, 'convert the document'));
    }
  }

  /// A `.qdoc` of that name is already there, and converting again would
  /// throw away whatever was done to it since, so the reader chooses.
  Future<void> _resolveConflict() async {
    final qdocPath = qdocPathFor(widget.filePath);
    final choice = await showDialog<DocumentConflictChoice>(
      context: context,
      builder: (context) => DocumentViewerConflictDialog(
        qdocName: qdocPath.split('/').last,
        documentName: _name,
      ),
    );
    if (!mounted) return;
    switch (choice) {
      case DocumentConflictChoice.replace:
        await _openAsQuarkDoc(overwrite: true);
      case DocumentConflictChoice.openExisting:
        _openQdoc(qdocPath);
      case DocumentConflictChoice.cancel || null:
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    // Nothing underneath is the usual case: Files opens the viewer at its own
    // URL, so leaving goes to the folder that holds the file (#1749).
    final canPop = context.canPop();
    void leave() => context.go(
      AppRoutes.containingFolder(widget.filePath, serial: widget.deviceSerial),
    );

    return PopScope(
      // A system back with nothing underneath would close the app.
      canPop: canPop,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && !canPop) leave();
      },
      child: ListenableBuilder(
        listenable: _controller,
        builder: (context, _) => Scaffold(
          appBar: ChromeAppBar(
            title: Text(_name),
            leading: BackButton(onPressed: canPop ? context.pop : leave),
            actions: [
              Row(
                mainAxisSize: MainAxisSize.min,
                spacing: QuarkTokens.of(context).spacingSm,
                children: [
                  QuarkBarChip(
                    key: const ValueKey('document_viewer_open_as_qdoc'),
                    icon: QuarkIcons.import_file,
                    label: 'Open as Quark doc',
                    tooltip: 'Save a copy as a Quark doc and edit it',
                    onPressed: _controller.isConverting
                        ? null
                        : _openAsQuarkDoc,
                  ),
                  QuarkBarIconButton(
                    key: const ValueKey('document_viewer_download'),
                    icon: QuarkIcons.download_outlined,
                    tooltip: 'Download',
                    onPressed: () => downloadViewedFile(
                      context,
                      path: widget.filePath,
                      serial: serialOrNull(widget.deviceSerial),
                      name: _name,
                    ),
                  ),
                  const AppThemeToggle(),
                ],
              ),
            ],
          ),
          body: SafeArea(
            top: false,
            child: DocumentViewerBody(
              controller: _controller,
              onLinkTap: _openLink,
            ),
          ),
        ),
      ),
    );
  }
}
