import 'package:http/http.dart' as http;
import 'package:quark/models/cirrus_file_node.dart';
import 'package:quark/services/cirrus_service.dart';
import 'package:quark/utils/file_browser_path_utils.dart';

Future<void> uploadMultipartFilesToCurrentPath({
  required String currentPath,
  required List<http.MultipartFile> selectedFiles,
  String? serial,
}) {
  return CirrusService.uploadFilesFromFormData(
    toRootDir(currentPath),
    selectedFiles,
    serial: serial,
  );
}

Future<void> createFolderAtCurrentPath({
  required String currentPath,
  required String folderName,
}) {
  return CirrusService.createFolder(toRootDir(currentPath), folderName);
}

Future<String?> downloadNode({required CirrusFileNode node}) {
  final itemName = trimTrailingSlashes(node.name);
  final filePath = node.apiPath;

  return CirrusService.saveFile(
    filePath,
    serial: serialOrNull(node.deviceSerial),
    fileName: itemName,
  );
}

Future<void> moveRenameNode({
  required CirrusFileNode node,
  required String targetInput,
  String? newDeviceSerial,
}) {
  final basePath = parentPath(node.apiPath);
  final oldPath = normalizePath(node.apiPath);
  final targetPath = targetInput.startsWith('/')
      ? normalizePath(targetInput)
      : joinPath(basePath, targetInput);

  final serial = serialOrNull(node.deviceSerial);
  return CirrusService.moveFile(
    oldPath,
    targetPath,
    oldDeviceSerial: serial,
    newDeviceSerial: newDeviceSerial ?? serial,
  );
}

Future<void> extractNode({required CirrusFileNode node}) {
  return CirrusService.extractFile(
    node.apiPath,
    serial: serialOrNull(node.deviceSerial),
  );
}

Future<void> deleteNode({required CirrusFileNode node}) {
  final rootDir = toRootDir(parentPath(node.apiPath));
  return CirrusService.deleteFile(
    rootDir,
    trimTrailingSlashes(node.name),
    deviceSerial: serialOrNull(node.deviceSerial),
  );
}
