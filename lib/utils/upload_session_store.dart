import 'package:quark/utils/upload_session_record.dart';
import 'package:quark/utils/upload_session_store_io.dart'
    if (dart.library.js_interop) 'package:quark/utils/upload_session_store_web.dart'
    as platform;

/// The store this platform resumes uploads from.
///
/// One instance for the app: the upload manager is a singleton and the records
/// are keyed by file, so there is nothing to scope them to.
UploadSessionStore get uploadSessionStore =>
    platform.uploadSessionStorePlatform;
