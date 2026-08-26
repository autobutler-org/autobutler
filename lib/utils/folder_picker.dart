import 'package:quark/utils/folder_picker_io.dart'
    if (dart.library.js_interop) 'package:quark/utils/folder_picker_web.dart'
    as platform;
import 'package:quark/utils/upload_tree_utils.dart';

/// Whether this platform can offer folder selection.
///
/// Web and desktop can. Mobile has no meaningful folder picker, so callers
/// hide the affordance rather than offering one that cannot work.
bool get isFolderPickerSupported => platform.isFolderPickerSupportedPlatform;

/// Prompts for a folder and returns its files, each carrying the directory it
/// sat in relative to the chosen folder.
///
/// Returns an empty list when the user cancels or the folder holds no files.
Future<List<PendingUpload>> pickFolderUploads() {
  return platform.pickFolderUploadsPlatform();
}
