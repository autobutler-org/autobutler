import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// A single file-system mutation event from the server.
class FileEvent {
  final String kind;
  final String path;
  final String? newPath;

  const FileEvent({required this.kind, required this.path, this.newPath});

  factory FileEvent.fromJson(Map<String, dynamic> json) => FileEvent(
    kind: json['kind'] as String? ?? '',
    path: json['path'] as String? ?? '',
    newPath: json['newPath'] as String?,
  );
}

/// Maintains a WebSocket connection to `GET /api/v1/events` and broadcasts
/// incoming [FileEvent]s to listeners.
///
/// Usage:
/// ```dart
/// final sub = EventsService.instance.events.listen((evt) {
///   if (evt.kind == 'upload' || evt.kind == 'delete') {
///     refresh();
///   }
/// });
/// // on dispose:
/// sub.cancel();
/// ```
class EventsService {
  EventsService._();
  static final EventsService instance = EventsService._();

  final _controller = StreamController<FileEvent>.broadcast();

  Stream<FileEvent> get events => _controller.stream;

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _sub;
  Timer? _reconnectTimer;
  bool _disposed = false;

  /// Start the WebSocket connection. Safe to call multiple times — no-ops if
  /// already connected. Call [stop] first if you want to force a reconnect.
  void start() {
    _disposed = false;
    if (_channel != null) return;
    _connect();
  }

  /// Stop the connection and cancel any pending reconnect.
  void stop() {
    _disposed = true;
    _reconnectTimer?.cancel();
    _sub?.cancel();
    _channel?.sink.close();
    _channel = null;
  }

  void _connect() {
    if (_disposed) return;
    final host = AppSettings.instance.activeHost;
    if (host == null) {
      // No host configured — don't schedule reconnect; caller must call start()
      // again once a host is set (e.g. from AppSettings change listener).
      return;
    }

    try {
      final httpUri = Uri.parse(host).resolve('/api/v1/events');
      final token = AppSettings.instance.sessionToken;
      final wsUri = httpUri.replace(
        scheme: httpUri.scheme == 'https' ? 'wss' : 'ws',
        queryParameters: token != null ? {'token': token} : null,
      );
      // On native platforms, use IOWebSocketChannel with a custom HttpClient
      // that accepts self-signed certs from local hosts (localhost / LAN IPs).
      // Web uses the default channel since the browser handles cert trust.
      if (!kIsWeb) {
        final httpClient = HttpClient()
          ..badCertificateCallback = (cert, host, port) {
            return host == 'localhost' ||
                host == '127.0.0.1' ||
                host == '::1' ||
                RegExp(
                  r'^(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.)',
                ).hasMatch(host);
          };
        _channel = IOWebSocketChannel.connect(wsUri, customClient: httpClient);
      } else {
        _channel = WebSocketChannel.connect(wsUri);
      }

      _sub = _channel!.stream.listen(
        (data) {
          try {
            final json = jsonDecode(data as String) as Map<String, dynamic>;
            _controller.add(FileEvent.fromJson(json));
          } catch (_) {
            // Ignore malformed frames
          }
        },
        onError: (_) => _onDisconnect(),
        onDone: _onDisconnect,
        cancelOnError: false,
      );
      debugPrint('[EventsService] connected to $wsUri');
    } catch (e) {
      debugPrint('[EventsService] connect error: $e');
      _onDisconnect();
    }
  }

  void _onDisconnect() {
    _sub?.cancel();
    _channel = null;
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (_disposed) return;
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(const Duration(seconds: 5), _connect);
  }
}
