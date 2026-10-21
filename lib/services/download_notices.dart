import 'dart:async';

/// Sentences about a finished download that the page which started it has no
/// way to say itself: the save path decides them, below every caller.
///
/// The app shows each one in a snack bar on whatever page is open (#3334).
final StreamController<String> _notices = StreamController<String>.broadcast();

/// What a download wants the person told, as it happens.
Stream<String> get downloadNotices => _notices.stream;

/// Tells the person [message] about a download, through [downloadNotices].
void announceDownloadNotice(String message) => _notices.add(message);
