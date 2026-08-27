import 'package:quark/utils/upload_unload_guard_io.dart'
    if (dart.library.js_interop) 'package:quark/utils/upload_unload_guard_web.dart'
    as platform;

/// Asks the browser to warn before the page is closed while uploads are
/// running.
///
/// Reloading the tab really does end an upload — the bytes are being read and
/// sent by this page — so that one case earns a prompt. Everything short of
/// it (navigating between folders, an auto-refresh) must not touch the
/// upload at all, which is the UploadManager's job rather than this one's.
///
/// A no-op off the web, where nothing can pull the process out from under us.
void setUploadUnloadGuard({required bool active}) {
  platform.setUploadUnloadGuardPlatform(active: active);
}
