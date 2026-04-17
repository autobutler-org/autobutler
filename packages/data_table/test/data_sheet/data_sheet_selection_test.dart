import 'package:data_table/src/data_sheet/data_sheet_selection.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('DataSheetSelectionModel', () {
    late DataSheetSelectionModel selection;

    setUp(() {
      selection = DataSheetSelectionModel();
    });

    tearDown(() {
      selection.dispose();
    });

    test('initial state has no active cell or highlight', () {
      expect(selection.hasActiveCell, false);
      expect(selection.hasHighlight, false);
      expect(selection.activeRow, -1);
      expect(selection.activeCol, -1);
      expect(selection.highlightedRow, -1);
      expect(selection.highlightedCol, -1);
    });

    group('setActive', () {
      test('marks cell as active', () {
        selection.setActive(2, 3);
        expect(selection.activeRow, 2);
        expect(selection.activeCol, 3);
        expect(selection.hasActiveCell, true);
      });

      test('clears any existing highlight', () {
        selection.setHighlighted(1, 1);
        selection.setActive(2, 3);
        expect(selection.hasHighlight, false);
        expect(selection.highlightedRow, -1);
      });

      test('notifies listeners', () {
        var notified = false;
        selection.addListener(() => notified = true);
        selection.setActive(0, 0);
        expect(notified, true);
      });
    });

    group('setHighlighted', () {
      test('marks cell as highlighted', () {
        selection.setHighlighted(1, 2);
        expect(selection.highlightedRow, 1);
        expect(selection.highlightedCol, 2);
        expect(selection.hasHighlight, true);
      });

      test('does not clear an active cell', () {
        selection.setActive(0, 0);
        selection.setHighlighted(1, 1);
        expect(selection.hasActiveCell, true);
        expect(selection.activeRow, 0);
      });

      test('notifies listeners', () {
        var notified = false;
        selection.addListener(() => notified = true);
        selection.setHighlighted(0, 0);
        expect(notified, true);
      });
    });

    group('goTo', () {
      test('sets highlight and clears active cell', () {
        selection.setActive(1, 1);
        selection.goTo(3, 4);
        expect(selection.highlightedRow, 3);
        expect(selection.highlightedCol, 4);
        expect(selection.hasActiveCell, false);
        expect(selection.activeRow, -1);
      });

      test('notifies listeners', () {
        var notified = false;
        selection.addListener(() => notified = true);
        selection.goTo(2, 2);
        expect(notified, true);
      });
    });

    group('clear', () {
      test('clears both active cell and highlight', () {
        selection.setActive(1, 1);
        selection.setHighlighted(2, 2);
        selection.clear();
        expect(selection.hasActiveCell, false);
        expect(selection.hasHighlight, false);
      });

      test('notifies listeners', () {
        var notified = false;
        selection.addListener(() => notified = true);
        selection.clear();
        expect(notified, true);
      });
    });

    group('contextRow / contextCol', () {
      test('returns highlighted row/col when highlighted', () {
        selection.setActive(0, 0);
        selection.setHighlighted(2, 3);
        expect(selection.contextRow, 2);
        expect(selection.contextCol, 3);
      });

      test('falls back to active row/col when no highlight', () {
        selection.setActive(1, 2);
        expect(selection.contextRow, 1);
        expect(selection.contextCol, 2);
      });
    });
  });
}
