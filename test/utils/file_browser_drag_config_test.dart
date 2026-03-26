import 'package:autobutler/utils/file_browser_drag_config.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FileBrowserDragConfig', () {
    test('folderHoverExitDebounceMs is a positive integer', () {
      expect(FileBrowserDragConfig.folderHoverExitDebounceMs, isA<int>());
      expect(FileBrowserDragConfig.folderHoverExitDebounceMs, greaterThan(0));
    });

    test('autoScrollEdgeActivationPx is positive', () {
      expect(FileBrowserDragConfig.autoScrollEdgeActivationPx, isA<double>());
      expect(FileBrowserDragConfig.autoScrollEdgeActivationPx, greaterThan(0));
    });

    test('autoScrollBaseDeltaPx is positive', () {
      expect(FileBrowserDragConfig.autoScrollBaseDeltaPx, isA<double>());
      expect(FileBrowserDragConfig.autoScrollBaseDeltaPx, greaterThan(0));
    });

    test('autoScrollMaxExtraDeltaPx is positive', () {
      expect(FileBrowserDragConfig.autoScrollMaxExtraDeltaPx, isA<double>());
      expect(FileBrowserDragConfig.autoScrollMaxExtraDeltaPx, greaterThan(0));
    });

    test('max extra delta is greater than base delta', () {
      expect(
        FileBrowserDragConfig.autoScrollMaxExtraDeltaPx,
        greaterThan(FileBrowserDragConfig.autoScrollBaseDeltaPx),
      );
    });
  });
}
