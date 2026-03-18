import 'dart:async';

import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/widgets.dart';

/// Adds auto-refresh and non-disruptive loading state to a [State].
///
/// Usage:
/// ```dart
/// class _MyPageState extends State<MyPage>
///     with AutoRefreshMixin {
///
///   @override
///   Duration get refreshInterval => const Duration(seconds: 30);
///
///   @override
///   Future<void> refresh() async {
///     // fetch data; update state inside here
///   }
/// }
/// ```
///
/// In the widget tree, use [isInitialLoad] for full-screen spinners and
/// [isRefreshing] for the in-progress indicator on the refresh button.
mixin AutoRefreshMixin<T extends StatefulWidget>
    on State<T>, WidgetsBindingObserver {
  // ── State ──────────────────────────────────────────────────────────────────

  /// True only for the very first load (no data yet).
  bool isInitialLoad = true;

  /// True while a refresh (initial or subsequent) is in flight.
  bool isRefreshing = false;

  Timer? _refreshTimer;
  bool _refreshInFlight = false;

  // ── Overrides ──────────────────────────────────────────────────────────────

  /// How often to auto-refresh. Reads from [AppSettings.refreshIntervalSeconds]
  /// by default. Override to use a fixed interval regardless of settings.
  /// Return null or [Duration.zero] to disable auto-refresh.
  Duration? get refreshInterval {
    final seconds = AppSettings.instance.refreshIntervalSeconds;
    return seconds > 0 ? Duration(seconds: seconds) : null;
  }

  /// Perform the data fetch. Called on initial load and on each auto-refresh.
  /// Update your widget state inside this method.
  Future<void> refresh();

  // ── Lifecycle ──────────────────────────────────────────────────────────────

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _triggerRefresh(initial: true);
    _startTimer();
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _startTimer();
      _triggerRefresh();
    } else if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.inactive) {
      _refreshTimer?.cancel();
    }
  }

  // ── Public API ─────────────────────────────────────────────────────────────

  /// Call from a manual refresh button. Same guard as the timer.
  Future<void> manualRefresh() => _triggerRefresh();

  // ── Internals ──────────────────────────────────────────────────────────────

  void _startTimer() {
    _refreshTimer?.cancel();
    final interval = refreshInterval;
    if (interval == null || interval == Duration.zero) return;
    _refreshTimer = Timer.periodic(interval, (_) => _triggerRefresh());
  }

  Future<void> _triggerRefresh({bool initial = false}) async {
    if (_refreshInFlight) return;
    _refreshInFlight = true;
    if (mounted) {
      setState(() => isRefreshing = true);
    }
    try {
      await refresh();
    } finally {
      _refreshInFlight = false;
      if (mounted) {
        setState(() {
          isRefreshing = false;
          if (initial) isInitialLoad = false;
        });
      }
    }
  }
}
