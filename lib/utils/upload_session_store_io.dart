import 'package:quark/utils/upload_session_record.dart';

/// Native has no page reload to survive, so records only need to outlive a
/// retry rather than a process.
final UploadSessionStore uploadSessionStorePlatform =
    InMemoryUploadSessionStore();
