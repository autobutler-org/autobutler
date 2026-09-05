import 'package:flutter/material.dart';

/// One health metric: a labelled value with a progress bar that turns orange
/// then red as it approaches [criticalThreshold].
class MetricCard extends StatelessWidget {
  const MetricCard({
    required this.label,
    required this.icon,
    required this.value,
    required this.unit,
    required this.criticalThreshold,
    this.maxValue = 100,
    this.detail,
    this.corePercents,
    super.key,
  });

  final String label;
  final IconData icon;
  final double value;
  final String unit;
  final double criticalThreshold;
  final double maxValue;
  final String? detail;
  final List<double>? corePercents;

  Color _barColor(BuildContext context) {
    if (value >= criticalThreshold) return Theme.of(context).colorScheme.error;
    if (value >= criticalThreshold * 0.75) return Colors.orange;
    return Theme.of(context).colorScheme.primary;
  }

  @override
  Widget build(BuildContext context) {
    final progress = (value / maxValue).clamp(0.0, 1.0);
    final barColor = _barColor(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, size: 20),
                const SizedBox(width: 8),
                Text(
                  label,
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                const Spacer(),
                Text(
                  '${value.toStringAsFixed(1)}$unit',
                  style: TextStyle(
                    fontWeight: FontWeight.bold,
                    fontSize: 18,
                    color: barColor,
                  ),
                ),
              ],
            ),
            if (detail != null) ...[
              const SizedBox(height: 2),
              Text(
                detail!,
                style: TextStyle(
                  fontSize: 12,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
            ],
            const SizedBox(height: 10),
            ClipRRect(
              borderRadius: BorderRadius.circular(4),
              child: LinearProgressIndicator(
                value: progress,
                color: barColor,
                backgroundColor: Theme.of(
                  context,
                ).colorScheme.surfaceContainerHighest,
                minHeight: 8,
              ),
            ),
            if (corePercents != null && corePercents!.isNotEmpty) ...[
              const SizedBox(height: 10),
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: () {
                  final total = corePercents!.fold(0.0, (s, v) => s + v);
                  return corePercents!.asMap().entries.map((e) {
                    final contribution = total > 0
                        ? (e.value / total * 100)
                        : 0.0;
                    final coreColor = e.value >= 90
                        ? Theme.of(context).colorScheme.error
                        : e.value >= 67
                        ? Colors.orange
                        : Theme.of(context).colorScheme.primary;
                    return Chip(
                      label: Text(
                        'Core ${e.key + 1}: ${e.value.toStringAsFixed(0)}% (${contribution.toStringAsFixed(0)}%)',
                        style: TextStyle(fontSize: 11, color: coreColor),
                      ),
                      padding: EdgeInsets.zero,
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      visualDensity: VisualDensity.compact,
                      side: BorderSide(color: coreColor.withValues(alpha: 0.4)),
                      backgroundColor: coreColor.withValues(alpha: 0.08),
                    );
                  }).toList();
                }(),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
