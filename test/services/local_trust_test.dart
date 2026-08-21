import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/local_trust.dart';

void main() {
  group('isLocalTrustHost', () {
    test('trusts mDNS .local hostnames', () {
      expect(isLocalTrustHost('openclaw.local'), isTrue);
      expect(isLocalTrustHost('quark.home.local'), isTrue);
      expect(isLocalTrustHost('OpenClaw.Local'), isTrue);
    });

    test('trusts other private-network suffixes', () {
      expect(isLocalTrustHost('butler.lan'), isTrue);
      expect(isLocalTrustHost('butler.home'), isTrue);
      expect(isLocalTrustHost('butler.home.arpa'), isTrue);
      expect(isLocalTrustHost('butler.internal'), isTrue);
    });

    test('trusts single-label hostnames', () {
      expect(isLocalTrustHost('openclaw'), isTrue);
      expect(isLocalTrustHost('localhost'), isTrue);
    });

    test('trusts RFC 1918 IPv4 ranges', () {
      expect(isLocalTrustHost('192.168.1.10'), isTrue);
      expect(isLocalTrustHost('10.0.2.2'), isTrue);
      expect(isLocalTrustHost('10.1.2.3'), isTrue);
      expect(isLocalTrustHost('172.16.0.1'), isTrue);
      expect(isLocalTrustHost('172.31.255.254'), isTrue);
      expect(isLocalTrustHost('127.0.0.1'), isTrue);
      expect(isLocalTrustHost('169.254.1.1'), isTrue);
    });

    test('rejects public IPv4 near the private ranges', () {
      expect(isLocalTrustHost('172.15.0.1'), isFalse);
      expect(isLocalTrustHost('172.32.0.1'), isFalse);
      expect(isLocalTrustHost('192.169.1.1'), isFalse);
      expect(isLocalTrustHost('11.0.0.1'), isFalse);
      expect(isLocalTrustHost('8.8.8.8'), isFalse);
    });

    test('trusts loopback, link-local, and unique-local IPv6', () {
      expect(isLocalTrustHost('::1'), isTrue);
      expect(isLocalTrustHost('fe80::1'), isTrue);
      expect(isLocalTrustHost('fe80::1%en0'), isTrue);
      expect(isLocalTrustHost('fd00::1'), isTrue);
      expect(isLocalTrustHost('fc00::1'), isTrue);
    });

    test('rejects public IPv6', () {
      expect(isLocalTrustHost('2001:4860:4860::8888'), isFalse);
    });

    test('rejects public hostnames', () {
      expect(isLocalTrustHost('example.com'), isFalse);
      expect(isLocalTrustHost('quark.io'), isFalse);
      // A public host that merely contains "local" is not a .local host.
      expect(isLocalTrustHost('local.example.com'), isFalse);
      expect(isLocalTrustHost('mylocal.com'), isFalse);
    });

    test('rejects empty and null hosts', () {
      expect(isLocalTrustHost(null), isFalse);
      expect(isLocalTrustHost(''), isFalse);
    });

    test('matches the host parsed out of a configured URL', () {
      expect(
        isLocalTrustHost(Uri.parse('https://openclaw.local:80').host),
        isTrue,
      );
      expect(isLocalTrustHost(Uri.parse('https://example.com').host), isFalse);
    });
  });
}
