import 'package:flutter/foundation.dart';
import 'package:quark/controllers/slide_present_controller.dart';
import 'package:quark/controllers/slides_controller.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark/services/slides_service.dart';
import 'package:quark_slides/quark_slides.dart';

/// Reads a presentation as a deck; see [PreviewsService.slides].
typedef LoadSlidesPreviewFn =
    Future<SlidesPreview> Function(String path, {String? serial});

/// State behind the presentation viewer for the `.pptx` or `.pptm` at
/// [path] (#3268): the deck the Quark converted, whether it is still loading
/// or failed to, the slide showing, whether its speaker notes are showing,
/// and the warnings about what the conversion left out until they are
/// dismissed.
///
/// The Quark reads the file in place, so nothing here ever holds the file
/// itself. Its pictures stay inside it, named by `part:` references that
/// [imageUrl] turns into `GET /previews/part` addresses.
///
/// Service calls are parameters, so a test passes fakes.
class PresentationViewerController extends ChangeNotifier {
  PresentationViewerController({
    required this.path,
    this.serial,
    int startIndex = 0,
    this.loadSlides = PreviewsService.slides,
    this.importPowerPoint = SlidesService.importPowerPoint,
    this.partUrl = previewPartUrl,
  }) : _index = startIndex;

  /// The presentation's files-relative path.
  final String path;

  /// The device the presentation is on, null for the internal drive.
  final String? serial;

  final LoadSlidesPreviewFn loadSlides;
  final ImportPowerPointFn importPowerPoint;

  /// Builds the URL a picture inside the file at [path] is fetched from.
  final PartUrlFn partUrl;

  Presentation? _deck;
  List<PowerPointImportWarning> _warnings = const [];
  bool _warningsDismissed = false;
  bool _isLoading = true;
  Object? _error;
  int _index;
  bool _notesVisible = false;
  int _generation = 0;
  bool _disposed = false;

  /// The presentation as last read; null before the first read lands.
  Presentation? get deck => _deck;

  /// The slides in show order; empty before the first read lands.
  List<Slide> get slides => _deck?.slides ?? const [];

  /// Whether the first read is still running. A later [load] keeps the deck
  /// already shown.
  bool get isLoading => _isLoading;

  /// What the last read failed with, null once it has worked. The page turns
  /// it into a sentence.
  Object? get error => _error;

  /// What the conversion left out or approximated, empty once dismissed.
  List<PowerPointImportWarning> get warnings =>
      _warningsDismissed ? const [] : _warnings;

  /// The position of [currentSlide] in [slides].
  int get index => _index;

  /// The slide showing, or null while there is none.
  Slide? get currentSlide => slides.isEmpty ? null : slides[_index];

  /// Whether [currentSlide] is the first.
  bool get isFirst => _index == 0;

  /// Whether [currentSlide] is the last.
  bool get isLast => _index >= slides.length - 1;

  /// Where the reader is, such as "Slide 2 of 5".
  String get position => 'Slide ${_index + 1} of ${slides.length}';

  /// Whether the current slide's speaker notes are showing.
  bool get notesVisible => _notesVisible;

  /// Reads the presentation, replacing what was read before once the new
  /// read lands. A read that a newer one overtook is dropped.
  Future<void> load() async {
    final generation = ++_generation;
    _error = null;
    _notify();
    try {
      final preview = await loadSlides(path, serial: serial);
      if (generation != _generation) return;
      _deck = preview.deck;
      _warnings = preview.warnings;
      _index = _clamped(_index);
    } catch (e) {
      if (generation != _generation) return;
      _error = e;
    }
    _isLoading = false;
    _notify();
  }

  /// Shows the slide at [index], kept within the slides there are.
  void goTo(int index) {
    final to = _clamped(index);
    if (to == _index) return;
    _index = to;
    _notify();
  }

  /// Shows the next slide; nothing at the last.
  void next() => goTo(_index + 1);

  /// Shows the previous slide; nothing at the first.
  void previous() => goTo(_index - 1);

  /// Shows or hides the speaker notes.
  void toggleNotes() {
    _notesVisible = !_notesVisible;
    _notify();
  }

  /// Hides the warnings until the viewer is opened again.
  void dismissWarnings() {
    _warningsDismissed = true;
    _notify();
  }

  /// Imports the presentation beside itself as a `.qslide` ("Open as Quark
  /// slides"). Throws what the Quark refused with; the page reports it.
  Future<PowerPointImport> importAsQuarkSlides() =>
      importPowerPoint(path, serial: serial);

  /// The URL of the picture a slide names by [source], or null for one that
  /// is not inside the file: a converted deck names only its own parts, and
  /// nothing outside it is fetched.
  Uri? imageUrl(String source) => switch (ImageSource.parse(source.trim())) {
    PackagePartImage(:final part) => partUrl(
      part,
      filePath: path,
      serial: serial,
    ),
    _ => null,
  };

  int _clamped(int index) =>
      slides.isEmpty ? 0 : index.clamp(0, slides.length - 1);

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
