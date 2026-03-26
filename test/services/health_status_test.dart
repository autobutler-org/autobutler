import 'package:autobutler/services/health_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('HealthStatus.fromJson', () {
    test('parses a complete JSON payload', () {
      final json = {
        'healthy': true,
        'alerts': ['disk usage high'],
        'cpuPercent': 42.5,
        'cpuCorePercents': [40.0, 45.0, 41.0, 43.0],
        'memPercent': 65.3,
        'memUsedBytes': 4294967296,
        'memTotalBytes': 8589934592,
        'diskPercent': 72.1,
        'diskUsedBytes': 100000000000,
        'diskTotalBytes': 250000000000,
        'temperatureCelsius': 55.2,
        'hostname': 'openclaw',
      };

      final status = HealthStatus.fromJson(json);

      expect(status.healthy, isTrue);
      expect(status.alerts, ['disk usage high']);
      expect(status.cpuPercent, 42.5);
      expect(status.cpuCorePercents, [40.0, 45.0, 41.0, 43.0]);
      expect(status.memPercent, 65.3);
      expect(status.memUsedBytes, 4294967296);
      expect(status.memTotalBytes, 8589934592);
      expect(status.diskPercent, 72.1);
      expect(status.diskUsedBytes, 100000000000);
      expect(status.diskTotalBytes, 250000000000);
      expect(status.temperatureCelsius, 55.2);
      expect(status.hostname, 'openclaw');
    });

    test('applies defaults for missing fields', () {
      final status = HealthStatus.fromJson({});

      expect(status.healthy, isTrue);
      expect(status.alerts, isEmpty);
      expect(status.cpuPercent, 0);
      expect(status.cpuCorePercents, isEmpty);
      expect(status.memPercent, 0);
      expect(status.memUsedBytes, 0);
      expect(status.memTotalBytes, 0);
      expect(status.diskPercent, 0);
      expect(status.diskUsedBytes, 0);
      expect(status.diskTotalBytes, 0);
      expect(status.temperatureCelsius, 0);
      expect(status.hostname, '');
    });

    test('handles integer values for floating-point fields', () {
      final json = {
        'cpuPercent': 50,
        'memPercent': 70,
        'diskPercent': 80,
        'temperatureCelsius': 60,
      };

      final status = HealthStatus.fromJson(json);

      expect(status.cpuPercent, 50.0);
      expect(status.memPercent, 70.0);
      expect(status.diskPercent, 80.0);
      expect(status.temperatureCelsius, 60.0);
    });

    test('healthy defaults to true when null', () {
      final status = HealthStatus.fromJson({'healthy': null});
      expect(status.healthy, isTrue);
    });

    test('parses empty alerts list', () {
      final status = HealthStatus.fromJson({'alerts': []});
      expect(status.alerts, isEmpty);
    });
  });
}
