import 'package:flutter/foundation.dart';
import 'package:quark/controllers/code_viewer_controller.dart';
import 'package:quark/services/code_viewer_service.dart';
import 'package:quark/services/model_viewer_service.dart';
import 'package:quark/utils/model_viewer_config.dart';
import 'package:quark/utils/model_viewer_files.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// Parses a model file's bytes; [ModelViewerService.parse] by default.
typedef ParseModelFn =
    Future<QuarkModelMesh> Function(
      Uint8List bytes,
      QuarkModelFormat format,
      QuarkModelLimits limits,
    );

/// State behind the 3D model viewer for the file at [path] (#2988): the
/// parsed mesh, whether it shows as a wireframe, and why it could not be
/// shown.
///
/// A file over [limits]' size is never downloaded: a one-byte read learns
/// its size first, and the real read stops at the cap either way. A file
/// the viewer will not draw — too large, too detailed, too slow to read,
/// not a valid model — sets [fallback], and the page shows the file's
/// details instead. A failed request sets [error], which offers Retry.
class ModelViewerController extends ChangeNotifier {
  ModelViewerController({
    required this.path,
    this.serial,
    this.readRange = CodeViewerService.readRange,
    this.parse = ModelViewerService.parse,
    this.limits = ModelViewerConfig.limits,
  });

  /// The file's files-relative path.
  final String path;

  /// The device the file is on, null for the internal drive.
  final String? serial;

  final ReadCodeRangeFn readRange;

  final ParseModelFn parse;

  /// The caps the file is held to.
  final QuarkModelLimits limits;

  QuarkModelMesh? _mesh;
  bool _isLoading = true;
  Object? _error;
  QuarkModelParseIssue? _fallback;
  int? _fileSize;
  bool _wireframe = false;
  int _generation = 0;
  bool _disposed = false;

  /// The model, once read.
  QuarkModelMesh? get mesh => _mesh;

  /// Whether the file is still being read or parsed.
  bool get isLoading => _isLoading;

  /// What the request failed with, null once it has worked. The page turns
  /// it into a sentence.
  Object? get error => _error;

  /// Why the viewer will not draw this file, null when it will. The page
  /// shows the file's details instead.
  QuarkModelParseIssue? get fallback => _fallback;

  /// The file's size in bytes, null until the Quark says.
  int? get fileSize => _fileSize;

  /// Whether the model shows as edges.
  bool get wireframe => _wireframe;

  /// Reads and parses the file, dropping what was read before.
  Future<void> load() async {
    final generation = ++_generation;
    _isLoading = true;
    _error = null;
    _fallback = null;
    _notify();
    try {
      final format = modelFormatForName(path);
      if (format == null) {
        throw const QuarkModelParseException(QuarkModelParseIssue.malformed);
      }
      final probe = await readRange(path, start: 0, length: 1, serial: serial);
      if (generation != _generation) return;
      _fileSize = probe.totalSize;
      _checkSize(probe.totalSize ?? 0);
      // One byte past the cap tells a file at the cap from one over it when
      // the Quark did not say how big it is.
      final page = await readRange(
        path,
        start: 0,
        length: limits.maxBytes + 1,
        serial: serial,
      );
      if (generation != _generation) return;
      _fileSize ??= page.totalSize ?? page.bytes.length;
      _checkSize(page.bytes.length);
      final mesh = await parse(page.bytes, format, limits);
      if (generation != _generation) return;
      _mesh = mesh;
    } on QuarkModelParseException catch (e) {
      if (generation != _generation) return;
      _mesh = null;
      _fallback = e.issue;
    } catch (e) {
      if (generation != _generation) return;
      _error = e;
    }
    _isLoading = false;
    _notify();
  }

  void _checkSize(int bytes) {
    if (bytes > limits.maxBytes) {
      throw QuarkModelParseException(
        QuarkModelParseIssue.tooLarge,
        '$bytes bytes',
      );
    }
  }

  /// Shows the model as edges, or as shaded faces.
  void setWireframe(bool wireframe) {
    if (_wireframe == wireframe) return;
    _wireframe = wireframe;
    _notify();
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
