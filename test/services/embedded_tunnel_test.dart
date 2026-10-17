import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/embedded_tunnel.dart';
import 'package:quark/services/embedded_tunnel_stub.dart';

/// The web stand-in, and the JSON the bridge returns mapped onto a URL.
void main() {
  test('the stub never starts a node', () async {
    expect(tunnelIsSupported, isFalse);
    expect(await tunnelProxyUrl('http://100.64.0.1'), isNull);
    await tunnelWarm('http://100.64.0.1');
    await tunnelWarm('');
    tunnelWatch(() => fail('the stub does not watch the app lifecycle'));
    final (port, error) = await tunnelStart(
      stateDir: '/none',
      controlUrl: 'https://example.invalid',
      authKey: '',
      hostname: 'phone',
      upstream: 'http://100.64.0.1',
    );
    expect(port, lessThan(0));
    expect(error, contains('unavailable'));
    await tunnelStop();
    expect(await tunnelStatusJson(), {
      'state': 'Stopped',
      'tailnetIPs': <String>[],
      'port': 0,
      'error': '',
    });
  });

  test('maps a running bridge onto the local proxy URL', () {
    final status = TunnelStatus.fromJson({
      'state': 'Running',
      'tailnetIPs': ['100.64.0.8'],
      'port': 43210,
      'error': '',
    });
    expect(status.isRunning, isTrue);
    expect(status.loopbackUrl, Uri.parse('http://127.0.0.1:43210'));
    expect(status.tailnetIPs, ['100.64.0.8']);
    expect(status.port, 43210);
    expect(status.error, isEmpty);
  });

  test('maps a stopped bridge and a failure without inventing a URL', () {
    final stopped = TunnelStatus.fromJson(const {});
    expect(stopped.state, 'Stopped');
    expect(stopped.isRunning, isFalse);
    expect(stopped.loopbackUrl, isNull);
    expect(stopped.tailnetIPs, isEmpty);
    expect(stopped.port, 0);

    final failed = TunnelStatus.fromJson({
      'state': 'NeedsLogin',
      'port': 9,
      'error': 'the tailnet rejected the auth key',
    });
    expect(failed.isRunning, isFalse);
    expect(failed.loopbackUrl, isNull);
    expect(failed.error, 'the tailnet rejected the auth key');
  });

  test('the public status on a host without the bridge is stopped', () async {
    expect(EmbeddedTunnel.isSupported, isFalse);
    final status = await EmbeddedTunnel.status();
    expect(status.state, 'Stopped');
    expect(status.loopbackUrl, isNull);
    expect(await EmbeddedTunnel.proxyUrl('http://100.64.0.1'), isNull);
    await expectLater(
      EmbeddedTunnel.start(
        stateDir: '/none',
        controlUrl: 'https://example.invalid',
        authKey: '',
        hostname: 'phone',
        upstream: 'http://100.64.0.1',
      ),
      throwsA(
        isA<EmbeddedTunnelException>().having(
          (error) => error.message,
          'message',
          contains('unavailable'),
        ),
      ),
    );
  });
}
