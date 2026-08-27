import 'package:desktop_drop/desktop_drop.dart';
import 'package:http/http.dart' as http;
import 'package:quark/services/upload_chunk_source_io.dart'
    if (dart.library.js_interop) 'package:quark/services/upload_chunk_source_web.dart'
    as platform;

/// One file, sent a byte range at a time and never held whole.
///
/// Reading and sending are one operation rather than two because on the web
/// they cannot be separated: `package:http`'s browser client materializes
/// whatever body it is handed, so slicing a `Blob` and then passing the slice
/// through `http` puts the bytes in the heap anyway. The only way a browser
/// streams a request is to hand `fetch` the `Blob` itself, which means the
/// thing that owns the `Blob` has to be the thing that sends it (#1629).
abstract class UploadChunkSource {
  /// Total bytes in the file.
  int get size;

  /// Last write time, or null where the platform does not report one.
  ///
  /// Part of the identity a stored session is matched against — see
  /// [uploadFileIdentity].
  DateTime? get lastModified;

  /// Sends `[start, end)` to [uri] as the whole request body.
  ///
  /// The response is small JSON either way, so it comes back materialized;
  /// only the request is streamed.
  Future<http.Response> putRange({
    required Uri uri,
    required Map<String, String> headers,
    required int start,
    required int end,
  });

  /// Releases whatever handle the platform held open.
  ///
  /// Called once the file is done or abandoned. A no-op is a valid
  /// implementation.
  void release() {}
}

/// A chunk source for a file that arrived by drag and drop, or null when the
/// platform cannot read it a range at a time.
///
/// Drops only reach this on the web today, where the item is an object URL
/// standing in for a `Blob` the browser is holding on disk. Fetching that URL
/// hands back the `Blob` itself rather than a copy of its bytes.
Future<UploadChunkSource?> openDroppedFileChunkSource(DropItemFile file) {
  return platform.openDroppedFileChunkSourcePlatform(file);
}
