import 'package:autobutler/services/app_settings.dart';
import 'package:flutter_test/flutter_test.dart';

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
}
