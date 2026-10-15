import 'package:flutter/material.dart';
import 'package:quark/controllers/remote_access_controller.dart';
import 'package:quark/services/remote_access_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/layout/remote_access_pause_banner.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The sheet behind the app bar's connection indicator (#2857): how the app
/// is reaching its Quark, in words, and the Quark's remote access state with
/// a way to its settings.
///
/// It reads the remote access status once when it opens. While that is in
/// flight the settings row says it is checking, and if the read fails it
/// says so in [Errors]' words (#2904).
class ConnectionSheet extends StatefulWidget {
  /// Creates the sheet's content for [mode].
  const ConnectionSheet({
    required this.mode,
    required this.onOpenSettings,
    this.readRemoteAccess = RemoteAccessService.getStatus,
    this.remoteAccessOnThisPhone = true,
    this.pairingError,
    this.onRemoteAccessChanged,
    super.key,
  });

  /// How the app is reaching its Quark.
  final ConnectionMode mode;

  /// Opens the Quark's remote access settings.
  final VoidCallback onOpenSettings;

  /// Reads the Quark's remote access status. A test passes a fake.
  final Future<RemoteAccessStatus> Function() readRemoteAccess;

  /// Whether remote access is on for this phone. The switch shows it.
  final bool remoteAccessOnThisPhone;

  /// Why this phone could not set itself up, in [Errors]' words, or null.
  final String? pairingError;

  /// Turns this phone's remote access on or off. Null hides the switch,
  /// which is how the sheet stays as it was where pausing does not apply.
  final ValueChanged<bool>? onRemoteAccessChanged;

  /// [mode] in a few words, as the indicator's tooltip and this sheet's
  /// heading say it.
  static String labelFor(ConnectionMode mode) => switch (mode) {
    ConnectionMode.local => 'Connected on your home network',
    ConnectionMode.remote => 'Connected through remote access',
    ConnectionMode.offline => 'Your Quark is not reachable',
  };

  @override
  State<ConnectionSheet> createState() => _ConnectionSheetState();
}

class _ConnectionSheetState extends State<ConnectionSheet> {
  late final Future<RemoteAccessState> _remoteAccess = _read();

  Future<RemoteAccessState> _read() async {
    try {
      return RemoteAccessController.stateOf(await widget.readRemoteAccess());
    } catch (error) {
      debugPrint('[connection_sheet.dart] Remote access read failed: $error');
      rethrow;
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<RemoteAccessState>(
      future: _remoteAccess,
      builder: (context, snapshot) => ConnectionStatusView(
        mode: widget.mode,
        label: ConnectionSheet.labelFor(widget.mode),
        detail: switch (widget.mode) {
          ConnectionMode.local =>
            "You're on your home network, so the app talks to your Quark "
                'directly.',
          ConnectionMode.remote =>
            "You're away from home, so the app reaches your Quark through "
                'remote access.',
          ConnectionMode.offline => Errors.quarkOutOfReach,
        },
        remoteAccess: snapshot.data,
        isCheckingRemoteAccess:
            snapshot.connectionState != ConnectionState.done,
        remoteAccessError: snapshot.hasError
            ? Errors.message(snapshot.error, 'load remote access status')
            : null,
        onOpenSettings: widget.onOpenSettings,
        thisPhoneLabel: remoteAccessOnThisPhone,
        thisPhoneOn: widget.remoteAccessOnThisPhone,
        thisPhoneError: widget.pairingError,
        onThisPhoneChanged: widget.onRemoteAccessChanged,
      ),
    );
  }
}
