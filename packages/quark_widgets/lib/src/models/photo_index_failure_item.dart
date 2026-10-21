import 'package:flutter/foundation.dart';

/// One photo the library index could not read, as `PhotoIndexStatusPanel`
/// lists it: an id, its file name, where it lives, and why, already worded
/// by the caller.
@immutable
class PhotoIndexFailureItem {
  /// Creates a failure.
  const PhotoIndexFailureItem({
    required this.id,
    required this.name,
    required this.location,
    required this.reason,
  });

  /// Stable and unique across the list; the row's key is built from it.
  final String id;

  /// The file name.
  final String name;

  /// Where the file is, such as "Photos › 2024" or "USB drive".
  final String location;

  /// Why it could not be read, in plain words.
  final String reason;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PhotoIndexFailureItem &&
          other.id == id &&
          other.name == name &&
          other.location == location &&
          other.reason == reason;

  @override
  int get hashCode => Object.hash(id, name, location, reason);
}
