import 'package:flutter/foundation.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// Reads a document's block stream; see [PreviewsService.document].
typedef LoadDocumentFn =
    Future<QuarkDocument> Function(String path, {String? serial});

/// Writes a document as a `.qdoc`; see [PreviewsService.convertToQdoc].
typedef ConvertDocumentFn =
    Future<DocumentConversion> Function(
      String path, {
      String? serial,
      bool overwrite,
    });

/// State behind the document viewer for the `.docx`, `.docm`, `.odt` or
/// `.rtf` at [path] (#2987): the parsed blocks the Quark sent, whether they
/// are still loading or failed to, and whether "Open as Quark doc" is
/// running.
///
/// The Quark reads the document in place, so nothing here ever holds the
/// file itself, only the blocks it was converted into.
class DocumentViewerController extends ChangeNotifier {
  DocumentViewerController({
    required this.path,
    this.serial,
    this.loadDocument = PreviewsService.document,
    this.convertDocument = PreviewsService.convertToQdoc,
  });

  /// The document's files-relative path.
  final String path;

  /// The device the document is on, null for the internal drive.
  final String? serial;

  final LoadDocumentFn loadDocument;
  final ConvertDocumentFn convertDocument;

  QuarkDocument _document = const QuarkDocument.empty();
  bool _isLoading = true;
  Object? _error;
  bool _isConverting = false;
  int _generation = 0;
  bool _disposed = false;

  /// The document as last read; empty before the first read lands.
  QuarkDocument get document => _document;

  /// Whether the first read is still running. A later [load] keeps the
  /// document already shown.
  bool get isLoading => _isLoading;

  /// What the last read failed with, null once it has worked. The page turns
  /// it into a sentence.
  Object? get error => _error;

  /// Whether "Open as Quark doc" is waiting on the Quark.
  bool get isConverting => _isConverting;

  /// Reads the document, replacing what was read before once the new read
  /// lands. A read that a newer one overtook is dropped.
  Future<void> load() async {
    final generation = ++_generation;
    _error = null;
    _notify();
    try {
      final document = await loadDocument(path, serial: serial);
      if (generation != _generation) return;
      _document = document;
    } catch (e) {
      if (generation != _generation) return;
      _error = e;
    }
    _isLoading = false;
    _notify();
  }

  /// Writes the document beside itself as a `.qdoc` and returns where it
  /// went. Throws what the Quark refused with, a 409 among them when a
  /// `.qdoc` of that name exists and [overwrite] was not set.
  Future<DocumentConversion> openAsQuarkDoc({bool overwrite = false}) async {
    _isConverting = true;
    _notify();
    try {
      return await convertDocument(path, serial: serial, overwrite: overwrite);
    } finally {
      _isConverting = false;
      _notify();
    }
  }

  void _notify() {
    if (!_disposed) notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _generation++;
    super.dispose();
  }
}
