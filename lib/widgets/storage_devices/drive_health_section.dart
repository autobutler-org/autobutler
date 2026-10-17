import 'package:flutter/material.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// A drive's health badge, the Quark's sentence, and the expandable readings.
///
/// Expansion stays here. The verdict and the readings come from [health].
class DriveHealthSection extends StatefulWidget {
  /// Creates the health block for [health], keyed with [id].
  const DriveHealthSection({required this.health, required this.id, super.key});

  /// The verdict, sentence, and readings to show.
  final DriveHealth health;

  /// Distinguishes this drive's keys from the others on the page.
  final String id;

  @override
  State<DriveHealthSection> createState() => _DriveHealthSectionState();
}

class _DriveHealthSectionState extends State<DriveHealthSection> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        DriveHealthBadge(verdict: widget.health.verdict, id: widget.id),
        const SizedBox(height: 4),
        DriveHealthDetail(
          health: widget.health,
          id: widget.id,
          expanded: _expanded,
          onExpandedChanged: (open) => setState(() => _expanded = open),
        ),
      ],
    );
  }
}
