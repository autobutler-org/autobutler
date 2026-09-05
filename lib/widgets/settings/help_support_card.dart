import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:url_launcher/url_launcher.dart';

class HelpSupportCard extends StatelessWidget {
  const HelpSupportCard({super.key});

  static const _supportUrl = 'https://quark.autobutler.org/support';
  static const _bugUrl =
      'https://github.com/autobutler-org/quark/issues/new?template=bug.yaml';

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Need help or found a bug?',
              style: TextStyle(fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: () => launchUrl(
                Uri.parse(_supportUrl),
                mode: LaunchMode.externalApplication,
              ),
              icon: const Icon(QuarkIcons.help_outline, size: 16),
              label: const Text('Visit support page'),
            ),
            const SizedBox(height: 8),
            OutlinedButton.icon(
              onPressed: () => launchUrl(
                Uri.parse(_bugUrl),
                mode: LaunchMode.externalApplication,
              ),
              icon: const Icon(QuarkIcons.bug_report_outlined, size: 16),
              label: const Text('Report an issue'),
            ),
          ],
        ),
      ),
    );
  }
}
