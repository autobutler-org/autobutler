import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
// flutter_secure_storage requires a secure context (HTTPS) on web.
// We only use it on native platforms; on web we fall back to in-memory only.
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Matches an explicit URI scheme prefix (`https://`, `http://`, `ws://`, ...).
///
/// Requires the `://` so a schemeless `host:port` is not mistaken for a scheme
/// — `Uri.parse('quark.local:8080')` reads `quark.local` as the scheme, which
/// is exactly the misparse this normalization exists to prevent.
final _schemePrefix = RegExp(r'^[a-zA-Z][a-zA-Z0-9+.\-]*://');

/// Ensures a host address carries an explicit scheme, defaulting to `https://`.
///
/// A quark serves TLS, so a bare hostname must resolve to `https://` — without
/// a scheme `Uri.parse` yields a path-only URI with no authority and the
/// request silently degrades to plain HTTP against port 80.
///
/// An address that already names a scheme is returned untouched: an explicit
/// `http://` the user typed stays `http://`. Empty and origin-relative
/// addresses (the web build stores `/`) are left alone — they have no host.
String normalizeHostAddress(String address) {
  final trimmed = address.trim();
  if (trimmed.isEmpty || trimmed.startsWith('/')) return trimmed;
  if (_schemePrefix.hasMatch(trimmed)) return trimmed;
  return 'https://$trimmed';
}

/// Base URL used when no host has been configured yet.
///
/// Matches the plain-HTTP dev target (`make serve/backend`), which serves
/// :8080 in insecure mode. Only ever reached when [AppSettings.activeHost] is
/// null — a configured host always wins.
const String defaultApiBaseUrl = 'http://localhost:8080';

/// The configured quark base URL, falling back to [defaultApiBaseUrl].
///
/// `API_BASE_URL` overrides the fallback at build time via
/// `--dart-define=API_BASE_URL=...`.
String get apiBaseUrl =>
    AppSettings.instance.activeHost ??
    const String.fromEnvironment(
      'API_BASE_URL',
      defaultValue: defaultApiBaseUrl,
    );

/// [apiBaseUrl] as a [Uri], with the Android emulator's loopback alias applied.
///
/// The emulator reaches the host machine at 10.0.2.2 rather than localhost, so
/// a loopback address is rewritten before any request goes out.
Uri get apiBaseUri {
  final uri = Uri.parse(apiBaseUrl);
  final isLoopback =
      uri.host == 'localhost' || uri.host == '127.0.0.1' || uri.host == '::1';
  if (!kIsWeb &&
      defaultTargetPlatform == TargetPlatform.android &&
      isLoopback) {
    return uri.replace(host: '10.0.2.2');
  }
  return uri;
}

class HostEntry {
  final String name;
  final String hostAddress;
  HostEntry({required this.name, required this.hostAddress});

  Map<String, String> toJson() => {'name': name, 'hostAddress': hostAddress};

  static HostEntry fromJson(Map<String, dynamic> m) =>
      HostEntry(name: m['name'] ?? '', hostAddress: m['hostAddress'] ?? '');
}

class AppSettings {
  AppSettings._();
  static final AppSettings instance = AppSettings._();

  final ValueNotifier<ThemeMode> themeMode = ValueNotifier(ThemeMode.system);

  List<HostEntry> _hosts = [];
  int _activeIndex = -1;
  String? _sessionToken;
  SharedPreferences? _prefs;

  /// Notifies listeners whenever the session token changes.
  /// The router listens to this to redirect to login when a 401 clears the token.
  final ValueNotifier<String?> sessionTokenNotifier = ValueNotifier(null);

  /// Notifies listeners whenever the terms acceptance state changes.
  /// The router listens to this to redirect to the terms page when not yet accepted.
  final ValueNotifier<bool> hasAcceptedTerms = ValueNotifier(false);
  final FlutterSecureStorage _secureStorage = const FlutterSecureStorage();

  static const _sessionTokenKey = 'session_token';

