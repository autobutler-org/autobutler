// cspell:words ffi

/// The on-device tailnet node for iOS and Android (#1881).
///
/// The phone joins the tailnet itself, so the user does not install the
/// official Tailscale app. [EmbeddedTunnel] is the whole API: [start] returns
/// the port of a plain HTTP proxy on 127.0.0.1, [stop] shuts it down, and
/// [status] reports the node. The proxy dials the Quark. The app's remote
/// address becomes that proxy, and [ConnectionController] already sends every
/// request there. The proxy forwards only requests that carry the token it
/// hands this app on each start; `tunnel_token_client.dart` adds it.
///
/// Web builds export the stub, which reports the tunnel unavailable. A browser
/// cannot run a node.
library;

import 'package:quark/services/embedded_tunnel_stub.dart'
    if (dart.library.ffi) 'package:quark/services/embedded_tunnel_io.dart'
    as impl;

/// What the bridge reports about its node.
///
/// Parsed from the JSON `quark_tsnet_status` returns:
/// `{state, tailnetIPs, port, error}`.
class TunnelStatus {
  /// Creates a status value.
  const TunnelStatus({
    required this.state,
    this.tailnetIPs = const [],
    this.port = 0,
    this.error = '',
  });

  /// Parses the bridge's status JSON.
  factory TunnelStatus.fromJson(Map<String, dynamic> json) => TunnelStatus(
    state: json['state'] as String? ?? 'Stopped',
    tailnetIPs: [
      for (final ip in json['tailnetIPs'] as List? ?? const []) ip as String,
    ],
    port: json['port'] as int? ?? 0,
    error: json['error'] as String? ?? '',
  );

  /// `Stopped`, `Starting`, or the backend state (`Running`, `NeedsLogin`).
  final String state;

  /// This device's own tailnet addresses.
  final List<String> tailnetIPs;

  /// The proxy port, 0 when it is not listening.
  final int port;

  /// The last start failure, empty when there was none.
  ///
  /// Diagnostic text for a log, not copy for a user.
  final String error;

  /// Whether the node is up and the proxy is serving.
  bool get isRunning => state == 'Running' && port > 0;

  /// The base URL to point the app at, or null when the proxy is down.
  Uri? get loopbackUrl =>
      isRunning ? Uri.parse('http://127.0.0.1:$port') : null;
}

/// Thrown when [EmbeddedTunnel.start] fails.
///
/// [message] is the bridge's own diagnostic text, not copy for a user.
class EmbeddedTunnelException implements Exception {
  /// Creates the exception.
  const EmbeddedTunnelException(this.message);

  /// The bridge's error text.
  final String message;

  @override
  String toString() => 'EmbeddedTunnelException: $message';
}

/// The app's embedded tailnet node and its local proxy.
///
/// One per process. On iOS the node runs in the foreground only: [watchLifecycle]
/// restarts it when the app comes back. There is no background transfer.
abstract final class EmbeddedTunnel {
  /// Whether this platform loads the native bridge.
  static bool get isSupported => impl.tunnelIsSupported;

  /// Joins the tailnet and starts the proxy. Returns the proxy port.
  ///
  /// [upstream] is the Quark's tailnet address (`http://100.x.y.z`). An empty
  /// [authKey] rejoins a node that already has state in [stateDir].
  static Future<int> start({
    required String stateDir,
    required String controlUrl,
    required String authKey,
    required String hostname,
    required String upstream,
  }) async {
    final (port, error) = await impl.tunnelStart(
      stateDir: stateDir,
      controlUrl: controlUrl,
      authKey: authKey,
      hostname: hostname,
      upstream: upstream,
    );
    if (port < 0) {
      throw EmbeddedTunnelException(
        error.isEmpty ? 'the embedded tailnet did not start' : error,
      );
    }
    return port;
  }

  /// Stops the proxy and the node. Stopping a stopped node does nothing.
  static Future<void> stop() => impl.tunnelStop();

  /// The node's current state.
  static Future<TunnelStatus> status() async =>
      TunnelStatus.fromJson(await impl.tunnelStatusJson());

  /// The proxy URL in front of [upstream], or null when this platform has no
  /// node or the phone has not paired yet.
  ///
  /// [ConnectionController] probes this instead of the Quark's tailnet address.
  static Future<String?> proxyUrl(String upstream) =>
      impl.tunnelProxyUrl(upstream);

  /// Pairs, if this phone has no node yet, and starts the proxy.
  ///
  /// Called while the phone can still reach the Quark at home. An empty
  /// [upstream] means remote access is off, and a running node is stopped.
  static Future<void> warm(String upstream) => impl.tunnelWarm(upstream);

  /// Restarts the node when the app returns to the foreground, then calls
  /// [onReconnected] so the address check runs again. The port can change.
  static void watchLifecycle(void Function() onReconnected) =>
      impl.tunnelWatch(onReconnected);
}
