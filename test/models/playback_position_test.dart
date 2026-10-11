import 'package:flutter_test/flutter_test.dart';
import 'package:quark/models/playback_position.dart';

void main() {
  test('reads a stored state', () {
    final state = PlaybackPosition.fromJson(const {
      'positionMs': 4000,
      'durationMs': 10000,
      'watched': true,
      'updatedAt': '2026-10-10T09:30:00Z',
    });

    expect(state.position, const Duration(seconds: 4));
    expect(state.duration, const Duration(seconds: 10));
    expect(state.watched, isTrue);
    expect(state.updatedAt, DateTime.utc(2026, 10, 10, 9, 30));
  });

  test('reads the zero state the Quark sends when nothing is stored', () {
    final state = PlaybackPosition.fromJson(const {
      'positionMs': 0,
      'durationMs': 0,
      'watched': false,
    });

    expect(state.position, Duration.zero);
    expect(state.duration, Duration.zero);
    expect(state.watched, isFalse);
    expect(state.updatedAt, isNull);
  });
}
