import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/slides_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_slides/quark_slides.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// What "Open as Quark doc" wrote: the new `.qdoc`'s [path], and what the
/// conversion could not carry across, in the Quark's words.
typedef DocumentConversion = ({String path, List<String> warnings});

/// A presentation as the viewer shows it: the [deck] the Quark converted,
/// with each picture a `part:` reference into the file, and the [warnings]
/// about what it left out or approximated.
typedef SlidesPreview = ({
  Presentation deck,
  List<PowerPointImportWarning> warnings,
});

/// Calls `/api/v0/previews`, the read-only views of office documents the
/// client cannot open itself (#2987): a word-processing document as a block
/// stream, a presentation as a `.qslide` deck, and the images inside them. Also converts a document to a `.qdoc`
/// through `POST /files/convert/document`, which reads it with the same
/// reader.
class PreviewsService with AuthenticatedService {
  PreviewsService._();
  static final PreviewsService instance = PreviewsService._();

  /// Reads the `.docx`, `.docm`, `.odt` or `.rtf` at [path] as the Quark's
  /// NDJSON block stream and parses it off the UI isolate.
  ///
  /// A refused request throws an [ApiException]. A stream cut short, with no
  /// `end` line, throws a [FormatException] rather than showing part of the
  /// document as if it were all of it.
  static Future<QuarkDocument> document(String path, {String? serial}) async {
    final response = await instance.authenticatedGet(
      _uri('/api/v0/previews/document', {'filePath': path, 'serial': serial}),
    );
    if (response.statusCode != 200) {
      throw ApiException(response.statusCode, 'preview the document');
    }
    // NDJSON names no charset, which `body` would read as Latin-1.
    return compute(_parseDocument, response.bodyBytes);
  }

  /// Reads the `.pptx` or `.pptm` at [path] as a deck, parsed off the UI
  /// isolate (#3268).
  ///
  /// A refused request throws an [ApiException]. A body that does not parse,
  /// which is how a conversion that failed partway arrives, throws a
  /// [FormatException] or a [QslideFormatException].
  static Future<SlidesPreview> slides(String path, {String? serial}) async {
    final response = await instance.authenticatedGet(
      _uri('/api/v0/previews/slides', {'filePath': path, 'serial': serial}),
    );
    if (response.statusCode != 200) {
      throw ApiException(response.statusCode, 'preview the presentation');
    }
    return compute(_parseSlides, response.bodyBytes);
  }

  /// The address of [part], an image inside the document at [path], for a
  /// reader that sends [AuthenticatedService.authHeaders] itself.
  static Uri partUrl(String path, String part, {String? serial}) => _uri(
    '/api/v0/previews/part',
    {'filePath': path, 'part': part, 'serial': serial},
  );

  /// Writes the document at [path] beside itself as a `.qdoc` ("Open as
  /// Quark doc").
  ///
  /// Throws an [ApiException] with status 409 when a `.qdoc` of that name is
  /// already there and [overwrite] was not set, so the caller can ask before
  /// replacing the reader's own work.
  static Future<DocumentConversion> convertToQdoc(
    String path, {
    String? serial,
    bool overwrite = false,
  }) async {
    final response = await instance.authenticatedPost(
      _uri('/api/v0/files/convert/document', {
        'filePath': path,
        'serial': serial,
        if (overwrite) 'overwrite': 'true',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(response.statusCode, 'convert the document');
    }
    final decoded = jsonDecode(response.body);
    final written = decoded is Map<String, dynamic> ? decoded['path'] : null;
    if (written is! String || written.isEmpty) {
      throw Exception('Convert response carried no path');
    }
    return (
      path: written,
      warnings: [
        for (final warning in decoded['warnings'] as List? ?? const [])
          if (warning is String) warning,
      ],
    );
  }

  /// [endpoint] on the active Quark with [query], leaving out empty values.
  static Uri _uri(String endpoint, Map<String, String?> query) => apiBaseUri
      .resolve(endpoint)
      .replace(
        queryParameters: {
          for (final MapEntry(:key, :value) in query.entries)
            if (value != null && value.trim().isNotEmpty) key: value,
        },
      );

  static SlidesPreview _parseSlides(Uint8List bytes) {
    final root = jsonDecode(utf8.decode(bytes));
    if (root is! Map<String, dynamic>) {
      throw const FormatException('Slides preview is not an object');
    }
    return (
      deck: QslideCodec.fromJson(root['deck']),
      warnings: [
        for (final w in root['warnings'] as List? ?? const [])
          if (w is Map<String, dynamic>)
            (
              slide: (w['slide'] as num?)?.toInt() ?? 0,
              message: w['message'] as String? ?? '',
            ),
      ],
    );
  }

  static QuarkDocument _parseDocument(Uint8List bytes) =>
      QuarkDocument.parse(utf8.decode(bytes, allowMalformed: true));
}
