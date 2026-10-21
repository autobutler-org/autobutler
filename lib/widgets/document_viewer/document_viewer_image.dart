import 'package:flutter/material.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// An image inside a document (#2987), loaded from the Quark through
/// `GET /previews/part`, which serves only the document's own media.
///
/// The view only asks for an image that names a part of the package; one
/// linked from outside it is never fetched. An image that fails to load or
/// decode (EMF and WMF, which nothing here decodes, among them) shows its alt
/// text.
class DocumentViewerImage extends StatelessWidget {
  /// The image block, with the part it names.
  final QuarkDocumentImage image;

  /// The document's files-relative path.
  final String filePath;

  /// The device the document is on, null for the internal drive.
  final String? serial;

  const DocumentViewerImage({
    required this.image,
    required this.filePath,
    this.serial,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final part = image.part;
    final fallback = Row(
      mainAxisAlignment: MainAxisAlignment.center,
      spacing: tokens.spacingSm,
      children: [
        Icon(QuarkIcons.broken_image_outlined, color: tokens.mutedForeground),
        Flexible(
          child: Text(image.alt.isEmpty ? 'Image not available' : image.alt),
        ),
      ],
    );
    if (part == null) return fallback;
    return Image.network(
      PreviewsService.partUrl(filePath, part, serial: serial).toString(),
      headers: PreviewsService.instance.authHeaders,
      semanticLabel: image.alt,
      fit: BoxFit.contain,
      errorBuilder: (context, error, stack) => fallback,
    );
  }
}
