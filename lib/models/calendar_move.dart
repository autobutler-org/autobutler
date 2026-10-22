import 'package:quark/models/calendar_event.dart';

/// Which occurrences of a repeating event a move changes (#3215).
///
/// The names are the `scope` values `POST /calendar/events/:id/move` takes.
enum CalendarMoveScope {
  /// The one occurrence that was dragged or nudged.
  one,

  /// That occurrence and every one after it.
  following,

  /// The whole series.
  all,
}

/// The series after a move, and the one-off or new series when the move
/// created one.
class CalendarMoveResult {
  /// A move that left [series] as stored and created [created], if it did.
  const CalendarMoveResult({required this.series, this.created});

  /// The original series after the move.
  final CalendarEvent series;

  /// The one-off or the new series. Null when the whole series moved.
  final CalendarEvent? created;
}
