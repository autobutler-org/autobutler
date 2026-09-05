import 'package:flutter/material.dart';

/// Marks the device that holds the vault.
class VaultBadge extends StatelessWidget {
  const VaultBadge({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(right: 4),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: Colors.purple.shade100,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        'Vault',
        style: TextStyle(
          fontSize: 12,
          color: Colors.purple.shade800,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}
