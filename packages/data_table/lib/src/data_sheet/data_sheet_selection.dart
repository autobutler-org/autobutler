import 'package:flutter/foundation.dart';

/// Tracks the active editing cell and highlighted navigation cell for a
/// [DataSheetController].
///
/// The controller embeds one of these and propagates its change notifications
/// through its own [notifyListeners], so any widget that listens to the
/// controller automatically reacts to selection changes too.
class DataSheetSelectionModel extends ChangeNotifier {
  int _activeRow = -1;
  int _activeCol = -1;
  int _highlightedRow = -1;
  int _highlightedCol = -1;

  int get activeRow => _activeRow;
  int get activeCol => _activeCol;
  int get highlightedRow => _highlightedRow;
  int get highlightedCol => _highlightedCol;

  /// True when a cell is in edit mode.
  bool get hasActiveCell => _activeRow >= 0;

  /// True when a cell is highlighted (keyboard-navigated but not editing).
  bool get hasHighlight => _highlightedRow >= 0;

  /// The row that context-sensitive operations (delete, sort, fill, …) should
  /// act on: the highlighted row if set, otherwise the active editing row.
  int get contextRow => _highlightedRow >= 0 ? _highlightedRow : _activeRow;

  /// The column that context-sensitive operations should act on.
  int get contextCol => _highlightedCol >= 0 ? _highlightedCol : _activeCol;

  /// Mark [row]/[col] as the active editing cell and clear any highlight.
  void setActive(int row, int col) {
    _activeRow = row;
    _activeCol = col;
    _highlightedRow = -1;
    _highlightedCol = -1;
    notifyListeners();
  }

  /// Highlight [row]/[col] for keyboard navigation (no editing).
  void setHighlighted(int row, int col) {
    _highlightedRow = row;
    _highlightedCol = col;
    notifyListeners();
  }

  /// Jump to [row]/[col] as a highlighted cell, clearing the active cell.
  void goTo(int row, int col) {
    _highlightedRow = row;
    _highlightedCol = col;
    _activeRow = -1;
    _activeCol = -1;
    notifyListeners();
  }

  /// Clear both active and highlighted state.
  void clear() {
    _activeRow = -1;
    _activeCol = -1;
    _highlightedRow = -1;
    _highlightedCol = -1;
    notifyListeners();
  }
}
