class CirrusFileNode {
  const CirrusFileNode({
    required this.name,
    required this.size,
    required this.isDir,
    required this.deviceName,
    required this.devicePath,
    required this.fullPath,
    required this.deviceSerial,
    required this.dirPath,
  });

  final String name;
  final int size;
  final bool isDir;
  final String deviceName;
  final String devicePath;
  final String fullPath;
  final String deviceSerial;
  final String dirPath;

  factory CirrusFileNode.fromJson(Map<String, dynamic> json) {
    int parseSize(Object? value) {
      if (value is int) {
        return value;
      }
      if (value is num) {
        return value.toInt();
      }
      if (value is String) {
        return int.tryParse(value) ?? 0;
      }
      return 0;
    }

    bool parseBool(Object? value) {
      if (value is bool) {
        return value;
      }
      if (value is String) {
        return value.toLowerCase() == 'true';
      }
      return false;
    }

    String parseString(Object? value) {
      return value?.toString() ?? '';
    }

    return CirrusFileNode(
      name: parseString(json['name']),
      size: parseSize(json['size']),
      isDir: parseBool(json['isDir'] ?? json['is_dir']),
      deviceName: parseString(json['deviceName'] ?? json['device_name']),
      devicePath: parseString(json['devicePath'] ?? json['device_path']),
      fullPath: parseString(json['fullPath'] ?? json['full_path']),
      deviceSerial: parseString(json['deviceSerial'] ?? json['device_serial']),
      dirPath: parseString(json['dirPath'] ?? json['dir_path']),
    );
  }
}
