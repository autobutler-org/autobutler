import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/photo_location.dart';

/// #2575: a duplicate copy's place reads as folder names, never as the
/// Quark's internal storage path.
void main() {
  group('photoLocation', () {
    test('names the Photos library rather than its home path', () {
      expect(photoLocation('users/ux-test/photos/dup-copy-1.jpg'), 'Photos');
    });

    test('names folders inside a home without the home', () {
      expect(
        photoLocation('users/ux-test/photos/2024/Beach/a.jpg'),
        'Photos › 2024 › Beach',
      );
      expect(photoLocation('users/ux-test/Camera/a.jpg'), 'Camera');
    });

    test('a file at the top of a home or the drive is in Files', () {
      expect(photoLocation('users/ux-test/a.jpg'), 'Files');
      expect(photoLocation('a.jpg'), 'Files');
    });

    test('names a group folder by the group', () {
      expect(photoLocation('groups/Family/Trips/a.jpg'), 'Family › Trips');
    });

    test('ignores stray slashes and dots', () {
      expect(photoLocation('/users/ux-test//./photos/a.jpg'), 'Photos');
    });

    test('leads with the drive on a drive other than the internal one', () {
      expect(
        photoLocation('Backups/a.jpg', deviceName: 'Backup drive'),
        'Backup drive · Backups',
      );
      expect(
        photoLocation('a.jpg', deviceName: 'Backup drive'),
        'Backup drive',
      );
    });

    test('keeps a users folder on another drive, which is not a home', () {
      expect(
        photoLocation('users/old/a.jpg', deviceName: 'USB'),
        'USB · users › old',
      );
    });

    test('never shows a slash', () {
      for (final path in [
        'users/ux-test/photos/a.jpg',
        'users/ux-test/x/y/z/a.jpg',
        'groups/g/a.jpg',
        'deep/er/a.jpg',
      ]) {
        expect(photoLocation(path), isNot(contains('/')));
      }
    });
  });
  // #3364: a photo's GPS position opens in the phone's maps app, or a maps
  // page in the browser, rather than sitting there as two numbers.
  group('photoMapUri', () {
    test('asks a maps search for the coordinates', () {
      final uri = photoMapUri(48.8566, 2.3522);
      expect(uri.scheme, 'https');
      expect(uri.host, 'www.google.com');
      expect(uri.path, '/maps/search/');
      expect(uri.queryParameters, {'api': '1', 'query': '48.8566,2.3522'});
    });

    test('keeps the sign of a southern or western position', () {
      expect(
        photoMapUri(-33.8568, -151.2153).queryParameters['query'],
        '-33.8568,-151.2153',
      );
    });
  });
}
