/// Where an `ImageElement`'s picture comes from: a file in the Quark, an
/// asset uploaded with the presentation, or a picture still inside an
/// office package being previewed.
///
/// The package never loads a picture. It stores the source as one string in
/// `.qslide` — [ref] — and hands it back to the host app's image builder,
/// which resolves it:
///
/// - [QuarkFileImage] — a path in the user's Quark files, stored as the path
///   itself (`photos/dog.jpg`). Files written before sources were typed
///   read this way.
/// - [UploadedAssetImage] — an asset the app uploaded for this
///   presentation, stored as `asset:<id>`.
/// - [PackagePartImage] — a zip entry inside the office file being
///   previewed, stored as `part:<entry>` (`part:ppt/media/image3.png`).
///   It is not a file in the Quark. A deck that still holds one is a
///   preview, so the editor can refuse to save it.
///
/// ```dart
/// doc.insertImage(slideId, const QuarkFileImage('photos/dog.jpg'),
///     (width: 4032, height: 3024));
/// switch (ImageSource.parse(element.source)) {
///   QuarkFileImage(:final path) => downloadUrl(path),
///   UploadedAssetImage(:final assetId) => assetUrl(assetId),
///   PackagePartImage(:final part) => partUrl(part),
/// }
/// ```
sealed class ImageSource {
  const ImageSource();

  /// Reads a stored reference.
  ///
  /// `part:<entry>` is a [PackagePartImage], `asset:<id>` is an
  /// [UploadedAssetImage], and anything else is a [QuarkFileImage] path.
  /// A path that only contains one of those prefixes later, such as
  /// `photos/part:x` or `/part:x`, stays a file.
  factory ImageSource.parse(String ref) {
    if (ref.startsWith(PackagePartImage.prefix)) {
      return PackagePartImage(ref.substring(PackagePartImage.prefix.length));
    }
    if (ref.startsWith(UploadedAssetImage.prefix)) {
      return UploadedAssetImage(
        ref.substring(UploadedAssetImage.prefix.length),
      );
    }
    return QuarkFileImage(ref);
  }

  /// The string stored in `.qslide`, which [ImageSource.parse] reads back.
  String get ref;

  @override
  bool operator ==(Object other) =>
      other is ImageSource &&
      other.runtimeType == runtimeType &&
      other.ref == ref;

  @override
  int get hashCode => ref.hashCode;

  @override
  String toString() => '$runtimeType($ref)';
}

/// A picture that is a file in the user's Quark, by its [path].
class QuarkFileImage extends ImageSource {
  /// Creates a source for the Quark file at [path].
  const QuarkFileImage(this.path);

  /// The file's path, as the Quark's file API names it.
  final String path;

  @override
  String get ref => path;
}

/// A picture uploaded with the presentation, by the [assetId] the app gave
/// it.
class UploadedAssetImage extends ImageSource {
  /// Creates a source for the uploaded asset [assetId].
  const UploadedAssetImage(this.assetId);

  /// What a stored reference to an uploaded asset starts with.
  static const prefix = 'asset:';

  /// The id the app's upload returned.
  final String assetId;

  @override
  String get ref => '$prefix$assetId';
}

/// A picture that is still inside the office package being previewed, named
/// by its zip [part] (`ppt/media/image3.png`).
///
/// Stored as `part:<part>`. The host fetches that entry from the package.
/// It is not a path in the user's files, and [ImageSource.parse] will not
/// read a saved file path that merely looks similar unless the path itself
/// starts with [prefix].
class PackagePartImage extends ImageSource {
  /// Creates a source for the zip entry [part].
  const PackagePartImage(this.part);

  /// What a stored reference to a package entry starts with.
  static const prefix = 'part:';

  /// The zip entry, such as `ppt/media/image3.png` or `Pictures/photo.png`.
  final String part;

  @override
  String get ref => '$prefix$part';
}
