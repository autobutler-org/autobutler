import 'package:quark/utils/file_kind.dart';

/// Where a downloaded file goes on this device (#3334).
///
/// A picture or a video goes into the phone's photo library, next to the rest
/// of the person's pictures: Apple Photos on iOS, the gallery on Android.
/// Everything else goes through the save dialog, which on iOS puts it in
/// Files.
enum DownloadDestination {
  /// The photo library, as a picture.
  photoLibraryImage,

  /// The photo library, as a video.
  photoLibraryVideo,

  /// The platform's save dialog.
  files,
}

/// Image extensions the photo library accepts. Narrower than
/// [FileKind.image] on purpose: Apple Photos refuses most camera RAW formats
/// and AVIF, so those keep going to Files rather than failing into it.
const photoLibraryImageExtensions = {
  '.jpg',
  '.jpeg',
  '.png',
  '.gif',
  '.heic',
  '.heif',
  '.webp',
  '.tif',
  '.tiff',
  '.bmp',
  '.dng',
};

/// Video extensions the photo library accepts. Narrower than
/// [FileKind.video]: Apple Photos plays MP4 and QuickTime, not MKV or WebM.
const photoLibraryVideoExtensions = {'.mp4', '.m4v', '.mov', '.3gp'};

/// Where a download named [fileName] should land on a phone. Only Android
/// and iOS ask: the desktop and the web have no photo library to save into.
DownloadDestination downloadDestinationFor(String fileName) {
  final extension = fileExtension(fileName);
  if (photoLibraryImageExtensions.contains(extension)) {
    return DownloadDestination.photoLibraryImage;
  }
  if (photoLibraryVideoExtensions.contains(extension)) {
    return DownloadDestination.photoLibraryVideo;
  }
  return DownloadDestination.files;
}
