/// What `AppSettings` stores in place of a session token when the browser
/// holds the session in the Quark's HttpOnly `session` cookie (#2156).
///
/// It only says "signed in": it is never sent as a credential, and it is
/// worthless to a script that reads it out of localStorage.
const String cookieSessionMarker = 'cookie';

/// Whether [hostAddress] is served from the same origin as the page at
/// [base], so the browser sends that page's cookies with a request to it.
///
/// [hostAddress] is resolved against [base] first, which makes the web
/// build's origin-relative `/` same-origin. An address that cannot be parsed
/// is not.
bool isSameOrigin(String hostAddress, Uri base) {
  final address = Uri.tryParse(hostAddress.trim());
  if (address == null) return false;
  final resolved = base.resolveUri(address);
  return resolved.scheme == base.scheme &&
      resolved.host == base.host &&
      resolved.port == base.port;
}
