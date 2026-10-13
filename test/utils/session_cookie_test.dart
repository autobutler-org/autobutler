import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/session_cookie.dart';

/// #2156: only a request to the page's own origin carries the Quark's
/// `session` cookie, so only that host may go without a stored token.
void main() {
  final base = Uri.parse('https://quark.local/files/photos');

  test('the origin-relative address the web build stores is same-origin', () {
    expect(isSameOrigin('/', base), isTrue);
  });

  test('an absolute address of the same origin is same-origin', () {
    expect(isSameOrigin('https://quark.local', base), isTrue);
    expect(isSameOrigin('https://Quark.local:443/', base), isTrue);
  });

  test('a different port is another origin', () {
    expect(isSameOrigin('https://quark.local:8443', base), isFalse);
    // The dev flow: the Flutter dev server's page against the backend.
    expect(
      isSameOrigin('http://localhost:8080', Uri.parse('http://localhost:5555')),
      isFalse,
    );
  });

  test('a different host is another origin', () {
    expect(isSameOrigin('https://other.local', base), isFalse);
  });

  test('a different scheme is another origin', () {
    expect(isSameOrigin('http://quark.local', base), isFalse);
  });

  test('an address that cannot be parsed is not same-origin', () {
    expect(isSameOrigin('https://[', base), isFalse);
  });
}
