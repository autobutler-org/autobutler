import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/utils/connection_error.dart';

/// #1637: the disconnected state hinges entirely on this predicate. Calling a
/// server error "you're not connected" sends people to check their wifi over a
/// bug on the Quark; calling an unreachable Quark a server error is the raw
/// socket dump this replaced. Both directions are pinned here.
void main() {
  group('is unreachable', () {
    test('a refused connection, as package:http reports it', () {
      // What IOClient throws once it has folded the SocketException in.
      expect(
        isQuarkUnreachableError(
          http.ClientException(
            'Connection refused',
            Uri.parse('https://quark.local/api/v0/health'),
          ),
        ),
        isTrue,
      );
    });

    test("the browser's opaque network error", () {
      expect(
        isQuarkUnreachableError(
          http.ClientException(
            'XMLHttpRequest error.',
            Uri.parse('https://quark.local/api/v0/health'),
          ),
        ),
        isTrue,
      );
    });

    test('a request that was accepted and then never answered', () {
      expect(isQuarkUnreachableError(TimeoutException('no response')), isTrue);
    });
  });

  group('is not unreachable', () {
    test('a status code the Quark itself returned', () {
      // Every service raises a plain Exception for a non-2xx response.
      expect(
        isQuarkUnreachableError(Exception('Failed to fetch health (500)')),
        isFalse,
      );
    });

    test('an expired session', () {
      expect(isQuarkUnreachableError(const UnauthorizedException()), isFalse);
    });

    test('a response body that did not parse', () {
      expect(
        isQuarkUnreachableError(const FormatException('Invalid health format')),
        isFalse,
      );
    });

    test('a bug in the app', () {
      expect(isQuarkUnreachableError(StateError('bad state')), isFalse);
    });
  });
}
