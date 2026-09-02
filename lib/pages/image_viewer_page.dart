import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:quark/controllers/photo_bytes_cache.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/models/photo_metadata.dart';
import 'package:quark/pages/album_page.dart';
import 'package:quark/services/album_service.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/services/favorites_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/image_viewer_config.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:quark/widgets/photos/photo_selection_bar.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:video_player/video_player.dart';

const _kSidebarOpenKey = 'photo_viewer_sidebar_open';
const _kSidebarWidth = 288.0;

// How long the page settles when the keyboard or a chevron turns it.
const _kPageAnimDuration = Duration(milliseconds: 250);

/// A full-screen photo viewer with metadata sidebar (desktop) / bottom drawer
/// (mobile), action toolbar, and keyboard shortcuts.
///
/// Required: [bytes], [name].
///
/// Optional Quark-device extras (enable metadata, download, delete, album actions):
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
  /// for local-device assets that have no Quark path.
  final Future<(Uint8List?, String, String?, String?)> Function(int index)?
  onLoadImage;

  /// Warms whatever cache backs [onLoadImage] for [index], in the background.
  ///
  /// Separate from [onLoadImage] because a prefetch must stay invisible: it
  /// may not show the user anything, and it may not react to a failure the way
  /// a navigation does (a 404 on a photo the user never asked for is not a
  /// reason to refetch the grid behind them).
  final Future<void> Function(int index)? onPrefetchImage;

  final Future<int> Function()? getImageCount;

  /// Relative path on the Quark device (enables metadata & server actions).
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
    this.onPrefetchImage,
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

  // The photo most recently asked for. A download whose target no longer
  // matches this has been superseded by a later swipe and drops its result.
  late int _targetIndex;

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

  // Live Photo playback
  VideoPlayerController? _liveVideoController;
  bool _liveVideoPlaying = false;
  bool _liveVideoReady = false;

  SharedPreferences? _prefs;
  final _focusNode = FocusNode();

  // Zoom/pan state of the photo. Swipe-to-navigate only applies while the
  // photo sits at 1x; once zoomed, a horizontal drag pans the photo, and the
  // downscaled decode stops being enough so the full-resolution one is worth
  // paying for.
  final _zoomController = TransformationController();
  bool _zoomedIn = false;
  late PageController _pageController;

  // DraggableScrollableSheet controller for mobile drawer
  final _drawerController = DraggableScrollableController();

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex;
    _targetIndex = widget.initialIndex;
    _currentBytes = widget.bytes;
    _currentName = widget.name;
    _currentRelPath = widget.relPath;
    _currentSerial = widget.serial;
    _liveImageCount = widget.imageCount;
    _pageController = PageController(initialPage: widget.initialIndex);
    _zoomController.addListener(_onZoomChanged);

    _rotationAnim = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 250),
    );
    _rotationValue = Tween<double>(begin: 0, end: 0).animate(_rotationAnim);

    _initPrefs();
    _prefetchNeighbors();
  }

  @override
  void dispose() {
    _liveVideoController?.dispose();
    _rotationAnim.dispose();
    _focusNode.dispose();
    _drawerController.dispose();
    _zoomController.removeListener(_onZoomChanged);
    _zoomController.dispose();
    _pageController.dispose();
    super.dispose();
  }

  Future<void> _initPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    _prefs = prefs;
    final sidebarOpen = prefs.getBool(_kSidebarOpenKey) ?? true;

    // Rotation and favorite state come from the server metadata response for
    // Quark photos; leave them at defaults here and let
    // _loadMetadataForCurrent apply the server values.
    setState(() => _sidebarOpen = sidebarOpen);
    _loadMetadata();
  }

  // Rotation is always server-backed. Only Quark photos (with a relPath)
  // support rotation — local device assets have no server path to key on.
  bool get _isQuarkPhoto =>
      _currentRelPath != null && _currentRelPath!.isNotEmpty;

  // --- Navigation ---

  // Measured from the newest requested photo so the chevrons stay live, and
  // keep stepping, while a download is still in flight.
  bool get _hasPrev => _targetIndex > 0;
  bool get _hasNext => _targetIndex < _liveImageCount - 1;

  // A PageView carries the swipe so the photo tracks the finger and settles
  // with an animation (#1707). The keyboard and the chevrons drive the same
  // controller so every route to the next photo animates the same way.
  //
  // The step counts from the newest requested photo rather than the one on
  // screen, so a second press while the first is still downloading moves on
  // instead of asking for the same photo twice (#1710).
  void _goToPage(int delta) {
    final target = _targetIndex + delta;
    if (target < 0 || target >= _liveImageCount) return;
    if (!_pageController.hasClients) {
      _navigate(target);
      return;
    }
    _pageController.animateToPage(
      target,
      duration: _kPageAnimDuration,
      curve: Curves.easeOut,
    );
  }

  Future<void> _onPageChanged(int index) async {
    if (index == _currentIndex) return;
    await _navigate(index);
    // The load failed, so there are no bytes for the page the finger settled
    // on. Put the view back on the photo we still have — but only once nothing
    // is in flight, because a target that no longer matches what is on screen
    // means the user has swiped on again and this recovery would drag them
    // back off the photo they asked for (#1710).
    if (!mounted || !_pageController.hasClients) return;
    if (_targetIndex != _currentIndex || index == _currentIndex) return;
    _pageController.jumpToPage(_currentIndex);
  }

  // A zoomed photo pans inside the InteractiveViewer, so the PageView has to
  // stop scrolling until it is back at 1x, and the photo has to be decoded
  // again at full resolution. Both start the moment the user pinches, so one flag
  // and one threshold cover them (#1707, #1710).
  void _onZoomChanged() {
    final zoomed =
        _zoomController.value.getMaxScaleOnAxis() >
        ImageViewerConfig.zoomedInScale;
    if (zoomed != _zoomedIn) setState(() => _zoomedIn = zoomed);
  }

  /// Downloads the photos either side of this one into the loader's cache.
  ///
  /// The next thing the user does is nearly always "next photo" or "previous
  /// photo", so paying for those while they look at this one is what makes the
  /// step feel instant. Nothing here is awaited, nothing here touches state,
  /// and a failure is swallowed: a photo the user never asked for must not
  /// produce a snackbar, and the widget may well be gone before the download
  /// lands (#1710).
  void _prefetchNeighbors() {
    final prefetch = widget.onPrefetchImage;
    if (prefetch == null) return;
    for (var delta = 1; delta <= ImageViewerConfig.prefetchWindow; delta++) {
      for (final index in [_currentIndex - delta, _currentIndex + delta]) {
        if (index < 0 || index >= _liveImageCount) continue;
        unawaited(prefetch(index).catchError((Object _) {}));
      }
    }
  }

  /// Shows the photo at [newIndex], downloading it first.
  ///
  /// Navigations coalesce: the newest one wins. A request that arrives while
  /// an earlier download is still in flight used to be dropped on the floor,
  /// which meant swiping faster than the network could answer threw away every
  /// swipe but the first — and left [_onPageChanged] snapping the view back to
  /// the photo it already had. Now every request runs, and each one checks
  /// after every await whether the user has since asked for somewhere else; if
  /// they have, it discards its result rather than yanking them back to a
  /// photo they have already passed. The bytes it downloaded still land in
  /// [PhotoBytesCache], so a superseded load is not wasted (#1710).
  Future<void> _navigate(int newIndex) async {
    if (!mounted) return;
    if (newIndex < 0 || newIndex >= _liveImageCount) return;
    if (widget.onLoadImage == null) return;

    _targetIndex = newIndex;
    setState(() => _loading = true);
    try {
      final (bytes, name, relPath, serial) = await widget.onLoadImage!(
        newIndex,
      );
      if (!mounted || _targetIndex != newIndex) return;
      if (bytes == null) {
        _targetIndex = _currentIndex;
        setState(() => _loading = false);
        _showLoadFailure(null, newIndex);
        return;
      }
      int updatedCount = _liveImageCount;
      if (widget.getImageCount != null) {
        updatedCount = await widget.getImageCount!();
      }
      if (!mounted || _targetIndex != newIndex) return;

      _disposeLiveVideo();
      _zoomController.value = Matrix4.identity();
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
      _prefetchNeighbors();
    } catch (e) {
      debugPrint('ImageViewerPage: failed to load image $newIndex: $e');
      if (!mounted || _targetIndex != newIndex) return;
      _targetIndex = _currentIndex;
      setState(() => _loading = false);
      _showLoadFailure(e, newIndex);
    }
  }

  /// Tells the user a photo wouldn't load, and offers another go at it.
  ///
  /// A 404 is the one answer that means the photo really is gone, so it gets
  /// the "no longer there" copy and no retry. Everything else — a dropped
  /// request, a busy Quark, a list that moved underneath us — is worth trying
  /// again, so the snackbar carries a Retry rather than stranding the viewer
  /// on the photo it was already showing (#1708).
  void _showLoadFailure(Object? error, int newIndex) {
    final gone = error is ApiException && error.statusCode == 404;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(Errors.message(error, 'load the photo')),
        action: gone
            ? null
            : SnackBarAction(
                label: 'Retry',
                onPressed: () => _navigate(newIndex),
              ),
      ),
    );
  }

  // --- Metadata ---

  Future<void> _loadMetadata() => _loadMetadataForCurrent();

  Future<void> _loadMetadataForCurrent() async {
    final relPath = _currentRelPath;
    if (relPath == null || relPath.isEmpty) return;
    setState(() => _metadataLoading = true);
    try {
      final meta = await FilesService.getPhotoMetadata(
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
      if (meta.isLivePhoto) _prepareLiveVideo();
    } catch (_) {
      if (mounted) setState(() => _metadataLoading = false);
    }
  }

  // --- Live Photo ---

  void _prepareLiveVideo() {
    _disposeLiveVideo();
    final videoPath = _metadata?.livePhotoVideoPath;
    if (videoPath == null) return;

    final url = FilesService.constructMediaUrl(
      videoPath,
      serial: _currentSerial,
    );
    final controller = VideoPlayerController.networkUrl(url);
    _liveVideoController = controller;
    controller.setLooping(true);
    controller
        .initialize()
        .then((_) {
          if (!mounted || _liveVideoController != controller) return;
          setState(() => _liveVideoReady = true);
        })
        .catchError((_) {});
  }

  void _disposeLiveVideo() {
    _liveVideoController?.dispose();
    _liveVideoController = null;
    _liveVideoReady = false;
    _liveVideoPlaying = false;
  }

  void _startLivePlayback() {
    if (!_liveVideoReady || _liveVideoController == null) return;
    _liveVideoController!.seekTo(Duration.zero);
    _liveVideoController!.play();
    setState(() => _liveVideoPlaying = true);
  }

  void _stopLivePlayback() {
    _liveVideoController?.pause();
    if (mounted) setState(() => _liveVideoPlaying = false);
  }

  // --- Actions ---

  void _toggleSidebar() {
    setState(() => _sidebarOpen = !_sidebarOpen);
    _prefs?.setBool(_kSidebarOpenKey, _sidebarOpen);
  }

  Future<void> _toggleFavorite() async {
    if (!_isQuarkPhoto) return;
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
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'update the favorite'))),
      );
    }
  }

  Future<void> _rotate() async {
    if (!_isQuarkPhoto) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Rotation is only supported for Quark photos'),
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
      await FilesService.rotatePhoto(
        _currentRelPath!,
        serial: _currentSerial,
        rotationQuarters: newQuarters,
      );
      // Evict the old decoded image from Flutter's Dart-level image cache so
      // the next Image.network load actually hits the network rather than
      // being served from memory. Combined with Cache-Control: no-cache on
      // the server, the revalidation request picks up the new ETag
      // (rotation-aware) and gets the updated thumbnail bytes.
      final thumbUrl = FilesService.constructThumbnailUrl(
        _currentRelPath!,
        serial: _currentSerial,
      ).toString();
      await NetworkImage(thumbUrl).evict();
      // Same reason, for the full-resolution bytes: the cache is keyed by path
      // and serial, neither of which a rotation changes (#1710).
      PhotoBytesCache.instance.evict(
        PhotoBytesCache.key(_currentRelPath!, _currentSerial),
      );
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
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'rotate the photo'))),
      );
    }
  }

  Future<void> _download() async {
    final relPath = _currentRelPath;
    if (relPath == null) return;
    try {
      await FilesService.saveFile(
        relPath,
        serial: _currentSerial,
        fileName: _currentName,
      );
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(Errors.message(e, 'download the photo'))),
        );
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
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(Errors.message(e, 'add it to the album'))),
        );
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
          SnackBar(
            content: Text(Errors.message(e, 'remove it from the album')),
          ),
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

  Future<void> _makeACopy() async {
    final relPath = _currentRelPath;
    if (relPath == null) return;
    try {
      final newRelPath = await FilesService.copyPhoto(
        relPath,
        serial: _currentSerial,
      );
      if (!mounted) return;
      final newName = newRelPath.contains('/')
          ? newRelPath.substring(newRelPath.lastIndexOf('/') + 1)
          : newRelPath;
      _listChanged = true;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Copy saved as $newName')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'copy the photo'))),
      );
    }
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
      await FilesService.deleteFile(
        dir,
        _currentName,
        deviceSerial: _currentSerial,
      );
      if (!mounted) return;
      _listChanged = true;
      Navigator.of(context).pop(_listChanged);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(Errors.message(e, 'delete the photo'))),
        );
      }
    }
  }

  // --- Keyboard ---

  KeyEventResult _handleKey(FocusNode _, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    switch (event.logicalKey) {
      case LogicalKeyboardKey.arrowLeft:
        _goToPage(-1);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.arrowRight:
        _goToPage(1);
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
    // `canPop: false` reports `RoutePopDisposition.doNotPop`, which is what
    // turns off the iOS left-edge back-swipe on this route. Without it that
    // edge gesture beats the photo page view near the bezel and drops the
    // user out of the viewer mid-swipe (#1707). Every other exit still calls
    // `Navigator.pop` directly, which never consults this scope, so only a
    // system back arrives here and it is popped by hand.
    return PopScope<bool>(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) return;
        Navigator.of(context).pop(_listChanged);
      },
      child: KeyboardListener(
        focusNode: _focusNode,
        autofocus: true,
        onKeyEvent: (e) => _handleKey(_focusNode, e),
        child: Scaffold(
          backgroundColor: Colors.black,
          appBar: _buildAppBar(context, isDesktop),
          body: isDesktop ? _buildDesktopBody() : _buildMobileBody(),
        ),
      ),
    );
  }

  /// The bar carries the close button, the counter and the prev/next
  /// chevrons at every width. Narrow screens have no room for the rest, so
  /// favorite, rotate, download and info fold into the more menu instead of
  /// crowding the close button (#1709).
  AppBar _buildAppBar(BuildContext context, bool isDesktop) {
    final showNav = _liveImageCount > 1;
    return AppBar(
      backgroundColor: Colors.black,
      foregroundColor: Colors.white,
      leading: IconButton(
        icon: const Icon(QuarkIcons.close),
        tooltip: 'Close (Esc)',
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
            icon: const Icon(QuarkIcons.chevron_left),
            tooltip: 'Previous (←)',
            onPressed: _hasPrev ? () => _goToPage(-1) : null,
          ),
          IconButton(
            icon: const Icon(QuarkIcons.chevron_right),
            tooltip: 'Next (→)',
            onPressed: _hasNext ? () => _goToPage(1) : null,
          ),
          const SizedBox(width: 8),
        ],
        if (isDesktop) ...[
          Tooltip(
            message: 'Favorite (F)',
            child: IconButton(
              icon: Icon(
                _isFavorite ? QuarkIcons.star : QuarkIcons.star_border,
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
              icon: const Icon(QuarkIcons.rotate_90_degrees_cw_outlined),
              onPressed: _rotate,
            ),
          ),
          if (_currentRelPath != null)
            Tooltip(
              message: 'Download',
              child: IconButton(
                icon: const Icon(QuarkIcons.download_outlined),
                onPressed: _download,
              ),
            ),
          Tooltip(
            message: 'Info (I)',
            child: IconButton(
              icon: Icon(
                _sidebarOpen ? QuarkIcons.info : QuarkIcons.info_outline,
                color: _sidebarOpen
                    ? Theme.of(context).colorScheme.primary
                    : Colors.white,
              ),
              onPressed: _toggleSidebar,
            ),
          ),
        ],
        if (!isDesktop || _currentRelPath != null)
          PopupMenuButton<_MoreAction>(
            icon: const Icon(QuarkIcons.more_vert),
            color: const Color(0xFF1E1E1E),
            onSelected: (action) {
              switch (action) {
                case _MoreAction.favorite:
                  _toggleFavorite();
                case _MoreAction.rotate:
                  _rotate();
                case _MoreAction.download:
                  _download();
                case _MoreAction.info:
                  _toggleSidebar();
                case _MoreAction.addToAlbum:
                  _addToAlbum();
                case _MoreAction.removeFromAlbum:
                  _removeFromAlbum(widget.sourceAlbum!);
                case _MoreAction.makeACopy:
                  _makeACopy();
                case _MoreAction.delete:
                  _confirmDelete();
              }
            },
            itemBuilder: (_) => [
              if (!isDesktop) ...[
                PopupMenuItem(
                  value: _MoreAction.favorite,
                  child: Text(
                    _isFavorite ? 'Unfavorite' : 'Favorite',
                    style: const TextStyle(color: Colors.white),
                  ),
                ),
                const PopupMenuItem(
                  value: _MoreAction.rotate,
                  child: Text(
                    'Rotate 90° CW',
                    style: TextStyle(color: Colors.white),
                  ),
                ),
                if (_currentRelPath != null)
                  const PopupMenuItem(
                    value: _MoreAction.download,
                    child: Text(
                      'Download',
                      style: TextStyle(color: Colors.white),
                    ),
                  ),
                PopupMenuItem(
                  value: _MoreAction.info,
                  child: Text(
                    _sidebarOpen ? 'Hide info' : 'Show info',
                    style: const TextStyle(color: Colors.white),
                  ),
                ),
              ],
              if (_currentRelPath != null) ...[
                if (!isDesktop) const PopupMenuDivider(),
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
                const PopupMenuItem(
                  value: _MoreAction.makeACopy,
                  child: Text(
                    'Make a Copy',
                    style: TextStyle(color: Colors.white),
                  ),
                ),
                PopupMenuItem(
                  value: _MoreAction.delete,
                  child: Text(
                    'Delete photo',
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                ),
              ],
            ],
          ),
        // Keyboard shortcuts and the theme toggle need a keyboard and a wider
        // bar; on a phone they stay on the pages that have room for them.
        if (isDesktop) ...[
          Tooltip(
            message: 'Keyboard shortcuts (?)',
            child: IconButton(
              icon: const Icon(QuarkIcons.keyboard_outlined, size: 20),
              onPressed: () => _showShortcutsDialog(context),
            ),
          ),
          const SizedBox(width: 4),
          const ThemeToggleButton(),
        ],
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

  /// Pixel width to decode the photo at, or null for its full resolution.
  ///
  /// A phone photo is several times wider than the screen showing it, and
  /// decoding one at full sensor resolution costs both the decode and ~48MB of
  /// RGBA held for a frame nobody can see the detail in. Same bytes off the
  /// network either way — this is free speed (#1710).
  ///
  /// The *larger* viewport side is used because `BoxFit.contain` may add a
  /// letterbox on either axis and `Transform.rotate` may have turned the
  /// photo a quarter turn, so the smaller side is not a safe bound. Zoomed in,
  /// the downscaled decode is no longer enough and the full-resolution one is
  /// what the user pinched for.
  int? _decodeWidth(BuildContext context, BoxConstraints constraints) {
    if (_zoomedIn) return null;
    final side = math.max(constraints.maxWidth, constraints.maxHeight);
    if (!side.isFinite || side <= 0) return null;
    return (side * MediaQuery.devicePixelRatioOf(context)).round();
  }

  Widget _buildPhotoArea({bool isMobile = false}) {
    final isLive = _metadata?.isLivePhoto ?? false;

    return GestureDetector(
      onTap: isMobile && _sidebarOpen
          ? () => setState(() => _sidebarOpen = false)
          : null,
      onLongPressStart: isLive && _liveVideoReady
          ? (_) => _startLivePlayback()
          : null,
      onLongPressEnd: isLive && _liveVideoPlaying
          ? (_) => _stopLivePlayback()
          : null,
      child: Stack(
        children: [
          // The viewer is index-based and only ever holds the bytes for the
          // photo on screen, so the pages the finger drags in from show a
          // spinner until _navigate has loaded them.
          PageView.builder(
            controller: _pageController,
            physics: _zoomedIn
                ? const NeverScrollableScrollPhysics()
                : const PageScrollPhysics(),
            itemCount: math.max(_liveImageCount, 1),
            onPageChanged: _onPageChanged,
            itemBuilder: (_, index) => Center(
              child: index == _currentIndex
                  ? _buildCurrentPhoto()
                  : const CircularProgressIndicator(color: Colors.white),
            ),
          ),
          if (isLive && !_loading)
            Positioned(
              top: 12,
              left: 12,
              child: _LiveBadge(ready: _liveVideoReady),
            ),
        ],
      ),
    );
  }

  Widget _buildCurrentPhoto() {
    if (_liveVideoPlaying && _liveVideoController != null) {
      return AspectRatio(
        aspectRatio: _liveVideoController!.value.aspectRatio.clamp(0.1, 10.0),
        child: VideoPlayer(_liveVideoController!),
      );
    }
    return AnimatedBuilder(
      animation: _rotationValue,
      builder: (_, child) =>
          Transform.rotate(angle: _rotationValue.value, child: child),
      child: InteractiveViewer(
        transformationController: _zoomController,
        child: LayoutBuilder(
          builder: (context, constraints) => Image.memory(
            _currentBytes,
            fit: BoxFit.contain,
            cacheWidth: _decodeWidth(context, constraints),
            // Keep the downscaled frame on screen while the full-resolution
            // one decodes, so starting a pinch doesn't blank the photo.
            gaplessPlayback: true,
            errorBuilder: (context, error, stack) => const Icon(
              QuarkIcons.broken_image,
              size: 64,
              color: Colors.white54,
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

enum _MoreAction {
  favorite,
  rotate,
  download,
  info,
  addToAlbum,
  removeFromAlbum,
  makeACopy,
  delete,
}

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
              icon: QuarkIcons.calendar_today_outlined,
              value: _formatDate(displayDate.toLocal()),
            ),
            _InfoRow(
              icon: QuarkIcons.access_time_outlined,
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
              icon: QuarkIcons.location_on_outlined,
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
        rows.add(
          _InfoRow(icon: QuarkIcons.camera_alt_outlined, value: makeModel),
        );
      }
      if (cam.lens != null && cam.lens!.isNotEmpty) {
        rows.add(_InfoRow(icon: QuarkIcons.lens_outlined, value: cam.lens!));
      }
      final settings = <String>[];
      if (cam.aperture != null) settings.add('f/${cam.aperture}');
      if (cam.shutterSpeed != null) settings.add(cam.shutterSpeed!);
      if (cam.iso != null) settings.add('ISO ${cam.iso}');
      if (settings.isNotEmpty) {
        rows.add(
          _InfoRow(
            icon: QuarkIcons.tune_outlined,
            value: settings.join('  ·  '),
          ),
        );
      }
      if (cam.focalLength != null) {
        rows.add(
          _InfoRow(
            icon: QuarkIcons.straighten_outlined,
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
            _InfoRow(
              icon: QuarkIcons.insert_drive_file_outlined,
              value: m.fileName,
            ),
            _InfoRow(icon: QuarkIcons.image_outlined, value: ext),
            _InfoRow(
              icon: QuarkIcons.storage_outlined,
              value: _formatBytes(m.fileSize),
            ),
            if (m.width > 0 && m.height > 0)
              _InfoRow(
                icon: QuarkIcons.photo_size_select_large_outlined,
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
                    icon: QuarkIcons.photo_album_outlined,
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
            const Icon(
              QuarkIcons.chevron_right,
              size: 16,
              color: Colors.white24,
            ),
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

class _LiveBadge extends StatelessWidget {
  final bool ready;

  const _LiveBadge({required this.ready});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: Colors.black54,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: Colors.white24),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.circle,
            size: 8,
            color: ready ? Colors.yellowAccent : Colors.white38,
          ),
          const SizedBox(width: 5),
          Text(
            'LIVE',
            style: TextStyle(
              color: ready ? Colors.white : Colors.white54,
              fontSize: 11,
              fontWeight: FontWeight.w600,
              letterSpacing: 0.5,
            ),
          ),
        ],
      ),
    );
  }
}
