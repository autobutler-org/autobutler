import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

import 'token_fields.dart';

/// One widget in the gallery: a live example built from fake data.
///
/// The builder is handed a `log` callback. Wire every callback the widget
/// exposes to it, so the gallery's event panel shows what the widget emits and
/// a callback that never fires is visible.
class GalleryEntry {
  /// Creates an entry for [name], filed under [group], rendering [build].
  const GalleryEntry({
    required this.name,
    required this.group,
    required this.build,
  });

  /// The class name of the widget, matching its entry in `docs.g.dart`.
  final String name;

  /// The heading this entry is listed under, usually its `lib/src/` directory.
  final String group;

  /// Builds the example. Pass `log` to every callback the widget takes.
  final Widget Function(BuildContext context, void Function(String event) log)
  build;
}

/// Every widget the gallery can show.
///
/// A package test fails when a widget exported from `quark_widgets.dart` has no
/// entry here, so this list stays complete as widgets land.
final List<GalleryEntry> registry = [
  GalleryEntry(
    name: 'Theme tokens',
    group: 'Theme',
    build: (context, log) => const _TokenSwatches(),
  ),

  // ── Core ──────────────────────────────────────────────────────────────────
  GalleryEntry(
    name: 'EmptyStateWidget',
    group: 'Core',
    build: (context, log) => EmptyStateWidget(
      icon: QuarkIcons.folder_outlined,
      headline: 'This folder is empty',
      subtext: 'Upload a file to get started.',
      action: FilledButton(
        onPressed: () => log('EmptyStateWidget action tapped'),
        child: const Text('Upload'),
      ),
    ),
  ),
  GalleryEntry(
    name: 'QuarkFileIcon',
    group: 'Core',
    build: (context, log) => Wrap(
      spacing: 24,
      runSpacing: 16,
      children: [
        for (final entry in const [
          ('Photos', true),
          ('holiday.jpg', false),
          ('clip.mp4', false),
          ('song.flac', false),
          ('report.pdf', false),
          ('notes.qdoc', false),
          ('budget.qsheet', false),
          ('backup.zip', false),
          ('unknown.xyz', false),
        ])
          Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              QuarkFileIcon(name: entry.$1, isDir: entry.$2, size: 32),
              const SizedBox(height: 4),
              Text(entry.$1, style: const TextStyle(fontSize: 11)),
            ],
          ),
      ],
    ),
  ),
  GalleryEntry(
    name: 'QuarkStorageBar',
    group: 'Core',
    build: (context, log) => Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final fraction in const [0.2, 0.8, 0.95]) ...[
          Text('${(fraction * 100).round()}% used'),
          const SizedBox(height: 4),
          QuarkStorageBar(usedFraction: fraction),
          const SizedBox(height: 16),
        ],
      ],
    ),
  ),
  GalleryEntry(
    name: 'CopyButton',
    group: 'Core',
    build: (context, log) => Wrap(
      spacing: 16,
      runSpacing: 16,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        CopyButton(
          text: 'quark-token-1234',
          onCopy: (value) async => log('CopyButton.onCopy($value)'),
        ),
        CopyButton(
          text: 'quark-token-1234',
          label: 'Copy phrase',
          variant: CopyButtonVariant.outlined,
          onCopy: (value) async => log('CopyButton.onCopy($value)'),
        ),
        CopyButton(
          text: 'quark-token-1234',
          unavailableReason: 'Clipboard unavailable — use HTTPS to enable',
          onCopy: (value) async => log('never called'),
        ),
      ],
    ),
  ),
  GalleryEntry(
    name: 'PasswordStrengthBar',
    group: 'Core',
    build: (context, log) => const _PasswordStrengthDemo(),
  ),
  GalleryEntry(
    name: 'QuarkDisconnectedView',
    group: 'Core',
    build: (context, log) => SizedBox(
      height: 520,
      child: QuarkDisconnectedView(
        hostAddress: 'https://quark.local',
        onRetry: () => log('QuarkDisconnectedView.onRetry'),
        onManageHosts: () => log('QuarkDisconnectedView.onManageHosts'),
      ),
    ),
  ),
  GalleryEntry(
    name: 'QuarkDisconnectedBanner',
    group: 'Core',
    build: (context, log) => QuarkDisconnectedBanner(
      onRetry: () => log('QuarkDisconnectedBanner.onRetry'),
    ),
  ),

  // ── Layout ────────────────────────────────────────────────────────────────
  GalleryEntry(
    name: 'QuarkAppBar',
    group: 'Layout',
    build: (context, log) => SizedBox(
      height: 360,
      child: Scaffold(
        appBar: QuarkAppBar(
          label: 'Photos',
          icon: QuarkIcons.photo_library_outlined,
          actions: [
            RefreshIconButton(
              isRefreshing: false,
              onPressed: () => log('QuarkAppBar refresh'),
            ),
            ThemeToggleButton(
              mode: ThemeMode.dark,
              onChanged: (mode) => log('ThemeToggleButton.onChanged($mode)'),
            ),
          ],
        ),
        drawer: QuarkDrawer(
          activeSection: QuarkDrawerSection.photos,
          onTapFiles: () => log('QuarkDrawer files'),
        ),
        body: const Center(child: Text('Tap the brand button')),
      ),
    ),
  ),
  GalleryEntry(
    name: 'QuarkBrandButton',
    group: 'Layout',
    build: (context, log) => Align(
      alignment: Alignment.centerLeft,
      child: QuarkBrandButton(
        label: 'Files',
        onTap: () => log('QuarkBrandButton.onTap'),
      ),
    ),
  ),
  GalleryEntry(
    name: 'QuarkDrawer',
    group: 'Layout',
    build: (context, log) => SizedBox(
      height: 520,
      width: 304,
      child: QuarkDrawer(
        activeSection: QuarkDrawerSection.photos,
        onTapFiles: () => log('QuarkDrawer.onTapFiles'),
        onTapPhotos: () => log('QuarkDrawer.onTapPhotos'),
        onTapDocs: () => log('QuarkDrawer.onTapDocs'),
        onTapSheets: () => log('QuarkDrawer.onTapSheets'),
        onTapDevices: () => log('QuarkDrawer.onTapDevices'),
        onTapHealth: () => log('QuarkDrawer.onTapHealth'),
        onTapVault: () => log('QuarkDrawer.onTapVault'),
        onTapSettings: () => log('QuarkDrawer.onTapSettings'),
      ),
    ),
  ),
  GalleryEntry(
    name: 'RefreshIconButton',
    group: 'Layout',
    build: (context, log) => Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        RefreshIconButton(
          isRefreshing: false,
          onPressed: () => log('RefreshIconButton.onPressed'),
        ),
        const RefreshIconButton(isRefreshing: true, onPressed: null),
      ],
    ),
  ),
  GalleryEntry(
    name: 'ThemeToggleButton',
    group: 'Layout',
    build: (context, log) => Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        for (final mode in ThemeMode.values)
          ThemeToggleButton(
            mode: mode,
            onChanged: (next) => log('ThemeToggleButton: $mode -> $next'),
          ),
      ],
    ),
  ),

  // ── File browser ──────────────────────────────────────────────────────────
  GalleryEntry(
    name: 'FileActionsBar',
    group: 'File browser',
    build: (context, log) => Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        FileActionsBar(
          isUploading: false,
          isCreatingFolder: false,
          isSearchMode: false,
          onUploadPressed: () => log('FileActionsBar.onUploadPressed'),
          onCreateFolderPressed: () =>
              log('FileActionsBar.onCreateFolderPressed'),
        ),
        FileActionsBar(
          isUploading: true,
          isCreatingFolder: true,
          isSearchMode: false,
          uploadTotal: 5,
          uploadCompleted: 2,
          onUploadPressed: () => log('never called'),
          onCreateFolderPressed: () => log('never called'),
        ),
      ],
    ),
  ),
  GalleryEntry(
    name: 'FileBreadcrumbBar',
    group: 'File browser',
    build: (context, log) => FileBreadcrumbBar(
      currentPath: '/photos/2024/june',
      isSearchMode: false,
      onGoHome: () => log('FileBreadcrumbBar.onGoHome'),
      onGoUp: () => log('FileBreadcrumbBar.onGoUp'),
      onPathSelected: (path) => log('FileBreadcrumbBar.onPathSelected($path)'),
    ),
  ),
  GalleryEntry(
    name: 'FileBrowserHeader',
    group: 'File browser',
    build: (context, log) => FileBrowserHeader(
      isSearchMode: true,
      searchQuery: 'invoice',
      resultCount: 4,
      onClose: () => log('FileBrowserHeader.onClose'),
    ),
  ),
  GalleryEntry(
    name: 'FileSelectionBar',
    group: 'File browser',
    build: (context, log) => FileSelectionBar(
      selectedCount: 2,
      totalCount: 7,
      onSelectAll: () => log('FileSelectionBar.onSelectAll'),
      onDeselectAll: () => log('FileSelectionBar.onDeselectAll'),
      onCancel: () => log('FileSelectionBar.onCancel'),
      onDelete: () => log('FileSelectionBar.onDelete'),
    ),
  ),
  GalleryEntry(
    name: 'NewFileDialog',
    group: 'File browser',
    build: (context, log) => NewFileDialog(
      onCreate: (name) => log('NewFileDialog.onCreate($name)'),
      onCancel: () => log('NewFileDialog.onCancel'),
    ),
  ),

  // ── Photos ────────────────────────────────────────────────────────────────
  GalleryEntry(
    name: 'PhotoSelectionBar',
    group: 'Photos',
    build: (context, log) => Column(
      children: [
        PhotoSelectionBar(
          selectedCount: 3,
          onAddToAlbum: () => log('PhotoSelectionBar.onAddToAlbum'),
          onCancel: () => log('PhotoSelectionBar.onCancel'),
        ),
        const SizedBox(height: 16),
        PhotoSelectionBar(
          selectedCount: 0,
          onAddToAlbum: () => log('never called'),
          onCancel: () => log('PhotoSelectionBar.onCancel'),
        ),
      ],
    ),
  ),
  GalleryEntry(
    name: 'LiveBadge',
    group: 'Photos',
    build: (context, log) => Container(
      width: 120,
      height: 120,
      color: const Color(0xFF7C8AA0),
      alignment: Alignment.topLeft,
      padding: const EdgeInsets.all(4),
      child: const LiveBadge(),
    ),
  ),

  // ── Albums ────────────────────────────────────────────────────────────────
  GalleryEntry(
    name: 'AlbumTreeTile',
    group: 'Albums',
    build: (context, log) => _AlbumTreeDemo(log: log),
  ),
];

