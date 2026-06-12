// IO implementation: trusts self-signed certs from local hosts.
import 'dart:io';

import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel connectLocalTrustWs(Uri uri) {
  final httpClient = HttpClient()
    ..badCertificateCallback = (cert, host, port) {
      return host == 'localhost' ||
          host == '127.0.0.1' ||
          host == '::1' ||
          RegExp(
            r'^(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.)',
          ).hasMatch(host);
    };
  return IOWebSocketChannel.connect(uri, customClient: httpClient);
}
