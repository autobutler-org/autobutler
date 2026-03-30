import 'dart:async';

import 'package:autobutler/services/branch_service.dart';
import 'package:flutter/material.dart';

class BranchTestingSection extends StatefulWidget {
  const BranchTestingSection({super.key, required this.currentVersion});

  final String? currentVersion;

  @override
  State<BranchTestingSection> createState() => _BranchTestingSectionState();
}

class _BranchTestingSectionState extends State<BranchTestingSection> {
  List<BranchBuild>? _branches;
  bool _isLoading = true;
  String? _error;
  bool _isRestarting = false;

  bool get _isOnBranch {
    final v = widget.currentVersion;
    return v != null && v.contains('/');
  }

  @override
  void initState() {
    super.initState();
    _loadBranches();
  }

  Future<void> _loadBranches() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final branches = await BranchService.listBranches();
      if (!mounted) return;
      setState(() {
        _branches = branches;
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

  Future<void> _confirmDeploy(BranchBuild build) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Deploy branch?'),
        content: Text(
          'This will restart AutoButler and switch to ${build.branch}. '
          "You'll use a clean database for this branch — your production data is unaffected.",
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Deploy'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await _deployAndWait(() => BranchService.deployBranch(build.branch));
  }

  Future<void> _confirmReturnToRelease() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Return to latest release?'),
        content: const Text('AutoButler will restart.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Return'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    try {
      final latestVersion = await BranchService.getLatestReleaseVersion();
      if (latestVersion == null) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Could not determine latest release version')),
        );
        return;
      }
      await _deployAndWait(() => BranchService.returnToRelease(latestVersion));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Failed: $e')),
      );
    }
  }

  Future<void> _deployAndWait(Future<void> Function() action) async {
    setState(() => _isRestarting = true);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Restarting AutoButler\u2026'),
        duration: Duration(seconds: 120),
      ),
    );

    try {
      await action();
    } catch (_) {}

    await Future<void>.delayed(const Duration(seconds: 3));

    final deadline = DateTime.now().add(const Duration(seconds: 120));
    while (DateTime.now().isBefore(deadline)) {
      if (!mounted) return;
      final ready = await BranchService.checkServerReady();
      if (ready) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).clearSnackBars();
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Ready'),
            duration: Duration(seconds: 2),
          ),
        );
        setState(() => _isRestarting = false);
        _loadBranches();
        return;
      }
      await Future<void>.delayed(const Duration(seconds: 2));
    }

    if (!mounted) return;
    ScaffoldMessenger.of(context).clearSnackBars();
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Server did not respond within 120 seconds')),
    );
    setState(() => _isRestarting = false);
  }

  String _formatBuildTime(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inSeconds < 60) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Branch Testing',
          style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
        ),
        const SizedBox(height: 8),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    const Text(
                      'Current build: ',
                      style: TextStyle(fontWeight: FontWeight.w600),
                    ),
                    Text(_isOnBranch
                        ? widget.currentVersion!
                        : 'Release build'),
                  ],
                ),
                if (_isOnBranch) ...[
                  const SizedBox(height: 8),
                  TextButton.icon(
                    onPressed: _isRestarting ? null : _confirmReturnToRelease,
                    icon: const Icon(Icons.undo, size: 18),
                    label: const Text('Return to release'),
                  ),
                ],
              ],
            ),
          ),
        ),
        const SizedBox(height: 8),
        if (_isLoading)
          const Center(
            child: Padding(
              padding: EdgeInsets.all(16),
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          )
        else if (_error != null)
          Card(
            child: ListTile(
              leading: Icon(
                Icons.error_outline,
                color: Theme.of(context).colorScheme.error,
              ),
              title: const Text('Failed to load branches'),
              subtitle: Text(_error!),
              trailing: IconButton(
                icon: const Icon(Icons.refresh),
                onPressed: _loadBranches,
              ),
            ),
          )
        else if (_branches == null || _branches!.isEmpty)
          const Card(
            child: ListTile(title: Text('No branch builds available')),
          )
        else
          ...(_branches!.map((build) => Card(
                child: ListTile(
                  title: Text(
                    build.branch,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                  subtitle: Text('PR #${build.prNumber} \u2014 ${build.prTitle}'),
                  trailing: Text(
                    _formatBuildTime(build.builtAt),
                    style: TextStyle(
                      fontSize: 12,
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
                  ),
                  onTap: _isRestarting ? null : () => _confirmDeploy(build),
                ),
              ))),
        if (_isRestarting)
          const Padding(
            padding: EdgeInsets.all(16),
            child: Center(child: CircularProgressIndicator()),
          ),
      ],
    );
  }
}
