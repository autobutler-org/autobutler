import 'dart:async';
import 'dart:io';

import 'package:bonsoir/bonsoir.dart';
import 'package:flutter/foundation.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/quark_discovery.dart';

/// The native browser on iOS, Android and Linux, where bonsoir browses mDNS.
/// Null elsewhere.
///
/// On Android, bonsoir holds the Wi-Fi multicast lock while it browses and
/// merges `CHANGE_WIFI_MULTICAST_STATE` into the manifest itself.
///
/// On Linux, bonsoir talks to the Avahi daemon over D-Bus. A Quark is
/// addressed by its IP first, as on Android: `.local` names resolve only
/// where nss-mdns is configured, and an address works everywhere.
QuarkBrowser? get quarkBrowser =>
    Platform.isIOS || Platform.isAndroid || Platform.isLinux
    ? () => browseQuarks()
    : null;

/// Browses for Quarks through bonsoir and emits every one found so far.
///
/// [create] builds the discovery, and a test passes one that fails.
/// [failuresAreEmpty] says a failed browse means "no Quarks found" rather than
/// an error; it defaults to true on Linux, where a machine with no Avahi
/// daemon (or no D-Bus) cannot browse and still has manual entry.
@visibleForTesting
Stream<List<HostEntry>> browseQuarks({
  Future<BonsoirDiscovery> Function()? create,
  bool? failuresAreEmpty,
}) {
  final degrade = failuresAreEmpty ?? Platform.isLinux;
  final found = <String, HostEntry>{};
  BonsoirDiscovery? discovery;
  StreamSubscription<BonsoirDiscoveryEvent>? events;
  var canceled = false;
  late final StreamController<List<HostEntry>> out;

  void fail(Object error, StackTrace st) {
    if (canceled) return;
    if (degrade) {
      out.add(const []);
    } else {
      out.addError(error, st);
    }
  }

  void onEvent(BonsoirDiscoveryEvent event) {
    switch (event) {
      case BonsoirDiscoveryServiceFoundEvent(:final service):
        service.resolve(discovery!.serviceResolver);
        return;
      case BonsoirDiscoveryServiceResolvedEvent(:final service) ||
          BonsoirDiscoveryServiceUpdatedEvent(:final service):
        // iOS keeps addressing a Quark by its SRV target; Android and Linux
        // by its IP address, since `.local` resolves on Linux only through
        // nss-mdns, which not every machine configures.
        final entry = discoveredQuark(
          name: service.name,
          hosts: Platform.isIOS
              ? [service.hostname, ...service.hostAddresses]
              : [...service.hostAddresses, service.hostname],
          port: service.port,
        );
        if (entry == null) return;
        found[service.name] = entry;
      case BonsoirDiscoveryServiceLostEvent(:final service):
        if (found.remove(service.name) == null) return;
      default:
        return;
    }
    out.add(List.unmodifiable(found.values));
  }

  Future<void> start() async {
    try {
      final d =
          await (create?.call() ??
              Future.value(BonsoirDiscovery(type: quarkServiceType)));
      discovery = d;
      await d.initialize();
      if (canceled) return;
      events = d.eventStream!.listen(onEvent, onError: fail);
      await d.start();
    } catch (e, st) {
      fail(e, st);
    }
  }

  Future<void> stop() async {
    canceled = true;
    await events?.cancel();
    try {
      await discovery?.stop();
    } catch (_) {
      // A browse canceled before it started has nothing to stop.
    }
  }

  out = StreamController<List<HostEntry>>(onListen: start, onCancel: stop);
  return out.stream;
}
