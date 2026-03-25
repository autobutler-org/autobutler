import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('CirrusFileNode.fromJson', () {
    test('parses a fully populated JSON payload', () {
      final json = <String, dynamic>{
        'name': 'photo.jpg',
        'size': 1024,
        'isDir': false,
        'deviceName': 'USB Drive',
        'devicePath': '/dev/sda1',
        'deviceSerial': 'ABC123',
        'dirPath': '/photos/vacation',
      };

      final node = CirrusFileNode.fromJson(json);

      expect(node.name, 'photo.jpg');
      expect(node.size, 1024);
      expect(node.isDir, false);
      expect(node.deviceName, 'USB Drive');
      expect(node.devicePath, '/dev/sda1');
      expect(node.deviceSerial, 'ABC123');
      expect(node.dirPath, '/photos/vacation');
    });

    test('uses snake_case fallback keys', () {
      final json = <String, dynamic>{
        'name': 'doc.pdf',
        'size': 500,
        'is_dir': false,
        'device_name': 'Internal',
        'device_path': '/dev/mmcblk0p2',
        'device_serial': '',
        'dir_path': '/documents',
      };

      final node = CirrusFileNode.fromJson(json);

      expect(node.isDir, false);
      expect(node.deviceName, 'Internal');
      expect(node.devicePath, '/dev/mmcblk0p2');
      expect(node.deviceSerial, '');
      expect(node.dirPath, '/documents');
    });

    test('defaults missing fields gracefully', () {
      final node = CirrusFileNode.fromJson(<String, dynamic>{});

      expect(node.name, '');
      expect(node.size, 0);
      expect(node.isDir, false);
      expect(node.deviceName, '');
      expect(node.devicePath, '');
      expect(node.deviceSerial, '');
      expect(node.dirPath, '');
    });

    test('parses size from string', () {
      final json = <String, dynamic>{'name': 'file.txt', 'size': '2048'};

      final node = CirrusFileNode.fromJson(json);
      expect(node.size, 2048);
    });

    test('parses size from double', () {
      final json = <String, dynamic>{'name': 'file.txt', 'size': 1024.5};

      final node = CirrusFileNode.fromJson(json);
      expect(node.size, 1024);
    });

    test('parses invalid size string as 0', () {
      final json = <String, dynamic>{
        'name': 'file.txt',
        'size': 'not-a-number',
      };

      final node = CirrusFileNode.fromJson(json);
      expect(node.size, 0);
    });

    test('parses isDir from string "true"', () {
      final json = <String, dynamic>{'name': 'folder', 'isDir': 'true'};

      final node = CirrusFileNode.fromJson(json);
      expect(node.isDir, true);
    });

    test('parses isDir from string "True" (case insensitive)', () {
      final json = <String, dynamic>{'name': 'folder', 'isDir': 'True'};

      final node = CirrusFileNode.fromJson(json);
      expect(node.isDir, true);
    });

    test('parses isDir from string "false"', () {
      final json = <String, dynamic>{'name': 'file.txt', 'isDir': 'false'};

      final node = CirrusFileNode.fromJson(json);
      expect(node.isDir, false);
    });
  });

  group('CirrusFileNode.apiPath', () {
    test('returns dirPath with leading/trailing slashes stripped', () {
      const node = CirrusFileNode(
        name: 'file.txt',
        size: 0,
        isDir: false,
        deviceName: '',
        devicePath: '',
        deviceSerial: '',
        dirPath: '/photos/vacation/',
      );

      expect(node.apiPath, 'photos/vacation');
    });

    test('falls back to name when dirPath is empty', () {
      const node = CirrusFileNode(
        name: 'readme.md',
        size: 0,
        isDir: false,
        deviceName: '',
        devicePath: '',
        deviceSerial: '',
        dirPath: '',
      );

      expect(node.apiPath, 'readme.md');
    });

    test('falls back to name when dirPath is whitespace', () {
      const node = CirrusFileNode(
        name: 'readme.md',
        size: 0,
        isDir: false,
        deviceName: '',
        devicePath: '',
        deviceSerial: '',
        dirPath: '   ',
      );

      expect(node.apiPath, 'readme.md');
    });

    test('strips multiple leading slashes', () {
      const node = CirrusFileNode(
        name: '',
        size: 0,
        isDir: true,
        deviceName: '',
        devicePath: '',
        deviceSerial: '',
        dirPath: '///deep/path///',
      );

      expect(node.apiPath, 'deep/path');
    });
  });
}