  Future<void> load() async {
    _prefs = await SharedPreferences.getInstance();
    final theme = _prefs!.getString('themeMode') ?? 'system';
    themeMode.value = theme == 'light'
        ? ThemeMode.light
        : theme == 'dark'
        ? ThemeMode.dark
        : ThemeMode.system;

    final hostsJson = _prefs!.getString('hosts') ?? '[]';
    try {
      final decoded = jsonDecode(hostsJson) as List;
      // Normalize on load, not just on add/update: an entry persisted by an
      // older build (or any path that skipped normalization) would otherwise
      // stay schemeless forever and keep resolving to plain HTTP.
      _hosts = decoded
          .map(
            (e) =>
                _normalizeHost(HostEntry.fromJson(e as Map<String, dynamic>)),
          )
          .toList();
    } catch (_) {
      debugPrint('[app_settings.dart] Error in catch block');
      _hosts = [];
    }

    _activeIndex =
        _prefs!.getInt('activeHostIndex') ?? (_hosts.isEmpty ? -1 : 0);

    if (kIsWeb) {
      // On web, use shared_preferences (localStorage) — flutter_secure_storage
      // requires HTTPS and will fail over plain HTTP during development.
      _sessionToken = _prefs!.getString(_sessionTokenKey);
    } else {
      _sessionToken = await _secureStorage.read(key: _sessionTokenKey);
    }
    sessionTokenNotifier.value = _sessionToken;

    hasAcceptedTerms.value = _prefs!.getBool('hasAcceptedTerms') ?? false;

    // If no hosts configured and running in debug (local development), add a local loopback
    // host appropriate for the running platform so developers can quickly connect.
    if (_hosts.isEmpty) {
      if (kDebugMode) {
        // Targets the plain-HTTP dev server (`make serve/backend`), which
        // listens on :8080 only in insecure mode. Scheme and port move
        // together: :8080 is never served over TLS, so `https://localhost:8080`
        // would connect to nothing. To develop against the secure server
        // (`make serve/backend/secure`, TLS on :443) point this at
        // `https://localhost` instead — its self-signed cert is accepted by
        // badCertificateCallback in AuthenticatedService for local-trust hosts.
        final loopback =
            !kIsWeb && defaultTargetPlatform == TargetPlatform.android
            ? 'http://10.0.2.2:8080'
            : 'http://localhost:8080';
        _hosts = [HostEntry(name: 'Local', hostAddress: loopback)];
        _activeIndex = 0;
        await _saveHosts();
      } else if (kIsWeb) {
        // Otherwise, add a default that targets the URL it is the web version
        _hosts = [HostEntry(name: 'Default', hostAddress: '/')];
        _activeIndex = 0;
        await _saveHosts();
      }
    }
  }

  List<HostEntry> get hosts => List.unmodifiable(_hosts);
  int get activeIndex => _activeIndex;

  /// Session token set after a successful login or setup.
  /// Persisted via [FlutterSecureStorage] — survives app restarts.
  /// Populated by [AuthService] after login/setup; consumed by [FilesService].
  String? get sessionToken => _sessionToken;

  Future<void> setSessionToken(String? token) async {
    _sessionToken = token;
    sessionTokenNotifier.value = token;
    if (kIsWeb) {
      // Use shared_preferences (localStorage) on web — flutter_secure_storage
      // requires HTTPS and fails over plain HTTP in development.
      if (token != null) {
        await _prefs?.setString(_sessionTokenKey, token);
      } else {
        await _prefs?.remove(_sessionTokenKey);
      }
    } else {
      if (token != null) {
        await _secureStorage.write(key: _sessionTokenKey, value: token);
      } else {
        await _secureStorage.delete(key: _sessionTokenKey);
      }
    }
  }

  String? get activeHost => (_activeIndex >= 0 && _activeIndex < _hosts.length)
      ? _hosts[_activeIndex].hostAddress
      : null;

  Future<void> _saveHosts() async {
    await _prefs?.setString(
      'hosts',
      jsonEncode(_hosts.map((e) => e.toJson()).toList()),
    );
    await _prefs?.setInt('activeHostIndex', _activeIndex);
  }

  Future<void> addHost(HostEntry h) async {
    _hosts.add(_normalizeHost(h));
    _activeIndex = _hosts.length - 1;
    await _saveHosts();
  }

  Future<void> updateHost(int idx, HostEntry h) async {
    if (idx >= 0 && idx < _hosts.length) {
      _hosts[idx] = _normalizeHost(h);
      await _saveHosts();
    }
  }

  /// Ensures the host address has a scheme, defaulting bare hostnames to
  /// `https://`. See [normalizeHostAddress].
  HostEntry _normalizeHost(HostEntry h) {
    final normalized = normalizeHostAddress(h.hostAddress);
    if (normalized == h.hostAddress) return h;
    return HostEntry(name: h.name, hostAddress: normalized);
  }

  Future<void> removeHost(int idx) async {
    if (idx >= 0 && idx < _hosts.length) {
      _hosts.removeAt(idx);
      if (_activeIndex >= _hosts.length) {
        _activeIndex = _hosts.length - 1;
      }
      await _saveHosts();
    }
  }

  Future<void> setActiveIndex(int idx) async {
    if (idx >= 0 && idx < _hosts.length) {
      _activeIndex = idx;
      // Clear cached file listings — they belong to the previous host.
      FileBrowserCache.instance.clear();
      await _prefs?.setInt('activeHostIndex', _activeIndex);
    }
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    themeMode.value = mode;
    final key = mode == ThemeMode.light
        ? 'light'
        : mode == ThemeMode.dark
        ? 'dark'
        : 'system';
    await _prefs?.setString('themeMode', key);
  }

  /// Marks the Terms and Conditions as accepted and persists the decision.
  Future<void> acceptTerms() async {
    hasAcceptedTerms.value = true;
    await _prefs?.setBool('hasAcceptedTerms', true);
  }

  /// Auto-refresh interval in seconds. 0 = disabled.
  int get refreshIntervalSeconds =>
      _prefs?.getInt('refreshIntervalSeconds') ?? 15;

  Future<void> setRefreshIntervalSeconds(int seconds) async {
    await _prefs?.setInt('refreshIntervalSeconds', seconds);
  }
}
