import 'dart:async';

import 'package:http/http.dart' as http;

/// Whether [error] means the app never got an answer from the Quark.
///
/// Distinguishes "I have no route to that address" from "the Quark answered
/// and said no". Only the first deserves the disconnected state and its
/// troubleshooting copy — labelling a real server error "you're not connected"
/// is the same class of mistake as calling a TLS failure a codec problem
/// (#1627).
///
/// Every HTTP call in the app goes out through `package:http`, which folds the
/// transport-level failures into [http.ClientException]: on native, `IOClient`
/// converts `SocketException` (DNS miss, connection refused, no route to host)
/// and `HttpException` (connection dropped mid-response) into one; on web,
/// `BrowserClient` reports every network-layer failure as an opaque
/// `XMLHttpRequest error`. A request the Quark actually served comes back as a
/// status code and is thrown by the calling service as a plain [Exception],
/// which is not matched here.
///
/// [TimeoutException] counts too: a Quark that accepts a connection and then
/// never replies — a half-open link on a network you have partially left — is
/// unreachable in every sense the user cares about.
bool isQuarkUnreachableError(Object error) =>
    error is http.ClientException || error is TimeoutException;
