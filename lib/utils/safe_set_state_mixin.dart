import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';

/// Mixin that provides [setStateSafely], a drop-in replacement for [setState]
/// that guards against calling setState during an active build phase or after
/// the widget has been unmounted.
mixin SafeSetStateMixin<T extends StatefulWidget> on State<T> {
  void setStateSafely(VoidCallback update) {
    if (!mounted) {
      return;
    }

    if (SchedulerBinding.instance.schedulerPhase ==
        SchedulerPhase.persistentCallbacks) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        setState(update);
      });
      return;
    }

    setState(update);
  }
}
