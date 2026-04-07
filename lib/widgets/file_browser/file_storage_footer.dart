import 'package:autobutler/services/health_service.dart';
import 'package:autobutler/widgets/core/autobutler_storage_bar.dart';
import 'package:flutter/material.dart';

class FileStorageFooter extends StatefulWidget {
  const FileStorageFooter({super.key});

  @override
  State<FileStorageFooter> createState() => _FileStorageFooterState();
}

class _FileStorageFooterState extends State<FileStorageFooter> {
  double _diskPercent = 0;
  String _label = 'Storage';
  bool _loaded = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final status = await HealthService.getHealth();
      if (!mounted) return;
      setState(() {
        _diskPercent = (status.diskPercent / 100).clamp(0.0, 1.0);
        _label =
            '${_formatBytes(status.diskUsedBytes)}'
            ' / ${_formatBytes(status.diskTotalBytes)}';
        _loaded = true;
      });
    } catch (_) {
      // Silent — footer stays in placeholder state if health is unreachable.
      if (mounted) setState(() => _loaded = true);
    }
  }

  String _formatBytes(int bytes) {
    if (bytes <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    var value = bytes.toDouble();
    var i = 0;
    while (value >= 1024 && i < units.length - 1) {
      value /= 1024;
      i++;
    }
    return '${value.toStringAsFixed(i == 0 ? 0 : 1)} ${units[i]}';
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final barColor = AutobutlerStorageBar.colorForFraction(_diskPercent);
    return Container(
      decoration: BoxDecoration(
        color: colorScheme.secondary,
        border: Border(top: BorderSide(color: colorScheme.outline)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          Icon(
            Icons.storage_rounded,
            size: 14,
            color: colorScheme.onSurface.withValues(alpha: 0.4),
          ),
          const SizedBox(width: 8),
          Text(
            _loaded ? _label : 'Storage',
            style: TextStyle(
              fontSize: 12,
              color: colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 320),
              child: AutobutlerStorageBar(usedFraction: _diskPercent),
            ),
          ),
          const SizedBox(width: 8),
          if (_loaded && _diskPercent > 0)
            Text(
              '${(_diskPercent * 100).toStringAsFixed(0)}%',
              style: TextStyle(
                fontSize: 11,
                color: barColor,
                fontWeight: FontWeight.w500,
              ),
            ),
        ],
      ),
    );
  }
}
