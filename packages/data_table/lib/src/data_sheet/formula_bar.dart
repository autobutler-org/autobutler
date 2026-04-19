import 'package:flutter/material.dart' hide DataCell;

import '../../data_table.dart';
import 'cell/heading/heading_cells.dart' show kResizeHandleSize;
import 'cell/heading/util.dart';
import 'data_sheet_controller.dart';

/// A formula bar companion for [DataSheetController].
///
/// Displays the current cell address (e.g. `A1`) in a fixed-width name box
/// on the left and the cell value in an editable text field on the right,
/// matching the layout used in Excel and Google Sheets.
///
/// Place it between [DataSheetControlBar] and [DataSheet]:
///
/// ```dart
/// Column(children: [
///   DataSheetControlBar(controller: myController),
///   DataSheetFormulaBar(controller: myController),
///   Expanded(child: DataSheet(controller: myController, table: myTable)),
/// ])
/// ```
///
/// Bidirectional sync:
/// - While the grid cell is being edited the bar mirrors every keystroke via
///   the shared [DataSheetController.activeCellEditingController].
/// - Typing in the bar pushes text back into that shared controller so the
///   active cell editor stays in sync.
/// - Pressing Enter (or losing focus) commits the value.
class DataSheetFormulaBar extends StatefulWidget {
  final DataSheetController controller;

  const DataSheetFormulaBar({super.key, required this.controller});

  @override
  State<DataSheetFormulaBar> createState() => _DataSheetFormulaBarState();
}

class _DataSheetFormulaBarState extends State<DataSheetFormulaBar> {
  late final TextEditingController _barCtrl;
  late final FocusNode _focusNode;

  static const double _minHeight = 32.0;
  double _height = 32.0;
  bool _handleHovered = false;

  /// Prevents the active-cell-controller listener from overwriting the bar
  /// while the bar itself is pushing a change into that controller.
  bool _suppressActiveCellSync = false;

  /// Prevents double-commit when [_commitFromBar] explicitly unfocuses.
  bool _isCommitting = false;

  DataSheetController get _controller => widget.controller;

  @override
  void initState() {
    super.initState();
    _barCtrl = TextEditingController();
    _focusNode = FocusNode();
    _controller.addListener(_onControllerChanged);
    _controller.activeCellEditingController
        .addListener(_onActiveCellTextChanged);
    _focusNode.addListener(_onFocusChanged);
  }

