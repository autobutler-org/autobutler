import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/feature_flags_controller.dart';
import 'package:quark/models/feature_flag.dart';
import 'package:quark/utils/error_text.dart';

void main() {
  const notesOn = FeatureFlag(
    key: 'notes',
    label: 'Notes',
    description: '',
    enabled: true,
  );

  test('a saved flag replaces the one it came from', () async {
    final flags = ValueNotifier<List<FeatureFlag>>(const [notesOn]);
    final controller = FeatureFlagsController(
      flags: flags,
      setFlag: (key, enabled) async => FeatureFlag(
        key: key,
        label: 'Notes',
        description: '',
        enabled: enabled,
      ),
    );
    addTearDown(controller.dispose);

    expect(await controller.setFlag('notes', false), isNull);
    expect(flags.value.single.enabled, isFalse);
    expect(controller.isSaving('notes'), isFalse);
  });

  test('a refusal keeps the flag and says what failed', () async {
    final flags = ValueNotifier<List<FeatureFlag>>(const [notesOn]);
    final controller = FeatureFlagsController(
      flags: flags,
      setFlag: (_, _) async => throw const ApiException(403, 'forbidden'),
    );
    addTearDown(controller.dispose);

    final error = await controller.setFlag('notes', false);
    expect(
      error,
      Errors.message(const ApiException(403, ''), 'change the feature'),
    );
    expect(flags.value.single.enabled, isTrue);
    expect(controller.isSaving('notes'), isFalse);
  });
}
