import 'package:autobutler/services/health_service.dart';
import 'package:autobutler/theme/autobutler_colors.dart';
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

  Color get _barColor {
    if (_diskPercent >= 0.9) return AutobutlerColors.error;
    if (_diskPercent >= 0.7) return const Color(0xFFF59E0B); // amber
    return AutobutlerColors.primary;
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: AutobutlerColors.sidebar,
        border: Border(top: BorderSide(color: AutobutlerColors.border)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          const Icon(
            Icons.storage_rounded,
            size: 14,
            color: AutobutlerColors.mutedForeground,
          ),
          const SizedBox(width: 8),
          Text(
            _loaded ? _label : 'Storage',
            style: const TextStyle(
              fontSize: 12,
              color: AutobutlerColors.mutedForeground,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Container(
              height: 8,
              constraints: const BoxConstraints(maxWidth: 320),
              decoration: BoxDecoration(
                color: AutobutlerColors.input,
                border: Border.all(color: AutobutlerColors.border),
                borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
              ),
              clipBehavior: Clip.antiAlias,
              child: AnimatedFractionallySizedBox(
                duration: const Duration(milliseconds: 400),
                curve: Curves.easeOut,
                alignment: Alignment.centerLeft,
                widthFactor: _diskPercent,
                child: Container(
                  decoration: BoxDecoration(
                    color: _barColor,
                    borderRadius: BorderRadius.circular(
                      AutobutlerColors.radiusMd,
                    ),
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(width: 8),
          if (_loaded && _diskPercent > 0)
            Text(
              '${(_diskPercent * 100).toStringAsFixed(0)}%',
              style: TextStyle(
                fontSize: 11,
                color: _barColor,
                fontWeight: FontWeight.w500,
              ),
            ),
        ],
      ),
    );
  }
}
