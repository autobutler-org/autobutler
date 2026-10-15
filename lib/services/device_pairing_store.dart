/// This phone's remote-access pairing and its pause, kept in platform secure
/// storage and keyed by the Quark's home address (#2876).
///
/// Web builds export the stub, which stores nothing. A browser does not run
/// a node, and this file never calls `dart:io`.
library;

import 'dart:convert';

import 'package:quark/services/device_pairing_store_stub.dart'
    if (dart.library.io) 'package:quark/services/device_pairing_store_io.dart'
    as impl;

/// How long this phone has paused its own remote access.
enum DevicePauseChoice {
  /// Turns itself back on after an hour.
  hour,

  /// Turns itself back on once the app reaches the Quark on the home network.
  untilHome,

  /// Stays off until the person turns it back on.
  untilManual,
}

/// The key, coordination server, and Quark address a phone joins with.
class StoredPairing {
  /// Creates a pairing saved for one Quark.
  const StoredPairing({
    required this.authKey,
    required this.controlUrl,
    required this.quarkAddress,
  });

  /// The single-use key from the Quark.
  final String authKey;

  /// The coordination server the phone registers with.
  final String controlUrl;

  /// The Quark's own address, which this phone's node dials.
  final String quarkAddress;
}

/// A pause that survives an app restart.
class StoredPause {
  /// Creates a pause. [until] is set only for [DevicePauseChoice.hour].
  const StoredPause({required this.choice, this.until});

  /// Which of the three pause choices this is.
  final DevicePauseChoice choice;

  /// When an hour-long pause ends, or null for the other choices.
  final DateTime? until;
}

String _token(String host) => base64Url.encode(utf8.encode(host));

String _pairingKey(String host) => 'quark.devicePairing.${_token(host)}';

String _pauseKey(String host) => 'quark.devicePause.${_token(host)}';

/// Reads and writes [StoredPairing] and [StoredPause] for one Quark address.
abstract final class DevicePairingStore {
  /// The pairing saved for [host], or null when this phone has not paired.
  static Future<StoredPairing?> readPairing(String host) async {
    final raw = await impl.readValue(_pairingKey(host));
    if (raw == null || raw.isEmpty) return null;
    try {
      final json = jsonDecode(raw);
      if (json is! Map) return null;
      final authKey = json['authKey'] as String? ?? '';
      final controlUrl = json['controlUrl'] as String? ?? '';
      final quarkAddress = json['quarkAddress'] as String? ?? '';
      if (authKey.isEmpty || controlUrl.isEmpty || quarkAddress.isEmpty) {
        return null;
      }
      return StoredPairing(
        authKey: authKey,
        controlUrl: controlUrl,
        quarkAddress: quarkAddress,
      );
    } catch (_) {
      return null;
    }
  }

  /// Saves [pairing] for [host], replacing whatever was there.
  static Future<void> savePairing(String host, StoredPairing pairing) =>
      impl.writeValue(
        _pairingKey(host),
        jsonEncode({
          'authKey': pairing.authKey,
          'controlUrl': pairing.controlUrl,
          'quarkAddress': pairing.quarkAddress,
        }),
      );

  /// Forgets the pairing for [host]. The next time at home pairs again.
  static Future<void> clearPairing(String host) =>
      impl.deleteValue(_pairingKey(host));

  /// The pause saved for [host], or null when remote access is not paused.
  static Future<StoredPause?> readPause(String host) async {
    final raw = await impl.readValue(_pauseKey(host));
    if (raw == null || raw.isEmpty) return null;
    try {
      final json = jsonDecode(raw);
      if (json is! Map) return null;
      final choice = switch (json['choice']) {
        'hour' => DevicePauseChoice.hour,
        'home' => DevicePauseChoice.untilHome,
        'manual' => DevicePauseChoice.untilManual,
        _ => null,
      };
      if (choice == null) return null;
      final untilRaw = json['until'];
      final until = untilRaw is String ? DateTime.tryParse(untilRaw) : null;
      return StoredPause(choice: choice, until: until);
    } catch (_) {
      return null;
    }
  }

  /// Saves [pause] for [host].
  static Future<void> savePause(String host, StoredPause pause) =>
      impl.writeValue(
        _pauseKey(host),
        jsonEncode({
          'choice': switch (pause.choice) {
            DevicePauseChoice.hour => 'hour',
            DevicePauseChoice.untilHome => 'home',
            DevicePauseChoice.untilManual => 'manual',
          },
          'until': ?pause.until?.toUtc().toIso8601String(),
        }),
      );

  /// Forgets the pause for [host].
  static Future<void> clearPause(String host) =>
      impl.deleteValue(_pauseKey(host));
}
