import 'package:flutter_test/flutter_test.dart';
import 'package:quark/router.dart';

/// Where the router sends source, config and log files (#2991).
void main() {
  test('codeViewerPath encodes the path and carries the serial', () {
    expect(
      AppRoutes.codeViewerPath('/my app/main.go'),
      '/code/my%20app/main.go',
    );
    expect(
      AppRoutes.codeViewerPath('logs/server.log', serial: 'usb 1'),
      '/code/logs/server.log?serial=usb+1',
    );
  });

  test('a /view link to source or a log opens the code viewer', () {
    expect(viewFileRedirect('src/main.go', null), '/code/src/main.go');
    expect(
      viewFileRedirect('logs/server.log', 's1'),
      '/code/logs/server.log?serial=s1',
    );
  });

  test('a /view link to plain text opens the code viewer too (#3186)', () {
    expect(viewFileRedirect('notes/todo.txt', null), '/code/notes/todo.txt');
  });
}
