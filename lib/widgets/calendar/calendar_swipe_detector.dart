import 'package:flutter/gestures.dart' show DragStartBehavior;
import 'package:flutter/widgets.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// Steps Day, Week or Month one span back or forward when [child] is swiped
/// sideways: right for [onPrevious], left for [onNext].
///
/// The view tracks the finger for the whole drag, then snaps the rest of the
/// way off screen when the swipe counts, or back to where it was when it
/// does not (#3250). The step is still one span, the same one the arrows
/// take. Under reduced motion the snap is instant; the drag still tracks
/// the finger, which is what shows where the page is going.
///
/// A swipe counts once it has traveled [distance] from where the pointer
/// went down, or is let go faster than [velocity], and it goes the way the
/// finger traveled. The down point is what counts, not where the drag won
/// the arena: a touch drag competes with the timeline's scroll, and starting
/// the measurement there dropped the first part of a short swipe and could
/// turn a small lift the wrong way (#3250). Speed alone would miss a mouse
/// drag, which usually comes to rest before the button is released and so
/// ends with no fling at all (#2885); a short, slow drag steps nowhere.
class CalendarSwipeDetector extends StatefulWidget {
  /// Wraps [child] in the swipe.
  const CalendarSwipeDetector({
    required this.onPrevious,
    required this.onNext,
    required this.child,
    super.key,
  });

  /// Steps one span back: a swipe to the right.
  final VoidCallback onPrevious;

  /// Steps one span forward: a swipe to the left.
  final VoidCallback onNext;

  /// What is swiped.
  final Widget child;

  /// How far a drag has to go to step, in logical pixels, from pointer down.
  static const double distance = 50;

  /// How fast a shorter fling has to be to step, in logical pixels a second.
  static const double velocity = 300;

  @override
  State<CalendarSwipeDetector> createState() => _CalendarSwipeDetectorState();
}

class _CalendarSwipeDetectorState extends State<CalendarSwipeDetector>
    with SingleTickerProviderStateMixin {
  /// How long the page takes to finish leaving, or to spring back.
  static const _snapDuration = Duration(milliseconds: 200);

  late final AnimationController _snap;

  /// Horizontal shift of [CalendarSwipeDetector.child], in logical pixels.
  /// Positive moves it to the right, the way a swipe back travels.
  double _offset = 0;

  /// [_offset] when this drag was recognized, so one that starts during a
  /// snap continues from where the page sits.
  double _base = 0;

  /// Pointer-down x for the drag in progress.
  double _downX = 0;

  double _from = 0;
  double _to = 0;

  /// -1 steps back, 1 steps forward, 0 springs back, null is not snapping.
  int? _pending;

  @override
  void initState() {
    super.initState();
    _snap = AnimationController(vsync: this, duration: _snapDuration)
      ..addListener(_tick)
      ..addStatusListener(_onStatus);
  }

  @override
  void dispose() {
    _snap.dispose();
    super.dispose();
  }

  void _tick() {
    if (_pending == null) return;
    setState(() {
      final t = Curves.easeOut.transform(_snap.value);
      _offset = _from + (_to - _from) * t;
    });
  }

  void _onStatus(AnimationStatus status) {
    if (status == AnimationStatus.completed) _finish();
  }

  void _start(DragStartDetails details) {
    _pending = null;
    _snap.stop();
    _base = _offset;
    _downX = details.globalPosition.dx;
  }

  void _update(DragUpdateDetails details) {
    var dx = _base + details.globalPosition.dx - _downX;
    final width = context.size?.width ?? 0;
    if (width > 0) dx = dx.clamp(-width, width).toDouble();
    setState(() => _offset = dx);
  }

  void _end(DragEndDetails details) {
    final traveled = details.globalPosition.dx - _downX;
    final fling = details.primaryVelocity ?? 0;
    final way = traveled.abs() >= CalendarSwipeDetector.distance
        ? traveled
        : (fling.abs() > CalendarSwipeDetector.velocity ? fling : 0);
    final width = context.size?.width ?? 0;
    if (way > 0) {
      _pending = -1;
      _animateTo(width);
    } else if (way < 0) {
      _pending = 1;
      _animateTo(-width);
    } else {
      _pending = 0;
      _animateTo(0);
    }
  }

  void _cancel() {
    _pending = 0;
    _animateTo(0);
  }

  void _animateTo(double target) {
    if (!mounted) return;
    if (reduceMotionOf(context) || (target - _offset).abs() < 1) {
      _finish();
      return;
    }
    _from = _offset;
    _to = target;
    _snap.forward(from: 0);
  }

  void _finish() {
    if (!mounted || _pending == null) return;
    final pending = _pending!;
    _pending = null;
    final VoidCallback? step = switch (pending) {
      < 0 => widget.onPrevious,
      > 0 => widget.onNext,
      _ => null,
    };
    setState(() => _offset = 0);
    step?.call();
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      // From the pointer-down point. The arena is won later, after the
      // timeline's scroll has competed, and measuring from there dropped
      // the first part of a short touch swipe (#3250).
      dragStartBehavior: DragStartBehavior.down,
      onHorizontalDragStart: _start,
      onHorizontalDragUpdate: _update,
      onHorizontalDragEnd: _end,
      onHorizontalDragCancel: _cancel,
      child: ClipRect(
        child: Transform.translate(
          offset: Offset(_offset, 0),
          child: widget.child,
        ),
      ),
    );
  }
}
