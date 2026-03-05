import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:http/http.dart' as http;

Future<void> uploadMultipartFileToCurrentPath({
  required String currentPath,
  required http.MultipartFile selectedFile,
}) {
  return CirrusService.uploadFilesFromFormData(toRootDir(currentPath), [
    selectedFile,
  ]);
}

Future<void> createFolderAtCurrentPath({
  required String currentPath,
  required String folderName,
}) {
  return CirrusService.createFolder(toRootDir(currentPath), folderName);
}

Future<String?> downloadNode({
  required String currentPath,
  required CirrusFileNode node,
}) {
  final itemName = trimTrailingSlashes(node.name);
  final filePath = toRootDir(joinPath(currentPath, itemName));

  return CirrusService.saveFile(
    filePath,
    serial: serialOrNull(node.deviceSerial),
    fileName: itemName,
  );
}

Future<void> moveRenameNode({
  required String currentPath,
  required CirrusFileNode node,
  required String targetInput,
}) {
  final itemName = trimTrailingSlashes(node.name);
  final oldPath = joinPath(currentPath, itemName);
  final targetPath = targetInput.startsWith('/')
      ? normalizePath(targetInput)
      : joinPath(currentPath, targetInput);

  final serial = serialOrNull(node.deviceSerial);
  return CirrusService.moveFile(
    oldPath,
    targetPath,
    oldDeviceSerial: serial,
    newDeviceSerial: serial,
  );
}

Future<void> deleteNode({
  required String currentPath,
  required CirrusFileNode node,
}) {
  return CirrusService.deleteFile(
    toRootDir(currentPath),
    trimTrailingSlashes(node.name),
    deviceSerial: serialOrNull(node.deviceSerial),
  );
}
