// cspell:words reallocated celsius
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/drive_health_controller.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The Storage tab shows the verdict the Quark sent and asks before a
/// Replace soon drive is committed to. It does not score the drive itself.
void main() {
  StorageDevice device(Map<String, dynamic> json) =>
      StorageDevice.fromJson(json);

  test('a missing health field is unknown', () {
    final health = DriveHealthController.view(device({}).health);
    expect(health.verdict, DriveHealthVerdict.unknown);
    expect(health.reasonText, isNull);
    expect(health.hasDetail, isFalse);
  });

  test('an unrecognized verdict is unknown, never healthy', () {
    final health = DriveHealthController.view(
      device({
        'health': {'verdict': 'passed', 'reason': 'looks fine'},
      }).health,
    );
    expect(health.verdict, DriveHealthVerdict.unknown);
    expect(health.reasonText, 'looks fine');
  });

  test('reads the verdict and the readings the Quark sent', () {
    final report = device({
      'health': {
        'verdict': 'replace-soon',
        'reason': 'This spinning drive is heavily used.',
        'smartStatus': 'failed',
        'reallocatedSectors': 4,
        'pendingSectors': 1,
        'powerOnHours': 60000,
        'temperatureCelsius': 46,
        'media': 'spinning',
        'linkSpeed': 'USB 2.0',
        'worseThanDefault': true,
      },
      'diskProbe': {'speedClass': 'slow'},
    }).health!;

    expect(report.worseThanDefault, isTrue);
    expect(report.speedClass, 'slow');
    final health = DriveHealthController.view(report);
    expect(health.verdict, DriveHealthVerdict.replaceSoon);
    expect(health.reasonText, 'This spinning drive is heavily used.');
    expect(health.powerOnHoursText, '60000 hours');
    expect(health.temperatureText, '46°C');
    expect(health.mediaText, 'Spinning');
    expect(health.speedClassText, 'Slow');
    expect(health.reasonText, isNot(contains('year')));
  });

  test('replace soon asks for an acknowledgment', () {
    final prompt = DriveHealthController.prompt(
      verdict: DriveHealthVerdict.replaceSoon,
      action: DriveHealthAction.startBackup,
      worseThanDefault: true,
    );
    expect(prompt, isNotNull);
    expect(prompt!.sendAcknowledgment, isTrue);
    expect(prompt.strongerWarning, isFalse);
    expect(prompt.addsBackup, isTrue);
    expect(prompt.keepsExistingFiles, isTrue);
  });

  test('watch does not ask, and a worse default storage drive does', () {
    expect(
      DriveHealthController.prompt(
        verdict: DriveHealthVerdict.watch,
        action: DriveHealthAction.startBackup,
        worseThanDefault: true,
      ),
      isNull,
    );
    expect(
      DriveHealthController.prompt(
        verdict: DriveHealthVerdict.watch,
        action: DriveHealthAction.setBackupRole,
        worseThanDefault: true,
      ),
      isNull,
    );

    final prompt = DriveHealthController.prompt(
      verdict: DriveHealthVerdict.watch,
      action: DriveHealthAction.setDefaultStorage,
      worseThanDefault: true,
    );
    expect(prompt!.sendAcknowledgment, isFalse);
    expect(prompt.strongerWarning, isTrue);
    expect(prompt.addsBackup, isFalse);
    expect(prompt.keepsExistingFiles, isTrue);
  });

  test(
    'unknown on a worse default storage drive warns and does not acknowledge',
    () {
      final prompt = DriveHealthController.prompt(
        verdict: DriveHealthVerdict.unknown,
        action: DriveHealthAction.setDefaultStorage,
        worseThanDefault: true,
      );
      expect(prompt!.sendAcknowledgment, isFalse);
      expect(prompt.strongerWarning, isTrue);
    },
  );

  test('a healthy drive proceeds', () {
    expect(
      DriveHealthController.prompt(
        verdict: DriveHealthVerdict.healthy,
        action: DriveHealthAction.setDefaultStorage,
        worseThanDefault: false,
      ),
      isNull,
    );
  });

  test('replace soon default storage that is also worse sends both', () {
    final prompt = DriveHealthController.prompt(
      verdict: DriveHealthVerdict.replaceSoon,
      action: DriveHealthAction.setDefaultStorage,
      worseThanDefault: true,
    );
    expect(prompt!.sendAcknowledgment, isTrue);
    expect(prompt.strongerWarning, isTrue);
    expect(prompt.addsBackup, isFalse);
  });
}
