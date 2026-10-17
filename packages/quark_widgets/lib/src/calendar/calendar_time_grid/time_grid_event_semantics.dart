import 'package:flutter/semantics.dart';
import 'package:flutter/widgets.dart';

/// The screen-reader hint and actions a [TimeGridEventMover] adds to the
/// event block it wraps.
///
/// They belong on the block's own semantics node. A second node will not
/// merge with one that is transparent to hit testing, and the web engine
/// would keep the tappable node as a click target. A flick that finishes on
/// the event would then open it instead of paging the day (#3251).
class TimeGridEventSemantics extends InheritedWidget {
  /// Carries [hint] and [actions] down to the wrapped event block.
  const TimeGridEventSemantics({
    required this.hint,
    required this.actions,
    required super.child,
    super.key,
  });

  /// The keys, read with the event.
  final String hint;

  /// Moves and resizes the event. The labels are what a screen reader speaks.
  final Map<CustomSemanticsAction, VoidCallback> actions;

  /// The nearest [TimeGridEventSemantics], or null when the block is not
  /// being moved.
  static TimeGridEventSemantics? maybeOf(BuildContext context) {
    return context.dependOnInheritedWidgetOfExactType<TimeGridEventSemantics>();
  }

  @override
  bool updateShouldNotify(TimeGridEventSemantics oldWidget) {
    return hint != oldWidget.hint ||
        actions.length != oldWidget.actions.length ||
        !actions.keys.every(oldWidget.actions.containsKey);
  }
}
