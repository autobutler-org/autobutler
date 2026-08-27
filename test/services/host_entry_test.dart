import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/app_settings.dart';

void main() {
  group('HostEntry', () {
    test('toJson returns correct map', () {
      final entry = HostEntry(
        name: 'Local',
        hostAddress: 'http://localhost:8080',
      );
      final json = entry.toJson();

      expect(json, {'name': 'Local', 'hostAddress': 'http://localhost:8080'});
    });

    test('fromJson parses name and hostAddress', () {
      final entry = HostEntry.fromJson({
        'name': 'Remote',
        'hostAddress': 'https://butler.example.com',
      });

      expect(entry.name, 'Remote');
      expect(entry.hostAddress, 'https://butler.example.com');
    });

    test('fromJson defaults missing fields to empty strings', () {
      final entry = HostEntry.fromJson({});

      expect(entry.name, '');
      expect(entry.hostAddress, '');
    });

    test('fromJson handles null values gracefully', () {
      final entry = HostEntry.fromJson({'name': null, 'hostAddress': null});

      expect(entry.name, '');
      expect(entry.hostAddress, '');
    });

    test('roundtrip: toJson -> fromJson preserves values', () {
      final original = HostEntry(
        name: 'Pi',
        hostAddress: 'http://192.168.1.100:80',
      );
      final restored = HostEntry.fromJson(original.toJson());

      expect(restored.name, original.name);
      expect(restored.hostAddress, original.hostAddress);
    });

    test('fromJson with extra keys ignores them', () {
      final entry = HostEntry.fromJson({
        'name': 'Test',
        'hostAddress': 'http://test.local',
        'extraField': 42,
        'anotherOne': true,
      });

      expect(entry.name, 'Test');
      expect(entry.hostAddress, 'http://test.local');
    });
  });

  group('normalizeHostAddress', () {
    test('prepends https:// to a bare hostname', () {
      expect(
        normalizeHostAddress('brandons-macbook-pro-2.local'),
        'https://brandons-macbook-pro-2.local',
      );
    });

    test('prepends https:// to a bare host:port', () {
      expect(
        normalizeHostAddress('quark.home.local:8443'),
        'https://quark.home.local:8443',
      );
    });

    test('prepends https:// to a bare IP address', () {
      expect(normalizeHostAddress('192.168.1.100'), 'https://192.168.1.100');
    });

    test('leaves an explicit https:// address untouched', () {
      expect(
        normalizeHostAddress('https://quark.home.local'),
        'https://quark.home.local',
      );
    });

    test('leaves an explicit http:// address untouched', () {
      expect(
        normalizeHostAddress('http://quark.home.local'),
        'http://quark.home.local',
      );
    });

    test('leaves a non-http scheme untouched', () {
      expect(
        normalizeHostAddress('ws://quark.home.local'),
        'ws://quark.home.local',
      );
    });

    test('leaves the origin-relative web default untouched', () {
      expect(normalizeHostAddress('/'), '/');
    });

    test('leaves an empty address empty', () {
      expect(normalizeHostAddress(''), '');
      expect(normalizeHostAddress('   '), '');
    });

    test('trims surrounding whitespace before adding the scheme', () {
      expect(
        normalizeHostAddress('  quark.home.local  '),
        'https://quark.home.local',
      );
    });

    test('is idempotent', () {
      const bare = 'quark.home.local';
      expect(
        normalizeHostAddress(normalizeHostAddress(bare)),
        normalizeHostAddress(bare),
      );
    });
  });
}