/// The fake album tree the gallery shows, three levels deep so indentation and
/// expansion are both visible.
const AlbumItem _galleryAlbums = AlbumItem(
  id: 1,
  name: 'Trips',
  itemCount: 128,
  children: [
    AlbumItem(
      id: 2,
      name: 'Iceland',
      parentId: 1,
      itemCount: 40,
      children: [
        AlbumItem(id: 4, name: 'Reykjavik', parentId: 2, itemCount: 9),
      ],
    ),
    AlbumItem(id: 3, name: 'Japan', parentId: 1, itemCount: 88),
  ],
);

/// Holds the expansion and selection the tile refuses to hold, which is the
/// point of the example: the gallery is the caller.
class _AlbumTreeDemo extends StatefulWidget {
  const _AlbumTreeDemo({required this.log});

  final void Function(String event) log;

  @override
  State<_AlbumTreeDemo> createState() => _AlbumTreeDemoState();
}

class _AlbumTreeDemoState extends State<_AlbumTreeDemo> {
  final Set<int> _expanded = {1};
  int? _selected = 2;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 280,
      child: AlbumTreeTile(
        album: _galleryAlbums,
        selectedAlbumId: _selected,
        expandedIds: _expanded,
        onSelected: (album) {
          widget.log('AlbumTreeTile.onSelected(${album.name})');
          setState(() => _selected = album.id);
        },
        onToggleExpanded: (id) {
          widget.log('AlbumTreeTile.onToggleExpanded($id)');
          setState(() {
            if (!_expanded.remove(id)) _expanded.add(id);
          });
        },
        onLongPress: (album) =>
            widget.log('AlbumTreeTile.onLongPress(${album.name})'),
      ),
    );
  }
}

