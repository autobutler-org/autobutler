import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
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
      _hosts = [];
    }

    _activeIndex =
        _prefs!.getInt('activeHostIndex') ?? (_hosts.isEmpty ? -1 : 0);

    _sessionToken = await _secureStorage.read(key: _sessionTokenKey);

    // If no hosts configured and running in debug (local development), add a local loopback
    // host appropriate for the running platform so developers can quickly connect.
    if (_hosts.isEmpty) {
      if (kDebugMode) {
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
  }

  List<HostEntry> get hosts => List.unmodifiable(_hosts);
  int get activeIndex => _activeIndex;

  /// Session token set after a successful login or setup.
  /// Persisted via [FlutterSecureStorage] — survives app restarts.
  /// Populated by [AuthService] after login/setup; consumed by [CirrusService].
  String? get sessionToken => _sessionToken;

  Future<void> setSessionToken(String? token) async {
    _sessionToken = token;
    if (token != null) {
      await _secureStorage.write(key: _sessionTokenKey, value: token);
    } else {
      await _secureStorage.delete(key: _sessionTokenKey);
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
    _hosts.add(h);
    _activeIndex = _hosts.length - 1;
    await _saveHosts();
  }

  Future<void> updateHost(int idx, HostEntry h) async {
    if (idx >= 0 && idx < _hosts.length) {
      _hosts[idx] = h;
      await _saveHosts();
    }
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
}
