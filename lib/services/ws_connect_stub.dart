// Stub for non-io platforms (web). Returns a plain WebSocketChannel.
import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel connectLocalTrustWs(Uri uri) => WebSocketChannel.connect(uri);
