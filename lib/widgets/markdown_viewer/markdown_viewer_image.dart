import 'package:flutter/material.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/utils/markdown_viewer_files.dart';
import 'package:quark_icons/quark_icons.dart';

/// An image in a Markdown file (#2987).
///
/// One on the Quark — a relative address, resolved against the file's folder
/// — loads through the download endpoint. One on the web is not fetched
/// until the reader taps "Load remote image": loading it tells that server
/// who opened the file. Anything else (`data:`, an address that does not
/// parse) shows its alt text.
///
/// Key prefix: `markdown_viewer_load_image_<src>` on the load button.
class MarkdownViewerImage extends StatelessWidget {
  /// The image's address, as written in the file.
  final String src;

  /// The image's alt text, shown when it cannot be or has not been loaded.
  final String alt;

  /// The Markdown file's files-relative path, which [src] is relative to.
  final String filePath;

  /// The device the file is on, null for the internal drive.
  final String? serial;

  /// Whether the reader has asked for this remote image to load.
  final bool remoteAllowed;

  /// Called when the reader taps the load button of a remote image.
  final VoidCallback onAllowRemote;

  const MarkdownViewerImage({
    required this.src,
    required this.alt,
    required this.filePath,
    required this.onAllowRemote,
    this.serial,
    this.remoteAllowed = false,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    final target = resolveMarkdownLink(filePath, src);
    final fallback = Text(alt.isEmpty ? src : alt);
    Widget onError(BuildContext context, Object error, StackTrace? stack) =>
        fallback;
    switch (target.kind) {
      // ponytail: Image.network decodes raster formats only, so an SVG shows
      // its alt text. Route by extension to the SVG renderer if that matters.
      case MarkdownLinkKind.file:
        return Image.network(
          FilesService.downloadUrl(target.value, serial: serial).toString(),
          headers: FilesService.instance.authHeaders,
          semanticLabel: alt,
          errorBuilder: onError,
        );
      case MarkdownLinkKind.remote:
        if (remoteAllowed) {
          return Image.network(
            target.value,
            semanticLabel: alt,
            errorBuilder: onError,
          );
        }
        return Tooltip(
          message: target.value,
          child: OutlinedButton.icon(
            key: ValueKey('markdown_viewer_load_image_$src'),
            onPressed: onAllowRemote,
            icon: const Icon(QuarkIcons.image_outlined),
            label: Text(
              alt.isEmpty ? 'Load remote image' : 'Load remote image: $alt',
            ),
          ),
        );
      case MarkdownLinkKind.other:
        return fallback;
    }
  }
}
