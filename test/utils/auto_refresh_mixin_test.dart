import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/auto_refresh_mixin.dart';

/// The reload button stops working for the rest of the session if one refresh
/// never returns: the in-flight guard is only released after `await refresh()`,
/// and the fetch underneath it has no timeout. That is reachable whenever the
/// connection pool is saturated — a large folder upload does it easily.
///
/// These run under [WidgetTester.runAsync] against real time, because the
/// mixin's own debounce reads DateTime.now(), which pump's fake clock does not
/// move. Hence the small but real durations.
void main() {
  testWidgets('a refresh that never returns does not wedge the next one', (
    tester,
  ) async {
    await tester.pumpWidget(const MaterialApp(home: _RefreshProbe()));
    final state = tester.state<_RefreshProbeState>(find.byType(_RefreshProbe));

    // The initial refresh hangs, as a listing request queued behind an upload
    // would.
    expect(state.starts, 1);

    // Not awaited past the call itself: refresh() bumps `starts` synchronously,
    // and awaiting a refresh that never returns is the whole point.
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 1200)),
    );
    state.manualRefresh();
    expect(state.starts, 1, reason: 'still genuinely in flight');

    // Past refreshTimeout the guard stops holding the door shut, even though
    // the first fetch never finished, and the reload button works again.
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 1200)),
    );
    state.manualRefresh();
    expect(state.starts, 2, reason: 'reload works again');

    state.releaseAll();
    await tester.pumpAndSettle();
  });

  testWidgets('a refresh that throws does not wedge the next one', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(home: _RefreshProbe(throwOnRefresh: true)),
    );
    final state = tester.state<_RefreshProbeState>(find.byType(_RefreshProbe));

    expect(state.starts, 1);

    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 1200)),
    );
    state.manualRefresh();

    expect(state.starts, 2);
  });
}

class _RefreshProbe extends StatefulWidget {
  const _RefreshProbe({this.throwOnRefresh = false});

  final bool throwOnRefresh;

  @override
  State<_RefreshProbe> createState() => _RefreshProbeState();
}

class _RefreshProbeState extends State<_RefreshProbe>
    with WidgetsBindingObserver, AutoRefreshMixin<_RefreshProbe> {
  int starts = 0;
  final List<Completer<void>> _pending = [];

  // No periodic timer: this is about the in-flight guard, not the schedule.
  @override
  Duration? get refreshInterval => null;

  @override
  Duration get refreshTimeout => const Duration(seconds: 2);

  @override
  Future<void> refresh() {
    starts++;
    if (widget.throwOnRefresh) {
      return Future<void>.error(Exception('listing failed'));
    }
    final completer = Completer<void>();
    _pending.add(completer);
    return completer.future;
  }

  void releaseAll() {
    for (final completer in _pending) {
      if (!completer.isCompleted) completer.complete();
    }
  }

  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}
