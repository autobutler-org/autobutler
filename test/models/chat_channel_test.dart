import 'package:flutter_test/flutter_test.dart';
import 'package:quark/models/chat_channel.dart';

/// The channel list's wire shape, with the unread count #2424 added.
void main() {
  test('reads lastReadMessageId, and zero from a Quark that sends none', () {
    final json = {'id': 4, 'name': 'family', 'lastReadMessageId': 5};

    expect(ChatChannel.fromJson(json).lastReadMessageId, 5);
    expect(
      ChatChannel.fromJson({'id': 4, 'name': 'family'}).lastReadMessageId,
      0,
    );
  });

  test('reads unreadCount, and zero from a Quark that sends none', () {
    final json = {'id': 4, 'name': 'family', 'unreadCount': 3};

    expect(ChatChannel.fromJson(json).unreadCount, 3);
    expect(ChatChannel.fromJson({'id': 4, 'name': 'family'}).unreadCount, 0);
  });

  // #3257: a conversation is a private channel named for both people.
  test('names the other person in a private conversation', () {
    const dm = ChatChannel(id: 1, name: 'grace, ux-test', isPrivate: true);

    expect(dm.directPartner('ux-test'), 'grace');
    expect(dm.directPartner('Grace'), 'ux-test');
    expect(dm.directPartner('someone-else'), isNull);
    expect(dm.directPartner(null), isNull);
    // A public channel with the same name, or a group of three, is not one.
    expect(
      const ChatChannel(id: 2, name: 'grace, ux-test').directPartner('ux-test'),
      isNull,
    );
    expect(
      const ChatChannel(
        id: 3,
        name: 'a, b, ux-test',
        isPrivate: true,
      ).directPartner('ux-test'),
      isNull,
    );
  });

  test('reads a read marker', () {
    final marker = ChatReadMarker.fromJson({
      'channelId': 4,
      'lastReadMessageId': 123,
      'unreadCount': 0,
    });

    expect(marker.channelId, 4);
    expect(marker.lastReadMessageId, 123);
    expect(marker.unreadCount, 0);
  });
}
