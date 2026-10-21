import 'package:flutter_test/flutter_test.dart';
import 'package:quark/models/legacy_vault.dart';

void main() {
  test('reads GET /vault/legacy entries, and fills what an entry leaves '
      'out (#3616)', () {
    final drive = LegacyVault.fromJson({
      'kind': 'drive',
      'path': '/media/usb/quark/data/vault.db',
      'readable': true,
      'entries': 3,
      'folders': 1,
      'deviceName': 'USB',
    });
    expect(drive.onDrive, isTrue);
    expect(drive.entries, 3);
    expect(drive.deviceName, 'USB');

    final unreadable = LegacyVault.fromJson({
      'kind': 'kept',
      'path': '/var/lib/quark/data/household-vault-1.db',
      'readable': false,
    });
    expect(unreadable.onDrive, isFalse);
    expect(unreadable.readable, isFalse);
    expect(unreadable.entries, 0);
    expect(unreadable.deviceName, '');
  });
}
