import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// Healthy/unhealthy banner at the top of the health page, listing any alerts.
class StatusBanner extends StatelessWidget {
  const StatusBanner({required this.healthy, required this.alerts, super.key});

  final bool healthy;
  final List<String> alerts;

  @override
  Widget build(BuildContext context) {
    final color = healthy
        ? Theme.of(context).colorScheme.primaryContainer
        : Theme.of(context).colorScheme.errorContainer;
    final onColor = healthy
        ? Theme.of(context).colorScheme.onPrimaryContainer
        : Theme.of(context).colorScheme.onErrorContainer;
    final icon = healthy
        ? QuarkIcons.check_circle_outline
        : QuarkIcons.warning_amber;
    final label = healthy ? 'All systems healthy' : 'Issues detected';

    return Card(
      color: color,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, color: onColor),
                const SizedBox(width: 8),
                Text(
                  label,
                  style: TextStyle(
                    color: onColor,
                    fontWeight: FontWeight.bold,
                    fontSize: 15,
                  ),
                ),
              ],
            ),
            if (alerts.isNotEmpty) ...[
              const SizedBox(height: 8),
              ...alerts.map(
                (a) => Padding(
                  padding: const EdgeInsets.only(top: 2),
                  child: Text(
                    '• $a',
                    style: TextStyle(color: onColor, fontSize: 13),
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
