/// Where the photo at [relPath] lives, worded for someone who has never seen
/// the Quark's storage layout (#2575): folder names joined with `›`, never a
/// slash-separated path.
///
/// On the internal drive an account's home, `users/<name>`, is dropped, so a
/// photo in `users/ux-test/photos` reads "Photos" and one at the top of the
/// home reads "Files". A group folder, `groups/<name>`, reads as the group's
/// name. A [deviceName] means the photo is on another drive, whose paths
/// start at the drive itself: the drive leads, and a `users` folder there is
/// only a folder.
String photoLocation(String relPath, {String? deviceName}) {
  final segments = relPath
      .split('/')
      .where((s) => s.isNotEmpty && s != '.')
      .toList();
  if (segments.isNotEmpty) segments.removeLast();
  if (deviceName == null && segments.length >= 2) {
    if (segments.first == 'users') {
      segments.removeRange(0, 2);
    } else if (segments.first == 'groups') {
      segments.removeAt(0);
    }
  }
  if (deviceName == null && segments.isNotEmpty && segments.first == 'photos') {
    segments.first = 'Photos';
  }
  final folder = segments.join(' › ');
  if (deviceName == null) return folder.isEmpty ? 'Files' : folder;
  return folder.isEmpty ? deviceName : '$deviceName · $folder';
}

/// A maps search for the GPS position a photo was taken at (#3364).
///
/// A plain https link, so it works everywhere the app runs: a phone hands it
/// to its maps app, a browser opens the maps page. The Quark cannot turn a
/// position into a place name, so the search is for the coordinates.
Uri photoMapUri(double latitude, double longitude) => Uri.https(
  'www.google.com',
  '/maps/search/',
  {'api': '1', 'query': '$latitude,$longitude'},
);
