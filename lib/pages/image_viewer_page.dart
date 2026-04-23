import 'dart:math' as math;

import 'package:autobutler/models/photo_album.dart';
import 'package:autobutler/models/photo_metadata.dart';
import 'package:autobutler/services/album_service.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/services/favorites_service.dart';
import 'package:autobutler/widgets/photos/photo_selection_bar.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:autobutler/pages/album_page.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _kSidebarOpenKey = 'photo_viewer_sidebar_open';
const _kSidebarWidth = 288.0;

/// A full-screen photo viewer with metadata sidebar (desktop) / bottom drawer
/// (mobile), action toolbar, and keyboard shortcuts.
///
/// Required: [bytes], [name].
///
/// Optional Cirrus extras (enable metadata, download, delete, album actions):
///   [relPath], [serial].
///
/// Optional context: [sourceAlbum] — when set, the more menu shows
/// "Remove from [Album]" instead of "Add to Album."
///
/// Navigation: pass [initialIndex], [imageCount], [onLoadImage].
/// Keyboard: ← → navigate, Escape closes, i toggles sidebar, f toggles
/// favorite, r rotates 90° CW.
class ImageViewerPage extends StatefulWidget {
  final Uint8List bytes;
  final String name;
  final int initialIndex;
  final int imageCount;

  /// Returns (bytes, name, relPath, serial). relPath and serial may be null
  /// for local-device assets that have no Cirrus path.
  final Future<(Uint8List?, String, String?, String?)> Function(int index)?
  onLoadImage;
  final Future<int> Function()? getImageCount;

  /// Relative path on the Cirrus server (enables metadata & server actions).
  final String? relPath;

  /// Device serial (paired with [relPath]).
  final String? serial;

  /// Album the user navigated from — makes the more menu show
  /// "Remove from [Album]" instead of "Add to Album."
  final PhotoAlbum? sourceAlbum;

  const ImageViewerPage({
    super.key,
    required this.bytes,
    required this.name,
    this.initialIndex = 0,
    this.imageCount = 1,
    this.onLoadImage,
    this.getImageCount,
    this.relPath,
    this.serial,
    this.sourceAlbum,
  });

  @override
  State<ImageViewerPage> createState() => _ImageViewerPageState();
}

