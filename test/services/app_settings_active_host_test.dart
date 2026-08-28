import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/app_settings.dart';

/// #1623: the router gates the app behind terms acceptance, but only re-runs
/// its redirect when `refreshListenable` fires. `activeHost` had no notifier,
/// so connecting to a Quark for the first time didn't re-run the gate and the
/// terms page showed up late. Every path that changes `activeHost` must now
/// publish to [AppSettings.activeHostNotifier].
void main() {
  final settings = AppSettings.instance;

  Future<void> clearHosts() async {
    while (settings.hosts.isNotEmpty) {
      await settings.removeHost(settings.hosts.length - 1);
    }
  }

  setUp(clearHosts);
  tearDown(clearHosts);

  group('activeHostNotifier', () {
    test('fires when the first host is added', () async {
      final seen = <String?>[];
      void listener() => seen.add(settings.activeHostNotifier.value);
      settings.activeHostNotifier.addListener(listener);
      addTearDown(() => settings.activeHostNotifier.removeListener(listener));

      await settings.addHost(
        HostEntry(name: 'My Quark', hostAddress: 'http://quark.local'),
      );

      expect(seen, ['http://quark.local']);
      expect(settings.activeHostNotifier.value, settings.activeHost);
    });

    test('fires when the active host is switched', () async {
      await settings.addHost(
        HostEntry(name: 'One', hostAddress: 'http://one.local'),
      );
      await settings.addHost(
        HostEntry(name: 'Two', hostAddress: 'http://two.local'),
      );

      final seen = <String?>[];
      void listener() => seen.add(settings.activeHostNotifier.value);
      settings.activeHostNotifier.addListener(listener);
      addTearDown(() => settings.activeHostNotifier.removeListener(listener));

      await settings.setActiveIndex(0);

      expect(seen, ['http://one.local']);
    });

    test('fires when the active host is edited', () async {
      await settings.addHost(
        HostEntry(name: 'One', hostAddress: 'http://one.local'),
      );

      final seen = <String?>[];
      void listener() => seen.add(settings.activeHostNotifier.value);
      settings.activeHostNotifier.addListener(listener);
      addTearDown(() => settings.activeHostNotifier.removeListener(listener));

      await settings.updateHost(
        0,
        HostEntry(name: 'One', hostAddress: 'http://renamed.local'),
      );

      expect(seen, ['http://renamed.local']);
    });

    test('fires with null when the last host is removed', () async {
      await settings.addHost(
        HostEntry(name: 'One', hostAddress: 'http://one.local'),
      );

      final seen = <String?>[];
      void listener() => seen.add(settings.activeHostNotifier.value);
      settings.activeHostNotifier.addListener(listener);
      addTearDown(() => settings.activeHostNotifier.removeListener(listener));

      await settings.removeHost(0);

      expect(seen, [null]);
      expect(settings.activeHost, isNull);
    });

    test(
      'stays in sync with activeHost across a sequence of changes',
      () async {
        await settings.addHost(
          HostEntry(name: 'One', hostAddress: 'http://one.local'),
        );
        expect(settings.activeHostNotifier.value, settings.activeHost);

        await settings.addHost(
          HostEntry(name: 'Two', hostAddress: 'http://two.local'),
        );
        expect(settings.activeHostNotifier.value, settings.activeHost);

        await settings.setActiveIndex(0);
        expect(settings.activeHostNotifier.value, settings.activeHost);

        await settings.removeHost(0);
        expect(settings.activeHostNotifier.value, settings.activeHost);
      },
    );
  });
}
