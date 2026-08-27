import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/folder_picker.dart';

/// Compiling this file is half the point: it resolves the conditional import
/// to the dart:io side of the folder picker, so a break there fails the suite
/// instead of waiting for a desktop build.
void main() {
  test('desktop can pick a folder, mobile cannot', () {
    // Mobile is deliberately out of scope — Android returns protected paths
    // and iOS has no directory chooser at all, so the action is hidden there
    // rather than offered broken.
    final expected = Platform.isLinux || Platform.isMacOS || Platform.isWindows;
    expect(isFolderPickerSupported, expected);
  });
}
