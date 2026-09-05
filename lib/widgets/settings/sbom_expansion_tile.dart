import 'package:flutter/material.dart';

/// One dependency row inside a [SbomExpansionTile].
class SbomEntry {
  const SbomEntry({required this.name, required this.version, this.url});
  final String name;
  final String version;
  final String? url;
}

class SbomExpansionTile extends StatelessWidget {
  const SbomExpansionTile({
    super.key,
    required this.title,
    required this.subtitle,
    required this.items,
  });

  final String title;
  final String subtitle;
  final List<SbomEntry> items;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ExpansionTile(
        title: Text(title, style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(subtitle),
        children: items
            .map(
              (item) => ListTile(
                dense: true,
                title: Text(item.name, style: const TextStyle(fontSize: 13)),
                trailing: Text(
                  item.version,
                  style: TextStyle(
                    fontSize: 12,
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                    fontFamily: 'monospace',
                  ),
                ),
              ),
            )
            .toList(),
      ),
    );
  }
}
