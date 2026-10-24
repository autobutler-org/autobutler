import 'package:quark/utils/local_time_zone_io.dart'
    if (dart.library.js_interop) 'package:quark/utils/local_time_zone_web.dart';

/// The IANA name of the device's time zone, such as `America/Los_Angeles`, or
/// an empty string where the platform does not say.
///
/// A browser knows it. iOS and Android report it over the app's own platform
/// channel (#3732); desktop sends an empty name. The Quark records it with each
/// event and steps the event's occurrences in it, today when a repeating event
/// is moved. An event saved with an empty name, as every phone's was before
/// #3732, is stepped in UTC.
Future<String> localTimeZoneName() => localTimeZoneNamePlatform();
