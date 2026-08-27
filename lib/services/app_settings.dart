import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
// flutter_secure_storage requires a secure context (HTTPS) on web.
// We only use it on native platforms; on web we fall back to in-memory only.
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:shared_preferences/shared_preferences.dart';

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

  /// Notifies listeners whenever [activeHost] changes — a host added, edited,
  /// removed, or selected.
  ///
  /// The router listens to this so the terms/login gate re-runs the moment a
  /// Quark is connected for the first time. Without it the redirect only fired
  /// on the next unrelated navigation, so terms showed up late (#1623).
  final ValueNotifier<String?> activeHostNotifier = ValueNotifier(null);
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
      _hosts = decoded
          .map((e) => HostEntry.fromJson(e as Map<String, dynamic>))
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
        // Use https:// so the Flutter client exercises the TLS path even in
        // debug mode. The self-signed cert is trusted via badCertificateCallback
        // in AuthenticatedService for local/LAN addresses.
        var loopback = 'http://localhost:8080';
        if (!kIsWeb && defaultTargetPlatform == TargetPlatform.android) {
          loopback = 'http://10.0.2.2:8080';
        }
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

    _publishActiveHost();
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

  /// Publishes the current [activeHost] to [activeHostNotifier].
  /// Every mutation of [_hosts] or [_activeIndex] must end with this call.
  void _publishActiveHost() {
    activeHostNotifier.value = activeHost;
  }

  Future<void> _saveHosts() async {
    await _prefs?.setString(
      'hosts',
      jsonEncode(_hosts.map((e) => e.toJson()).toList()),
    );
    await _prefs?.setInt('activeHostIndex', _activeIndex);
    _publishActiveHost();
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

  /// Ensures the host address has a scheme — accepts both http:// and https://.
  /// Bare hostnames default to https://.
  HostEntry _normalizeHost(HostEntry h) {
    final addr = h.hostAddress.trim();
    if (addr.startsWith('https://') || addr.startsWith('http://')) {
      return h;
    }
    // No scheme — prepend https://
    return HostEntry(name: h.name, hostAddress: 'https://$addr');
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
      _publishActiveHost();
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
