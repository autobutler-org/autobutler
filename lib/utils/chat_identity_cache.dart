import 'package:quark/utils/chat_identity_cache_io.dart'
    if (dart.library.js_interop) 'package:quark/utils/chat_identity_cache_web.dart'
    as platform;

/// Where the unlocked chat identity is kept between app starts (#2416), so
/// chat does not ask for the password again.
///
/// Native keeps it in the platform keystore, which outlives a restart. The
/// web keeps it in the tab's `sessionStorage` (#3257): a reload stays
/// unlocked, while closing the tab, or opening a new one, asks again.
/// Signing out deletes it either way.
Future<String?> readChatIdentityCache(String key) => platform.read(key);

/// Stores [value] under [key]; see [readChatIdentityCache].
Future<void> writeChatIdentityCache(String key, String value) =>
    platform.write(key, value);

/// Forgets [key]; see [readChatIdentityCache].
Future<void> deleteChatIdentityCache(String key) => platform.delete(key);