class _ImageViewerPageState extends State<ImageViewerPage>
    with SingleTickerProviderStateMixin {
  // Navigation state
  late int _currentIndex;
  late Uint8List _currentBytes;
  late String _currentName;
  late String? _currentRelPath;
  late String? _currentSerial;
  bool _loading = false;
  late int _liveImageCount;

  // UI state
  bool _sidebarOpen = true;
  bool _isFavorite = false;
  int _rotationQuarters = 0; // 0/1/2/3 × 90°

  // Set to true whenever the photo list in the caller may need a refresh
  // (photo deleted, or album membership changed). Returned via pop().
  bool _listChanged = false;
  late AnimationController _rotationAnim;
  late Animation<double> _rotationValue;

  // Metadata
  PhotoMetadata? _metadata;
  bool _metadataLoading = false;

  SharedPreferences? _prefs;
  final _focusNode = FocusNode();

  // DraggableScrollableSheet controller for mobile drawer
  final _drawerController = DraggableScrollableController();

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex;
    _currentBytes = widget.bytes;
    _currentName = widget.name;
    _currentRelPath = widget.relPath;
    _currentSerial = widget.serial;
    _liveImageCount = widget.imageCount;

    _rotationAnim = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 250),
    );
    _rotationValue = Tween<double>(begin: 0, end: 0).animate(_rotationAnim);

    _initPrefs();
  }

  @override
  void dispose() {
    _rotationAnim.dispose();
    _focusNode.dispose();
    _drawerController.dispose();
    super.dispose();
  }

  Future<void> _initPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    _prefs = prefs;
    final sidebarOpen = prefs.getBool(_kSidebarOpenKey) ?? true;

    // Rotation and favorite state come from the server metadata response for
    // Cirrus photos; leave them at defaults here and let
    // _loadMetadataForCurrent apply the server values.
    setState(() => _sidebarOpen = sidebarOpen);
    _loadMetadata();
  }

  // Rotation is always server-backed. Only Cirrus photos (with a relPath)
  // support rotation — local device assets have no server path to key on.
  bool get _isCirrusPhoto =>
      _currentRelPath != null && _currentRelPath!.isNotEmpty;

  // --- Navigation ---

  bool get _hasPrev => _currentIndex > 0;
  bool get _hasNext => _currentIndex < _liveImageCount - 1;

  Future<void> _navigate(int delta) async {
    if (_loading) return;
    final newIndex = _currentIndex + delta;
    if (newIndex < 0 || newIndex >= _liveImageCount) return;
    if (widget.onLoadImage == null) return;

    setState(() => _loading = true);
    try {
      final (bytes, name, relPath, serial) = await widget.onLoadImage!(
        newIndex,
      );
      if (!mounted) return;
      if (bytes == null) {
        setState(() => _loading = false);
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Image no longer available')),
          );
        }
        return;
      }
      int updatedCount = _liveImageCount;
      if (widget.getImageCount != null) {
        updatedCount = await widget.getImageCount!();
      }
      if (!mounted) return;

      // Reset rotation and favorite to defaults; _loadMetadataForCurrent
      // will apply server-persisted values once metadata arrives.
      setState(() {
        _currentIndex = newIndex;
        _currentBytes = bytes;
        _currentName = name;
        _currentRelPath = relPath;
        _currentSerial = serial;
        _liveImageCount = updatedCount;
        _loading = false;
        _metadata = null;
        _isFavorite = false;
        _rotationQuarters = 0;
        _rotationValue = Tween<double>(begin: 0, end: 0).animate(_rotationAnim);
      });
      _loadMetadataForCurrent();
    } catch (_) {
      if (mounted) {
        setState(() => _loading = false);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Image no longer available')),
        );
      }
    }
  }

  // --- Metadata ---

  Future<void> _loadMetadata() => _loadMetadataForCurrent();

  Future<void> _loadMetadataForCurrent() async {
    final relPath = _currentRelPath;
    if (relPath == null || relPath.isEmpty) return;
    setState(() => _metadataLoading = true);
    try {
      final meta = await CirrusService.getPhotoMetadata(
        relPath,
        serial: _currentSerial,
      );
      if (!mounted) return;
      // Apply server-persisted rotation and favorite state for this photo.
      final quarters = meta.rotationQuarters.clamp(0, 3);
      setState(() {
        _metadata = meta;
        _metadataLoading = false;
        _isFavorite = meta.isFavorite;
        _rotationQuarters = quarters;
        _rotationValue = Tween<double>(
          begin: quarters * math.pi / 2,
          end: quarters * math.pi / 2,
        ).animate(_rotationAnim);
      });
    } catch (_) {
      if (mounted) setState(() => _metadataLoading = false);
    }
  }

  // --- Actions ---

  void _toggleSidebar() {
    setState(() => _sidebarOpen = !_sidebarOpen);
    _prefs?.setBool(_kSidebarOpenKey, _sidebarOpen);
  }

  Future<void> _toggleFavorite() async {
    if (!_isCirrusPhoto) return;
    final relPath = _currentRelPath!;
    final serial = _currentSerial;

    // Optimistic update.
    setState(() => _isFavorite = !_isFavorite);

    try {
      final nowFav = await FavoritesService.toggle(
        relPath: relPath,
        serial: serial?.isNotEmpty == true ? serial : null,
      );
      if (!mounted) return;
      // Reconcile with server truth in case of any race, and flag the list
      // as changed so the caller refreshes (e.g. removes from Favorites tab).
      setState(() => _isFavorite = nowFav);
      _listChanged = true;
    } catch (e) {
      if (!mounted) return;
      // Roll back and surface the error.
      setState(() => _isFavorite = !_isFavorite);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Failed to update favorite: $e')));
    }
  }

  Future<void> _rotate() async {
    if (!_isCirrusPhoto) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Rotation is only supported for Cirrus photos'),
        ),
      );
      return;
    }

    final oldQuarters = _rotationQuarters;
    final oldAngle = oldQuarters * math.pi / 2;
    final newQuarters = (oldQuarters + 1) % 4;
    final newAngle =
        newQuarters * math.pi / 2 +
        (newQuarters == 0 ? math.pi * 2 : 0); // always animate forward

    // Optimistically apply the rotation in the UI.
    setState(() => _rotationQuarters = newQuarters);
    _rotationValue = Tween<double>(
      begin: oldAngle,
      end: newAngle,
    ).animate(CurvedAnimation(parent: _rotationAnim, curve: Curves.easeOut));
    _rotationAnim.forward(from: 0);

    try {
      await CirrusService.rotatePhoto(
        _currentRelPath!,
        serial: _currentSerial,
        rotationQuarters: newQuarters,
      );
      // Evict the old decoded image from Flutter's Dart-level image cache so
      // the next Image.network load actually hits the network rather than
      // being served from memory. Combined with Cache-Control: no-cache on
      // the server, the revalidation request picks up the new ETag
      // (rotation-aware) and gets the updated thumbnail bytes.
      final thumbUrl = CirrusService.constructThumbnailUrl(
        _currentRelPath!,
        serial: _currentSerial,
      ).toString();
      await NetworkImage(thumbUrl).evict();
      _listChanged = true;
    } catch (e) {
      // Roll back the visual rotation and inform the user.
      if (!mounted) return;
      setState(() {
        _rotationQuarters = oldQuarters;
        _rotationValue = Tween<double>(begin: newAngle, end: oldAngle).animate(
          CurvedAnimation(parent: _rotationAnim, curve: Curves.easeOut),
        );
      });
      _rotationAnim.forward(from: 0);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Rotation failed: $e')));
    }
  }

  Future<void> _download() async {
    final relPath = _currentRelPath;
    if (relPath == null) return;
    try {
      await CirrusService.saveFile(
        relPath,
        serial: _currentSerial,
        fileName: _currentName,
      );
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Download failed: $e')));
      }
    }
  }

  Future<void> _addToAlbum() async {
    final relPath = _currentRelPath;
    if (relPath == null) return;
    final album = await AlbumPickerSheet.show(context, selectedCount: 1);
    if (album == null || !mounted) return;
    try {
      await AlbumService.addPhotoToAlbum(
        album.id,
        deviceSerial: _currentSerial ?? '',
        relPath: relPath,
      );
      if (!mounted) return;
      _listChanged = true;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Added to ${album.name}')));
      _loadMetadata(); // refresh album list in sidebar
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to add to album: $e')));
      }
    }
  }

  Future<void> _removeFromAlbum(PhotoAlbum album) async {
    final relPath = _currentRelPath;
    if (relPath == null) return;
    try {
      await AlbumService.removePhotoFromAlbum(
        album.id,
        deviceSerial: _currentSerial ?? '',
        relPath: relPath,
      );
      if (!mounted) return;
      _listChanged = true;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Removed from ${album.name}')));
      _loadMetadata();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to remove from album: $e')),
        );
      }
    }
  }

  Future<void> _navigateToAlbum(AlbumRef ref) async {
    // Construct a minimal PhotoAlbum from the AlbumRef already in hand —
    // no extra fetch needed since AlbumPage only needs id + name.
    final album = PhotoAlbum(
      id: ref.id,
      name: ref.name,
      parentId: null,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      itemCount: 0,
    );
    if (!mounted) return;
    await Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => AlbumPage(album: album)));
  }

  Future<void> _confirmDelete() async {
    final relPath = _currentRelPath;
    if (relPath == null) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete photo?'),
        content: Text(
          'This permanently deletes "$_currentName" from the server. '
          'This cannot be undone.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: TextButton.styleFrom(
              foregroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    try {
      final dir = relPath.contains('/')
          ? relPath.substring(0, relPath.lastIndexOf('/'))
          : '';
      await CirrusService.deleteFile(
        dir,
        _currentName,
        deviceSerial: _currentSerial,
      );
      if (!mounted) return;
      _listChanged = true;
      Navigator.of(context).pop(_listChanged);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Delete failed: $e')));
      }
    }
  }

  // --- Keyboard ---

  KeyEventResult _handleKey(FocusNode _, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    switch (event.logicalKey) {
      case LogicalKeyboardKey.arrowLeft:
        _navigate(-1);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.arrowRight:
        _navigate(1);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.escape:
        Navigator.of(context).pop(_listChanged);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.keyI:
        _toggleSidebar();
        return KeyEventResult.handled;
      case LogicalKeyboardKey.keyF:
        _toggleFavorite();
        return KeyEventResult.handled;
      case LogicalKeyboardKey.keyR:
        _rotate();
        return KeyEventResult.handled;
      case LogicalKeyboardKey.question:
        _showShortcutsDialog(context);
        return KeyEventResult.handled;
    }
    return KeyEventResult.ignored;
  }

  // --- Build ---

  @override
  Widget build(BuildContext context) {
    final isDesktop = MediaQuery.of(context).size.width >= 600;
    return KeyboardListener(
      focusNode: _focusNode,
      autofocus: true,
      onKeyEvent: (e) => _handleKey(_focusNode, e),
      child: Scaffold(
        backgroundColor: Colors.black,
        appBar: _buildAppBar(context),
        body: isDesktop ? _buildDesktopBody() : _buildMobileBody(),
      ),
    );
  }

  AppBar _buildAppBar(BuildContext context) {
    final showNav = _liveImageCount > 1;
    return AppBar(
      backgroundColor: Colors.black,
      foregroundColor: Colors.white,
      leading: IconButton(
        icon: const Icon(Icons.arrow_back),
        tooltip: 'Back (Esc)',
        onPressed: () => Navigator.of(context).pop(_listChanged),
      ),
      title: showNav
          ? Text(
              '${_currentIndex + 1} / $_liveImageCount',
              style: const TextStyle(color: Colors.white70, fontSize: 14),
            )
          : null,
      actions: [
        if (showNav) ...[
          IconButton(
            icon: const Icon(Icons.chevron_left),
            tooltip: 'Previous (←)',
            onPressed: (_hasPrev && !_loading) ? () => _navigate(-1) : null,
          ),
          IconButton(
            icon: const Icon(Icons.chevron_right),
            tooltip: 'Next (→)',
            onPressed: (_hasNext && !_loading) ? () => _navigate(1) : null,
          ),
          const SizedBox(width: 8),
        ],
        Tooltip(
          message: 'Favorite (F)',
          child: IconButton(
            icon: Icon(
              _isFavorite ? Icons.star : Icons.star_border,
              color: _isFavorite
                  ? Theme.of(context).colorScheme.primary
                  : Colors.white,
            ),
            onPressed: _toggleFavorite,
          ),
        ),
        Tooltip(
          message: 'Rotate 90° CW (R)',
          child: IconButton(
            icon: const Icon(Icons.rotate_90_degrees_cw_outlined),
            onPressed: _rotate,
          ),
        ),
        if (_currentRelPath != null)
          Tooltip(
            message: 'Download',
            child: IconButton(
              icon: const Icon(Icons.download_outlined),
              onPressed: _download,
            ),
          ),
        Tooltip(
          message: 'Info (I)',
          child: IconButton(
            icon: Icon(
              _sidebarOpen ? Icons.info : Icons.info_outline,
              color: _sidebarOpen
                  ? Theme.of(context).colorScheme.primary
                  : Colors.white,
            ),
            onPressed: _toggleSidebar,
          ),
        ),
        if (_currentRelPath != null)
          PopupMenuButton<_MoreAction>(
            icon: const Icon(Icons.more_vert),
            color: const Color(0xFF1E1E1E),
            onSelected: (action) {
              switch (action) {
                case _MoreAction.addToAlbum:
                  _addToAlbum();
                case _MoreAction.removeFromAlbum:
                  _removeFromAlbum(widget.sourceAlbum!);
                case _MoreAction.delete:
                  _confirmDelete();
              }
            },
            itemBuilder: (_) => [
              if (widget.sourceAlbum != null)
                PopupMenuItem(
                  value: _MoreAction.removeFromAlbum,
                  child: Text(
                    'Remove from ${widget.sourceAlbum!.name}',
                    style: const TextStyle(color: Colors.white),
                  ),
                )
              else
                const PopupMenuItem(
                  value: _MoreAction.addToAlbum,
                  child: Text(
                    'Add to Album',
                    style: TextStyle(color: Colors.white),
                  ),
                ),
              PopupMenuItem(
                value: _MoreAction.delete,
                child: Text(
                  'Delete photo',
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            ],
          ),
        Tooltip(
          message: 'Keyboard shortcuts (?)',
          child: IconButton(
            icon: const Icon(Icons.keyboard_outlined, size: 20),
            onPressed: () => _showShortcutsDialog(context),
          ),
        ),
        const SizedBox(width: 4),
      ],
    );
  }

  void _showShortcutsDialog(BuildContext context) {
    showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: const Color(0xFF1E1E1E),
        title: const Text(
          'Keyboard shortcuts',
          style: TextStyle(color: Colors.white),
        ),
        content: const SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              _ShortcutRow(
                shortcut: '← →',
                description: 'Previous / Next photo',
              ),
              _ShortcutRow(shortcut: 'F', description: 'Toggle favorite'),
              _ShortcutRow(shortcut: 'R', description: 'Rotate 90° clockwise'),
              _ShortcutRow(shortcut: 'I', description: 'Toggle info panel'),
              _ShortcutRow(shortcut: 'Esc', description: 'Close viewer'),
              _ShortcutRow(shortcut: '?', description: 'Show this help'),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Close', style: TextStyle(color: Colors.white70)),
          ),
        ],
      ),
    );
  }

  Widget _buildPhotoArea({bool isMobile = false}) {
    return GestureDetector(
      onTap: isMobile && _sidebarOpen
          ? () => _drawerController.animateTo(
              0.08,
              duration: const Duration(milliseconds: 200),
              curve: Curves.easeOut,
            )
          : null,
      onHorizontalDragEnd: (details) {
        if (details.primaryVelocity == null) return;
        if (details.primaryVelocity! < -200) _navigate(1);
        if (details.primaryVelocity! > 200) _navigate(-1);
      },
      child: Center(
        child: _loading
            ? const CircularProgressIndicator(color: Colors.white)
            : AnimatedBuilder(
                animation: _rotationValue,
                builder: (_, child) =>
                    Transform.rotate(angle: _rotationValue.value, child: child),
                child: InteractiveViewer(
                  child: Image.memory(
                    _currentBytes,
                    fit: BoxFit.contain,
                    errorBuilder: (context, error, stack) => const Icon(
                      Icons.broken_image,
                      size: 64,
                      color: Colors.white54,
                    ),
                  ),
                ),
              ),
      ),
    );
  }

  Widget _buildDesktopBody() {
    return Row(
      children: [
        Expanded(child: _buildPhotoArea()),
        AnimatedSize(
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeInOut,
          child: _sidebarOpen
              ? SizedBox(
                  width: _kSidebarWidth,
                  child: _MetadataSidebar(
                    name: _currentName,
                    metadata: _metadata,
                    loading: _metadataLoading,
                    onAlbumTap: _navigateToAlbum,
                  ),
                )
              : const SizedBox.shrink(),
        ),
      ],
    );
  }

  Widget _buildMobileBody() {
    return Stack(
      children: [
        _buildPhotoArea(isMobile: true),
        if (_sidebarOpen)
          DraggableScrollableSheet(
            controller: _drawerController,
            initialChildSize: 0.28,
            minChildSize: 0.08,
            maxChildSize: 0.85,
            snap: true,
            snapSizes: const [0.08, 0.28, 0.85],
            builder: (context, scrollController) => _MetadataDrawer(
              scrollController: scrollController,
              name: _currentName,
              metadata: _metadata,
              loading: _metadataLoading,
              onAlbumTap: _navigateToAlbum,
            ),
          ),
      ],
    );
  }
}

