import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

void main() {
  test('compares by value, children included', () {
    const a = AlbumItem(
      id: 1,
      name: 'Trips',
      children: [AlbumItem(id: 2, name: 'Iceland')],
    );
    const same = AlbumItem(
      id: 1,
      name: 'Trips',
      children: [AlbumItem(id: 2, name: 'Iceland')],
    );
    const differentChild = AlbumItem(
      id: 1,
      name: 'Trips',
      children: [AlbumItem(id: 2, name: 'Japan')],
    );

    expect(a, same);
    expect(a.hashCode, same.hashCode);
    expect(a, isNot(differentChild));
  });

  test('defaults to a childless, user-owned, empty album', () {
    const item = AlbumItem(id: 1, name: 'Trips');

    expect(item.children, isEmpty);
    expect(item.itemCount, 0);
    expect(item.isSystem, isFalse);
    expect(item.isFavorites, isFalse);
    expect(item.parentId, isNull);
  });
}
