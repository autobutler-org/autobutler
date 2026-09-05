# Worked examples

Three test files in this package, picked because between them they cover every
line of the checklist against real widgets. Read them in this order.

## `test/file_browser/file_selection_bar_test.dart`

The best example of callbacks and of a defect a test now keeps caught.

- **Callbacks, collected in order.** `pumpSelectionBar` takes an optional
  `log` and wires every handler to it, so one case taps three keyed controls
  and asserts `expect(events, ['deselectAll', 'delete', 'cancel'])`. Order and
  count both come free.
- **Every tap goes through a `ValueKey`**, never through the label:
  `find.byKey(const ValueKey('file_selection_toggle_all'))`. The label changes
  between "Select all" and "Deselect all"; the key does not.
- **A disabled action is asserted, not skipped.** With `onDelete: null` the
  case reads the `IconButton` back out and expects `onPressed` to be null.
- **Geometry as the assertion.** The bar drew under an iPhone notch (#1597),
  so the file pumps it under a `MediaQuery` carrying real device insets and
  checks `tester.getRect(...).top` against the inset for each control, plus a
  landscape case for the side insets. This is the pattern whenever a bug is
  "it was in the wrong place" rather than "it threw".
- **A helper per file.** `pumpSelectionBar` wraps `pumpAt` with this widget's
  arguments defaulted, so each case sets only what it is about.

## `test/albums/album_tree_tile_test.dart`

The best example of state as an input and of per-item keys at depth.

- **Expansion is data, not `State`.** The tile used to keep its expanded flag
  internally. Cases pump it with `expandedIds: {1}` and assert the
  children are or are not in the tree, then tap the chevron and assert the id
  came back out through `onToggleExpanded` instead of the tile changing
  itself.
- **Selection is data too**: `marks the selected album` pumps `selectedAlbumId` and
  checks the marker.
- **Keys survive nesting.** `reports the album that was tapped, at any depth`
  taps a grandchild by its own key and asserts the grandchild's id came out,
  which is what catches a parent-level `GestureDetector` swallowing the tap.
- **The absent case is a case.** `gives a childless album no chevron to tap`
  asserts the control is missing, so a chevron that always renders fails.
- **Layout, not just behavior.** `indents each level further than the last`
  compares `tester.getRect(...).left` between depths.

## `test/core/quark_disconnected_state_test.dart`

The best example of state coverage and of error copy staying in the caller.

- **Two widgets, two `group`s**, one file, because they are one concern in two
  shapes.
- **Every optional input has a present case and an absent case.**
  `shows the address it could not reach` against `shows no address when the
  app names none`; `retries through the callback the page supplied` against
  `hides the retry button when there is nothing to retry`.
- **Copy comes in, never out.** `renders the steps the caller chose` pumps a
  custom step list and asserts those exact strings, which is what stops the
  package composing a sentence of its own.
- **A behavior worth naming.** `does not send the user to Settings they are
  already on` is the kind of case that only exists because someone wrote the
  behavior down as a sentence first.

## What none of them do yet

The checklist is a higher bar than these files clear today. Nothing in the
package yet asserts `tester.takeException()` explicitly, and only
`test/photos/live_badge_test.dart` renders under both brightnesses. When you
touch one of these files, add the missing cases from the checklist rather than
matching what is already there.
