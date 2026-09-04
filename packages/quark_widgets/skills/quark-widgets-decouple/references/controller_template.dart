// A ChangeNotifier controller: state, actions, and injectable service calls.
//
// To use it:
//   1. Copy to `lib/controllers/<page>_controller.dart`.
//   2. Delete the "stand-in service" section at the bottom and import your own
//      service, so the defaults point at its real static methods.
//   3. Rename `Thing`/`ThingsController` and replace the actions with the ones
//      the page actually offers.
//
// It sits here rather than under `lib/` so `flutter analyze` and
// `dart format --set-exit-if-changed` keep it compiling, which is the only way
// a template stays honest. Nothing imports it.

import 'package:flutter/foundation.dart';

/// The value type the widgets render. Immutable, no JSON, no service imports.
@immutable
class ThingItem {
  /// Creates a thing item.
  const ThingItem({
    required this.id,
    required this.name,
    this.isFavorite = false,
  });

  /// Stable identity, and the suffix of every key a row emits for this item.
  final String id;

  /// What the row shows.
  final String name;

  /// Whether the user has starred it.
  final bool isFavorite;

  /// Returns a copy with the fields given replaced.
  ThingItem copyWith({bool? isFavorite}) =>
      ThingItem(id: id, name: name, isFavorite: isFavorite ?? this.isFavorite);

  @override
  bool operator ==(Object other) =>
      other is ThingItem &&
      other.id == id &&
      other.name == name &&
      other.isFavorite == isFavorite;

  @override
  int get hashCode => Object.hash(id, name, isFavorite);
}

/// Owns the domain state for the things page and makes every service call.
///
/// Every service call arrives as a function parameter defaulting to the real
/// implementation, so a test constructs the controller with fakes and needs no
/// mocking library and no HTTP:
///
/// ```dart
/// final controller = ThingsController(
///   fetchThings: () async => const [ThingItem(id: 'a', name: 'Alpha')],
/// );
/// await controller.load();
/// expect(controller.items, hasLength(1));
/// ```
///
/// It never touches `BuildContext`, `Navigator`, or `ScaffoldMessenger`. The
/// page owns navigation, dialogs, snack bars, and the sentence a user reads for
/// [error].
class ThingsController extends ChangeNotifier {
  /// Creates a controller. Pass fakes in a test; the page passes nothing.
  ThingsController({
    Future<List<ThingItem>> Function() fetchThings = ThingsService.list,
    Future<void> Function(String id) deleteThing = ThingsService.delete,
    Future<void> Function(String id, {required bool favorite}) setFavorite =
        ThingsService.setFavorite,
  }) : _fetchThings = fetchThings,
       _deleteThing = deleteThing,
       _setFavorite = setFavorite;

  final Future<List<ThingItem>> Function() _fetchThings;
  final Future<void> Function(String id) _deleteThing;
  final Future<void> Function(String id, {required bool favorite}) _setFavorite;

  List<ThingItem> _items = const [];
  Set<String> _selectedIds = const {};
  bool _isLoading = false;
  Object? _error;

  // Bumped on every load. A response whose token is stale is dropped, so a
  // slow first request landing after a fast second one cannot show the wrong
  // list.
  int _loadToken = 0;

  /// The things to render, in server order.
  List<ThingItem> get items => _items;

  /// The ids the user has selected.
  Set<String> get selectedIds => _selectedIds;

  /// Whether a load is in flight.
  bool get isLoading => _isLoading;

  /// The last failure, raw. The page turns it into copy.
  Object? get error => _error;

  /// Fetches the list, replacing whatever is there.
  ///
  /// Safe to call again while one is in flight: the older response is dropped.
  Future<void> load() async {
    final token = ++_loadToken;
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final fetched = await _fetchThings();
      if (token != _loadToken) return;
      _items = fetched;
      // Drop selections for things that are gone, so a delete elsewhere
      // cannot leave a phantom id selected.
      _selectedIds = _selectedIds
          .where((id) => fetched.any((thing) => thing.id == id))
          .toSet();
    } catch (e) {
      if (token != _loadToken) return;
      _error = e;
    } finally {
      // In `finally`, so a throw cannot leave the spinner up forever.
      if (token == _loadToken) {
        _isLoading = false;
        notifyListeners();
      }
    }
  }

  /// Adds or removes [id] from the selection. Pure state, no service call.
  void toggleSelected(String id) {
    _selectedIds = {..._selectedIds};
    if (!_selectedIds.remove(id)) _selectedIds.add(id);
    notifyListeners();
  }

  /// Clears the selection.
  void clearSelection() {
    if (_selectedIds.isEmpty) return;
    _selectedIds = const {};
    notifyListeners();
  }

  /// Deletes [id], then removes it locally rather than reloading the list.
  ///
  /// Returns true when the delete landed, so the page knows whether to raise
  /// a snack bar. The controller does not raise it itself.
  Future<bool> delete(String id) async {
    _error = null;
    try {
      await _deleteThing(id);
      _items = _items.where((thing) => thing.id != id).toList();
      _selectedIds = {..._selectedIds}..remove(id);
      return true;
    } catch (e) {
      _error = e;
      return false;
    } finally {
      notifyListeners();
    }
  }

  /// Flips the star on [id], showing the new state immediately and putting it
  /// back if the write fails.
  Future<void> toggleFavorite(String id) async {
    final index = _items.indexWhere((thing) => thing.id == id);
    if (index < 0) return;

    final before = _items[index];
    final after = before.copyWith(isFavorite: !before.isFavorite);
    _items = [..._items]..[index] = after;
    _error = null;
    notifyListeners();

    try {
      await _setFavorite(id, favorite: after.isFavorite);
    } catch (e) {
      // Optimistic update rolled back: the user sees the truth, not a lie
      // that survives until the next load.
      final now = _items.indexWhere((thing) => thing.id == id);
      if (now >= 0) _items = [..._items]..[now] = before;
      _error = e;
      notifyListeners();
    }
  }
}

// ---------------------------------------------------------------------------
// Stand-in service. Delete this and import your own, so the constructor
// defaults above point at the real static methods.
// ---------------------------------------------------------------------------

/// Where the real HTTP would live, app-side.
abstract final class ThingsService {
  /// Fetches every thing.
  static Future<List<ThingItem>> list() async => const [];

  /// Deletes one thing.
  static Future<void> delete(String id) async {}

  /// Sets or clears the star on one thing.
  static Future<void> setFavorite(String id, {required bool favorite}) async {}
}
