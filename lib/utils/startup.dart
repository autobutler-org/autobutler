import 'dart:async';

import 'package:flutter/foundation.dart';

/// Awaits [work], calling [onSlow] once if it is still running after [grace].
///
/// `main` uses it to paint a loader when the saved settings are slow to load,
/// instead of leaving the blank native launch screen up (#2066). [work]'s
/// result or error comes back unchanged, and [onSlow] is never called once
/// [work] has finished.
Future<T> awaitWithSlowNotice<T>(
  Future<T> work, {
  required Duration grace,
  required VoidCallback onSlow,
}) async {
  final timer = Timer(grace, onSlow);
  try {
    return await work;
  } finally {
    timer.cancel();
  }
}
