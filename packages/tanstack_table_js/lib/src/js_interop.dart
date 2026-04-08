// Conditional platform facade for JS interop.
// ignore_for_file: unused_import

// On web this imports the web implementation, otherwise a stub that throws.
import 'js_interop_stub.dart'
  if (dart.library.html) 'js_interop_web.dart';