enum _MoreAction { addToAlbum, removeFromAlbum, delete }

// ---------------------------------------------------------------------------
// Metadata sidebar (desktop)
// ---------------------------------------------------------------------------

class _MetadataSidebar extends StatelessWidget {
  final String name;
  final PhotoMetadata? metadata;
  final bool loading;
  final void Function(AlbumRef album) onAlbumTap;

  const _MetadataSidebar({
    required this.name,
    required this.metadata,
    required this.loading,
    required this.onAlbumTap,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      color: const Color(0xFF111111),
      child: ListView(
        padding: const EdgeInsets.symmetric(vertical: 12),
        children: _buildSections(context),
      ),
    );
  }

  List<Widget> _buildSections(BuildContext context) =>
      _MetadataContent.sections(
        context: context,
        name: name,
        metadata: metadata,
        loading: loading,
        onAlbumTap: onAlbumTap,
      );
}

// ---------------------------------------------------------------------------
// Metadata drawer (mobile)
// ---------------------------------------------------------------------------

class _MetadataDrawer extends StatelessWidget {
  final ScrollController scrollController;
  final String name;
  final PhotoMetadata? metadata;
  final bool loading;
  final void Function(AlbumRef album) onAlbumTap;

  const _MetadataDrawer({
    required this.scrollController,
    required this.name,
    required this.metadata,
    required this.loading,
    required this.onAlbumTap,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: Color(0xFF111111),
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      child: Column(
        children: [
          const SizedBox(height: 8),
          Container(
            width: 36,
            height: 4,
            decoration: BoxDecoration(
              color: Colors.white24,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          const SizedBox(height: 8),
          Expanded(
            child: ListView(
              controller: scrollController,
              padding: const EdgeInsets.symmetric(horizontal: 0, vertical: 4),
              children: _MetadataContent.sections(
                context: context,
                name: name,
                metadata: metadata,
                loading: loading,
                onAlbumTap: onAlbumTap,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Shared metadata section builder
// ---------------------------------------------------------------------------

class _MetadataContent {
  static List<Widget> sections({
    required BuildContext context,
    required String name,
    required PhotoMetadata? metadata,
    required bool loading,
    required void Function(AlbumRef) onAlbumTap,
  }) {
    if (loading) {
      return const [
        Center(
          child: Padding(
            padding: EdgeInsets.all(32),
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
        ),
      ];
    }

    final sections = <Widget>[];

    // --- Date & time ---
    final exif = metadata?.exif;
    final dateTaken = exif?.dateTaken;
    final mtime = metadata?.mtime;
    final displayDate = dateTaken ?? mtime;
    if (displayDate != null) {
      sections.add(
        _Section(
          title: 'Date & Time',
          children: [
            _InfoRow(
              icon: Icons.calendar_today_outlined,
              value: _formatDate(displayDate.toLocal()),
            ),
            _InfoRow(
              icon: Icons.access_time_outlined,
              value: _formatTime(displayDate.toLocal()),
            ),
            if (dateTaken == null && mtime != null)
              const Padding(
                padding: EdgeInsets.only(left: 40, top: 2),
                child: Text(
                  'Estimated from file date',
                  style: TextStyle(color: Colors.white38, fontSize: 11),
                ),
              ),
          ],
        ),
      );
    }

    // --- Location ---
    if (exif?.hasLocation == true) {
      sections.add(
        _Section(
          title: 'Location',
          children: [
            _InfoRow(
              icon: Icons.location_on_outlined,
              value:
                  '${exif!.latitude!.toStringAsFixed(5)}, '
                  '${exif.longitude!.toStringAsFixed(5)}',
            ),
          ],
        ),
      );
    }

    // --- Camera & settings ---
    if (exif?.hasCameraInfo == true) {
      final rows = <Widget>[];
      final cam = exif!;
      final makeModel = [
        cam.make,
        cam.model,
      ].where((s) => s != null && s.isNotEmpty).join(' ');
      if (makeModel.isNotEmpty) {
        rows.add(_InfoRow(icon: Icons.camera_alt_outlined, value: makeModel));
      }
      if (cam.lens != null && cam.lens!.isNotEmpty) {
        rows.add(_InfoRow(icon: Icons.lens_outlined, value: cam.lens!));
      }
      final settings = <String>[];
      if (cam.aperture != null) settings.add('f/${cam.aperture}');
      if (cam.shutterSpeed != null) settings.add(cam.shutterSpeed!);
      if (cam.iso != null) settings.add('ISO ${cam.iso}');
      if (settings.isNotEmpty) {
        rows.add(
          _InfoRow(icon: Icons.tune_outlined, value: settings.join('  ·  ')),
        );
      }
      if (cam.focalLength != null) {
        rows.add(
          _InfoRow(
            icon: Icons.straighten_outlined,
            value: '${cam.focalLength} mm',
          ),
        );
      }
      if (rows.isNotEmpty) {
        sections.add(_Section(title: 'Camera', children: rows));
      }
    }

    // --- File info ---
    if (metadata != null) {
      final m = metadata;
      final ext = m.fileName.contains('.')
          ? m.fileName.split('.').last.toUpperCase()
          : 'FILE';
      sections.add(
        _Section(
          title: 'File Info',
          children: [
            _InfoRow(icon: Icons.insert_drive_file_outlined, value: m.fileName),
            _InfoRow(icon: Icons.image_outlined, value: ext),
            _InfoRow(
              icon: Icons.storage_outlined,
              value: _formatBytes(m.fileSize),
            ),
            if (m.width > 0 && m.height > 0)
              _InfoRow(
                icon: Icons.photo_size_select_large_outlined,
                value: '${m.width} × ${m.height}',
              ),
          ],
        ),
      );
    }

    // --- Albums ---
    if (metadata != null && metadata.albums.isNotEmpty) {
      sections.add(
        _Section(
          title: 'Albums',
          children: metadata.albums
              .map(
                (a) => InkWell(
                  onTap: () => onAlbumTap(a),
                  child: _InfoRow(
                    icon: Icons.photo_album_outlined,
                    value: a.name,
                    tappable: true,
                  ),
                ),
              )
              .toList(),
        ),
      );
    }

    if (sections.isEmpty && !loading) {
      sections.add(
        const Padding(
          padding: EdgeInsets.all(24),
          child: Text(
            'No metadata available',
            style: TextStyle(color: Colors.white38),
            textAlign: TextAlign.center,
          ),
        ),
      );
    }

    return sections;
  }

  static const _weekdays = [
    'Monday',
    'Tuesday',
    'Wednesday',
    'Thursday',
    'Friday',
    'Saturday',
    'Sunday',
  ];
  static const _months = [
    'January',
    'February',
    'March',
    'April',
    'May',
    'June',
    'July',
    'August',
    'September',
    'October',
    'November',
    'December',
  ];

  static String _formatDate(DateTime d) =>
      '${_weekdays[d.weekday - 1]}, ${_months[d.month - 1]} ${d.day}, ${d.year}';

  static String _formatTime(DateTime d) {
    final h = d.hour % 12 == 0 ? 12 : d.hour % 12;
    final m = d.minute.toString().padLeft(2, '0');
    final ampm = d.hour < 12 ? 'AM' : 'PM';
    return '$h:$m $ampm';
  }

  static String _formatBytes(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }
}

class _Section extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const _Section({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 6),
          child: Text(
            title.toUpperCase(),
            style: const TextStyle(
              color: Colors.white38,
              fontSize: 11,
              fontWeight: FontWeight.w600,
              letterSpacing: 0.8,
            ),
          ),
        ),
        ...children,
        const Divider(
          color: Colors.white12,
          height: 1,
          indent: 16,
          endIndent: 16,
        ),
      ],
    );
  }
}

class _InfoRow extends StatelessWidget {
  final IconData icon;
  final String value;
  final bool tappable;

  const _InfoRow({
    required this.icon,
    required this.value,
    this.tappable = false,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 5),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 16, color: Colors.white38),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              value,
              style: TextStyle(
                color: tappable
                    ? Theme.of(context).colorScheme.primary
                    : Colors.white70,
                fontSize: 13,
              ),
            ),
          ),
          if (tappable)
            const Icon(Icons.chevron_right, size: 16, color: Colors.white24),
        ],
      ),
    );
  }
}

class _ShortcutRow extends StatelessWidget {
  final String shortcut;
  final String description;

  const _ShortcutRow({required this.shortcut, required this.description});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: Colors.white12,
              borderRadius: BorderRadius.circular(4),
            ),
            child: Text(
              shortcut,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 12,
                fontFamily: 'monospace',
              ),
            ),
          ),
          const SizedBox(width: 12),
          Text(description, style: const TextStyle(color: Colors.white70)),
        ],
      ),
    );
  }
}
