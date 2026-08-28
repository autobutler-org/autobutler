import 'package:flutter/material.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark_icons/quark_icons.dart';

/// The app's "can't reach your Quark" states (#1637), in the two shapes the
/// app needs: [QuarkDisconnectedView] takes over a page whose content failed
/// to load, and [QuarkDisconnectedBanner] sits above a page that stays usable.
///
/// Both are for the case where a Quark *is* configured and simply cannot be
/// reached — off the local network, Tailscale down, device asleep. The
/// zero-config case belongs to `QuarkConnectForm`, and a vault whose USB drive
/// is unplugged has its own state on the vault page; neither is this.
///
/// The copy says only what the app actually knows. It does not guess between
/// DNS, TLS and timeout, and it never shows the underlying exception: a
/// misattributed cause reads as authoritative and sends people off to fix the
/// wrong thing (#1627).
///
/// The two shapes live in one file because they are one message in two sizes.
/// The headline, the steps and the bullet rendering are shared, and letting
/// them drift apart is the inconsistency this replaces.

/// Headline for both shapes. A plain statement of fact, no error vocabulary.
const String quarkDisconnectedHeadline = "You're not connected";

/// One-line explanation shown under the headline.
const String quarkDisconnectedBody =
    "The app can't reach your Quark right now.";

/// Headline and body on one line, for somewhere neither shape fits — a form's
/// error line, a single row in a settings list. Still says the same thing, so
/// a user who sees both does not have to reconcile two wordings.
const String quarkDisconnectedInline =
    "You're not connected — the app can't reach your Quark right now.";

/// Terse form, for a row in a list where a [QuarkDisconnectedBanner] on the
/// same page already carries the explanation and the steps.
const String quarkDisconnectedShort = 'Not connected';

/// The troubleshooting steps, in the order they are worth checking.
///
/// Deliberately static: v1 does not autodetect *why* the Quark is unreachable,
/// so every step here has to hold for every cause.
const List<String> quarkTroubleshootingSteps = [
  "Check you're on the same network as your Quark, or connected to it through "
      'Tailscale or another remote access route.',
  'Confirm the Quark address in Settings is correct.',
  'Confirm your Quark is powered on.',
];

/// [quarkTroubleshootingSteps] for pages that already show host management —
/// Settings and login — where "go to Settings" is advice to stay put.
///
/// Says nothing about *where* on the page the address is. Direction does not
/// survive contact with a real layout: login puts the host card above this
/// state, Settings puts host management below it, and either can move. Naming
/// a destination the user navigates to ("in Settings") is stable; pointing at
/// a screen position is not.
const List<String> quarkTroubleshootingStepsInPlace = [
  "Check you're on the same network as your Quark, or connected to it through "
      'Tailscale or another remote access route.',
  'Confirm the Quark address is correct.',
  'Confirm your Quark is powered on.',
];

/// Full-page disconnected state, for a page with nothing left to show.
class QuarkDisconnectedView extends StatelessWidget {
  const QuarkDisconnectedView({
    super.key,
    this.onRetry,
    this.onManageHosts,
    this.manageHostsLabel = 'Check the address',
    this.steps = quarkTroubleshootingSteps,
  });

  /// Re-runs whatever load failed. Usually the page's own refresh.
  final VoidCallback? onRetry;

  /// Takes the user to where the Quark address can be corrected. Omit on pages
  /// that already show host management, so the button never leads nowhere.
  final VoidCallback? onManageHosts;

  final String manageHostsLabel;

  final List<String> steps;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final address = AppSettings.instance.activeHost;

    return SingleChildScrollView(
      padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                QuarkIcons.cloud_off_outlined,
                size: 56,
                color: colorScheme.onSurface.withValues(alpha: 0.4),
              ),
              const SizedBox(height: 16),
              Text(
                quarkDisconnectedHeadline,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 17,
                  fontWeight: FontWeight.w600,
                  color: colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                quarkDisconnectedBody,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 14,
                  height: 1.5,
                  color: colorScheme.onSurface.withValues(alpha: 0.5),
                ),
              ),
              if (address != null && address.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(
                  address,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 13,
                    color: colorScheme.onSurface.withValues(alpha: 0.5),
                  ),
                ),
              ],
              const SizedBox(height: 24),
              _TroubleshootingList(steps: steps),
              const SizedBox(height: 24),
              Wrap(
                spacing: 12,
                runSpacing: 12,
                alignment: WrapAlignment.center,
                children: [
                  if (onRetry != null)
                    FilledButton.icon(
                      onPressed: onRetry,
                      icon: const Icon(QuarkIcons.refresh),
                      label: const Text('Try again'),
                    ),
                  if (onManageHosts != null)
                    OutlinedButton(
                      onPressed: onManageHosts,
                      child: Text(manageHostsLabel),
                    ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Compact disconnected state, for a page that stays usable while
/// disconnected — Settings and login, where the fix lives on the page itself.
class QuarkDisconnectedBanner extends StatelessWidget {
  const QuarkDisconnectedBanner({
    super.key,
    this.onRetry,
    this.steps = quarkTroubleshootingStepsInPlace,
  });

  final VoidCallback? onRetry;

  final List<String> steps;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Semantics(
      liveRegion: true,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colorScheme.outlineVariant),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  QuarkIcons.cloud_off_outlined,
                  size: 20,
                  color: colorScheme.onSurface.withValues(alpha: 0.7),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    quarkDisconnectedHeadline,
                    style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                      color: colorScheme.onSurface,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              quarkDisconnectedBody,
              style: TextStyle(
                fontSize: 14,
                height: 1.5,
                color: colorScheme.onSurface.withValues(alpha: 0.7),
              ),
            ),
            const SizedBox(height: 12),
            _TroubleshootingList(steps: steps),
            if (onRetry != null) ...[
              const SizedBox(height: 4),
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton.icon(
                  onPressed: onRetry,
                  icon: const Icon(QuarkIcons.refresh, size: 18),
                  label: const Text('Try again'),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// The bulleted checklist, left-aligned so the lines scan as a list rather
/// than as more centred prose.
class _TroubleshootingList extends StatelessWidget {
  const _TroubleshootingList({required this.steps});

  final List<String> steps;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final style = TextStyle(
      fontSize: 14,
      height: 1.5,
      color: colorScheme.onSurface.withValues(alpha: 0.7),
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final step in steps)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Excluded from semantics so a screen reader announces the
                // step, not a bullet character before every line.
                ExcludeSemantics(child: Text('•  ', style: style)),
                Expanded(child: Text(step, style: style)),
              ],
            ),
          ),
      ],
    );
  }
}
