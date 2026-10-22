import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/download_destination.dart';

// Where a download lands on a phone (#3334): pictures and videos go to the
// photo library, everything else to Files.
void main() {
  test('pictures go to the photo library', () {
    for (final name in [
      'beach.jpg',
      'BEACH.JPEG',
      'scan.png',
      'live.heic',
      'anim.gif',
      'web.webp',
      'raw.dng',
    ]) {
      expect(
        downloadDestinationFor(name),
        DownloadDestination.photoLibraryImage,
        reason: name,
      );
    }
  });

  test('videos go to the photo library', () {
    for (final name in ['clip.mp4', 'clip.MOV', 'clip.m4v', 'clip.3gp']) {
      expect(
        downloadDestinationFor(name),
        DownloadDestination.photoLibraryVideo,
        reason: name,
      );
    }
  });

  test('everything else goes to Files', () {
    for (final name in [
      'report.pdf',
      'notes.txt',
      'song.mp3',
      'folder.zip',
      'drawing.svg',
      'camera.cr2',
      'movie.mkv',
      'no-extension',
    ]) {
      expect(
        downloadDestinationFor(name),
        DownloadDestination.files,
        reason: name,
      );
    }
  });
}
