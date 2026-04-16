/// Convert a 0-based column index to a spreadsheet label (A, B, …, Z, AA, …).
///
/// Uses bijective base-26: there is no "zero" digit, so column 0 → A,
/// 25 → Z, 26 → AA, 27 → AB, … 701 → ZZ, 702 → AAA, etc.
String columnLabel(int col) {
  var label = '';
  var n = col;
  do {
    label = String.fromCharCode('A'.codeUnitAt(0) + n % 26) + label;
    n = n ~/ 26 - 1;
  } while (n >= 0);
  return label;
}
