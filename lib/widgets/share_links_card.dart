import 'package:autobutler/services/share_service.dart';
import 'package:autobutler/widgets/refresh_icon_button.dart';
import 'package:autobutler_icons/autobutler_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Settings-page card listing all public share links with copy and revoke
/// actions. Self-contained: loads its own data via [ShareService].
class ShareLinksCard extends StatefulWidget {
  const ShareLinksCard({super.key});

  @override
  State<ShareLinksCard> createState() => _ShareLinksCardState();
}

class _ShareLinksCardState extends State<ShareLinksCard> {
  List<ShareLink> _shares = [];
  bool _isLoading = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final shares = await ShareService.list();
      if (!mounted) return;
      setState(() {
        _shares = shares;
        _isLoading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  Future<void> _revoke(ShareLink share) async {
    try {
      await ShareService.revoke(share.id);
      await _load();
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to revoke share link')),
      );
    }
  }

  Future<void> _copy(ShareLink share) async {
    await Clipboard.setData(ClipboardData(text: share.fullUrl));
    if (!mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(const SnackBar(content: Text('Link copied')));
  }

  String _subtitle() {
    if (_isLoading) return 'Loading...';
    if (_error != null) return 'Failed to load share links';
    if (_shares.isEmpty) return 'No active share links';
    final active = _shares.where((s) => !s.expired).length;
    return '$active active link${active == 1 ? '' : 's'}';
  }

  String _shareDetails(ShareLink share) {
    final parts = <String>[];
    if (share.expired) {
      parts.add('expired');
    } else if (share.expiresAt != null) {
      parts.add('expires ${_formatRelative(share.expiresAt!)}');
    } else {
      parts.add('never expires');
    }
    if (share.passwordProtected) parts.add('password');
    parts.add(
      '${share.accessCount} download${share.accessCount == 1 ? '' : 's'}',
    );
    return parts.join(' · ');
  }

  String _formatRelative(DateTime dt) {
    final diff = dt.difference(DateTime.now());
    if (diff.isNegative) return 'now';
    if (diff.inMinutes < 60) return 'in ${diff.inMinutes}m';
    if (diff.inHours < 24) return 'in ${diff.inHours}h';
    return 'in ${diff.inDays}d';
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ExpansionTile(
        title: const Text(
          'Share links',
          style: TextStyle(fontWeight: FontWeight.w600),
        ),
        subtitle: Text(_subtitle()),
        children: [
          Align(
            alignment: Alignment.centerRight,
            child: Padding(
              padding: const EdgeInsets.only(right: 12),
              child: RefreshIconButton(
                isRefreshing: _isLoading,
                onPressed: _load,
                tooltip: 'Refresh share links',
              ),
            ),
          ),
          if (_isLoading)
            const Padding(
              padding: EdgeInsets.all(8),
              child: Center(
                child: SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            )
          else if (_error != null)
            ListTile(
              leading: Icon(
                AutobutlerIcons.error_outline,
                color: Theme.of(context).colorScheme.error,
              ),
              title: const Text('Failed to load share links'),
              subtitle: Text(_error!),
            )
          else if (_shares.isEmpty)
            const ListTile(
              title: Text('No share links yet'),
              subtitle: Text(
                'Share a file from the file browser to create one.',
              ),
            )
          else
            ..._shares.map(
              (share) => ListTile(
                leading: Icon(
                  share.expired
                      ? AutobutlerIcons.link_off
                      : AutobutlerIcons.link_outlined,
                ),
                title: Text(
                  share.filePath,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  _shareDetails(share),
                  style: const TextStyle(fontSize: 12),
                ),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      icon: const Icon(AutobutlerIcons.content_copy),
                      tooltip: 'Copy link',
                      onPressed: share.expired ? null : () => _copy(share),
                    ),
                    IconButton(
                      icon: const Icon(AutobutlerIcons.delete_outline),
                      tooltip: 'Revoke',
                      onPressed: () => _revoke(share),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