  @override
  void didUpdateWidget(DataSheetFormulaBar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller.removeListener(_onControllerChanged);
      oldWidget.controller.activeCellEditingController
          .removeListener(_onActiveCellTextChanged);
      _controller.addListener(_onControllerChanged);
      _controller.activeCellEditingController
          .addListener(_onActiveCellTextChanged);
      _syncBarFromController();
    }
  }

  @override
  void dispose() {
    _controller.removeListener(_onControllerChanged);
    _controller.activeCellEditingController
        .removeListener(_onActiveCellTextChanged);
    _focusNode.removeListener(_onFocusChanged);
    _barCtrl.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  // ---------------------------------------------------------------------------
  // Sync helpers
  // ---------------------------------------------------------------------------

  void _onControllerChanged() {
    // Don't overwrite the bar while the user is actively typing in it.
    if (_focusNode.hasFocus) return;
    _syncBarFromController();
  }

  void _onActiveCellTextChanged() {
    if (_suppressActiveCellSync) return;
    // Don't overwrite the bar while the user is typing in it.
    if (_focusNode.hasFocus) return;
    final newText = _controller.activeCellEditingController.text;
    if (_barCtrl.text != newText) {
      _barCtrl.text = newText;
    }
  }

  /// Reads the canonical cell value from the controller and updates the bar.
  void _syncBarFromController() {
    final sel = _controller.selection;
    final r = sel.contextRow;
    final c = sel.contextCol;
    if (r < 0 || c < 0) {
      if (_barCtrl.text.isNotEmpty) _barCtrl.clear();
      return;
    }
    // If a cell is actively being edited in the grid, prefer the live text
    // from the shared editing controller rather than the stored cell value.
    final newText = sel.hasActiveCell
        ? _controller.activeCellEditingController.text
        : _controller.cellAt(r, c).value;
    if (_barCtrl.text != newText) {
      _barCtrl.text = newText;
    }
  }

  // ---------------------------------------------------------------------------
  // Focus handling
  // ---------------------------------------------------------------------------

  void _onFocusChanged() {
    if (_focusNode.hasFocus) {
      // When the bar gains focus, put the highlighted cell into edit mode so
      // that keystrokes in the bar are reflected in the grid cell.
      final sel = _controller.selection;
      if (!sel.hasActiveCell && sel.hasHighlight) {
        final r = sel.highlightedRow;
        final c = sel.highlightedCol;
        final value = _controller.cellAt(r, c).value;
        _suppressActiveCellSync = true;
        _controller.activeCellEditingController.text = value;
        _suppressActiveCellSync = false;
        _controller.selection.setActive(r, c);
        // Move cursor to end of text in the bar.
        _barCtrl.selection =
            TextSelection.collapsed(offset: _barCtrl.text.length);
      }
    } else {
      // Lost focus — commit any pending edit unless we triggered this unfocus
      // ourselves inside [_commitFromBar].
      if (!_isCommitting) {
        _commitIfActive();
      }
    }
  }

  // ---------------------------------------------------------------------------
  // Commit logic
  // ---------------------------------------------------------------------------

  /// Commit the bar's value to the active cell and return to highlighted state.
  void _commitFromBar() {
    _isCommitting = true;
    final sel = _controller.selection;
    final r = sel.activeRow;
    final c = sel.activeCol;
    // Unfocus first so _onFocusChanged does not double-commit.
    _focusNode.unfocus();
    if (r >= 0 && c >= 0) {
      _controller.updateCell(r, c, DataCell(_barCtrl.text));
      _controller.activeCellEditingController.clear();
      _controller.selection.goTo(r, c);
    }
    _isCommitting = false;
  }

  /// Commit when focus is lost naturally (e.g. user clicks a cell in the grid).
  void _commitIfActive() {
    final sel = _controller.selection;
    final r = sel.activeRow;
    final c = sel.activeCol;
    if (r < 0 || c < 0) return;
    _controller.updateCell(r, c, DataCell(_barCtrl.text));
    _controller.activeCellEditingController.clear();
    _controller.selection.goTo(r, c);
  }

  // ---------------------------------------------------------------------------
  // Bar text change
  // ---------------------------------------------------------------------------

  void _onBarChanged(String value) {
    if (!_focusNode.hasFocus) return;
    final sel = _controller.selection;
    if (!sel.hasActiveCell) return;
    // Mirror bar text into the shared active-cell controller so the cell
    // editor (if visible) stays in sync.
    _suppressActiveCellSync = true;
    _controller.activeCellEditingController.text = value;
    _suppressActiveCellSync = false;
  }

  // ---------------------------------------------------------------------------
  // Address label
  // ---------------------------------------------------------------------------

  String _cellLabel() {
    final sel = _controller.selection;
    final r = sel.contextRow;
    final c = sel.contextCol;
    if (r < 0 || c < 0) return '';
    return '${columnLabel(c)}${r + 1}';
  }

  // ---------------------------------------------------------------------------
  // Build
  // ---------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: _controller,
      builder: (context, _) {
        final sel = _controller.selection;
        final hasSelection = sel.contextRow >= 0 && sel.contextCol >= 0;
        final label = _cellLabel();
        final cs = Theme.of(context).colorScheme;
        final dividerColor = Theme.of(context).dividerColor;

        return Stack(
          children: [
            SizedBox(
              height: _height,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  border: Border.all(color: dividerColor),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    // ── Name box ──────────────────────────────────────────
                    SizedBox(
                      width: 80,
                      child: DecoratedBox(
                        decoration: BoxDecoration(
                          border: Border(
                            right: BorderSide(color: dividerColor),
                          ),
                        ),
                        child: Center(
                          child: Text(
                            label,
                            style: Theme.of(context)
                                .textTheme
                                .bodySmall
                                ?.copyWith(
                                  fontFamily: 'monospace',
                                  fontWeight: FontWeight.w600,
                                ),
                          ),
                        ),
                      ),
                    ),
                    // ── fx icon ───────────────────────────────────────────
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                      child: Center(
                        child: Text(
                          'fx',
                          style:
                              Theme.of(context).textTheme.bodySmall?.copyWith(
                                    fontStyle: FontStyle.italic,
                                    color: cs.primary,
                                  ),
                        ),
                      ),
                    ),
                    // ── Divider ───────────────────────────────────────────
                    VerticalDivider(
                        width: 1, thickness: 1, color: dividerColor),
                    // ── Value field ───────────────────────────────────────
                    Expanded(
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 4),
                        child: TextField(
                          controller: _barCtrl,
                          focusNode: _focusNode,
                          enabled: hasSelection,
                          maxLines: null,
                          decoration: const InputDecoration(
                            border: InputBorder.none,
                            isDense: true,
                            contentPadding: EdgeInsets.symmetric(
                              horizontal: 4,
                              vertical: 6,
                            ),
                          ),
                          style: Theme.of(context).textTheme.bodyMedium,
                          onChanged: _onBarChanged,
                          onSubmitted: (_) => _commitFromBar(),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
            // ── Bottom-edge resize handle ──────────────────────────────────
            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              height: kResizeHandleSize,
              child: MouseRegion(
                cursor: SystemMouseCursors.resizeRow,
                onEnter: (_) => setState(() => _handleHovered = true),
                onExit: (_) => setState(() => _handleHovered = false),
                child: GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onVerticalDragUpdate: (d) {
                    setState(() {
                      _height =
                          (_height + d.delta.dy).clamp(_minHeight, double.infinity);
                    });
                  },
                  child: Container(
                    color: _handleHovered
                        ? cs.primary.withValues(alpha: 0.4)
                        : Colors.transparent,
                  ),
                ),
              ),
            ),
          ],
        );
      },
    );
  }
}