/// Types into a real field so the bar animates, which a static example cannot
/// show.
class _PasswordStrengthDemo extends StatefulWidget {
  const _PasswordStrengthDemo();

  @override
  State<_PasswordStrengthDemo> createState() => _PasswordStrengthDemoState();
}

class _PasswordStrengthDemoState extends State<_PasswordStrengthDemo> {
  final _controller = TextEditingController(text: 'hunter2');

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 320,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _controller,
            decoration: const InputDecoration(labelText: 'Password'),
            onChanged: (_) => setState(() {}),
          ),
          const SizedBox(height: 8),
          PasswordStrengthBar(password: _controller.text),
        ],
      ),
    );
  }
}

/// Every token in the current theme, drawn from the theme itself.
///
/// This is the gallery's own canary: edit a color in the theme panel and the
/// matching swatch has to move with it.
class _TokenSwatches extends StatelessWidget {
  const _TokenSwatches();

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Colors', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: tokens.spacingSm),
        Wrap(
          spacing: tokens.spacingSm,
          runSpacing: tokens.spacingSm,
          children: [
            for (final field in colorFields)
              _Swatch(name: field.name, color: field.read(tokens)),
          ],
        ),
        SizedBox(height: tokens.spacingLg),
        Text('Radii', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: tokens.spacingSm),
        Wrap(
          spacing: tokens.spacingSm,
          runSpacing: tokens.spacingSm,
          children: [
            _RadiusSample(name: 'radiusSm', radius: tokens.radiusSm),
            _RadiusSample(name: 'radiusMd', radius: tokens.radiusMd),
            _RadiusSample(name: 'radiusLg', radius: tokens.radiusLg),
          ],
        ),
        SizedBox(height: tokens.spacingLg),
        Text('Spacing', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: tokens.spacingSm),
        for (final step in [
          ('spacingXs', tokens.spacingXs),
          ('spacingSm', tokens.spacingSm),
          ('spacingMd', tokens.spacingMd),
          ('spacingLg', tokens.spacingLg),
          ('spacingXl', tokens.spacingXl),
        ])
          Padding(
            padding: EdgeInsets.only(bottom: tokens.spacingXs),
            child: Row(
              children: [
                SizedBox(
                  width: 96,
                  child: Text(
                    step.$1,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                    ),
                  ),
                ),
                Container(width: step.$2, height: 12, color: tokens.primary),
                SizedBox(width: tokens.spacingSm),
                Text(
                  step.$2.toStringAsFixed(0),
                  style: TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 12,
                    color: tokens.mutedForeground,
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

class _Swatch extends StatelessWidget {
  const _Swatch({required this.name, required this.color});

  final String name;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return SizedBox(
      width: 132,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            height: 44,
            decoration: BoxDecoration(
              color: color,
              border: Border.all(color: tokens.border),
              borderRadius: BorderRadius.circular(tokens.radiusMd),
            ),
          ),
          SizedBox(height: tokens.spacingXs),
          Text(
            name,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            overflow: TextOverflow.ellipsis,
          ),
          Text(
            toHex(color),
            style: TextStyle(
              fontFamily: 'monospace',
              fontSize: 11,
              color: tokens.mutedForeground,
            ),
          ),
        ],
      ),
    );
  }
}

class _RadiusSample extends StatelessWidget {
  const _RadiusSample({required this.name, required this.radius});

  final String name;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return SizedBox(
      width: 132,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            height: 44,
            decoration: BoxDecoration(
              color: tokens.card,
              border: Border.all(color: tokens.primary, width: 2),
              borderRadius: BorderRadius.circular(radius),
            ),
          ),
          SizedBox(height: tokens.spacingXs),
          Text(
            '$name ${radius.toStringAsFixed(0)}',
            style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
          ),
        ],
      ),
    );
  }
}
