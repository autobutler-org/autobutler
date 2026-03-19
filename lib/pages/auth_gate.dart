import 'package:autobutler/pages/login_page.dart';
import 'package:autobutler/pages/setup_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/auth_service.dart';
import 'package:flutter/material.dart';

/// Wraps the app's main content and enforces authentication.
///
/// On startup it checks [AuthService.checkStatus]:
///   - If setup is not complete → shows [SetupPage]
///   - If setup is complete but no session token → shows [LoginPage]
///   - If a session token exists → renders [child]
///
/// Both [SetupPage] and [LoginPage] call [onAuthenticated] on success,
/// which causes this widget to show [child].
class AuthGate extends StatefulWidget {
  final Widget child;

  const AuthGate({super.key, required this.child});

  @override
  State<AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<AuthGate> {
  _GateState _state = _GateState.loading;

  @override
  void initState() {
    super.initState();
    _check();
  }

  Future<void> _check() async {
    // No host configured — the app has its own "no host" state handling.
    // Skip the auth check entirely; the user will be prompted to add a host
    // before any butler calls are made, so there's nothing to authenticate against.
    if (AppSettings.instance.activeHost == null) {
      if (mounted) setState(() => _state = _GateState.authenticated);
      return;
    }

    // Already authenticated in this session.
    if (AppSettings.instance.sessionToken != null) {
      if (mounted) setState(() => _state = _GateState.authenticated);
      return;
    }

    try {
      final status = await AuthService.checkStatus();
      if (!mounted) return;
      setState(() {
        _state = status.setupComplete ? _GateState.login : _GateState.firstBoot;
      });
    } catch (e) {
      debugPrint('[auth_gate.dart] Error: $e');
      if (!mounted) return;
      // If we can't reach the butler, let the main app handle the error.
      setState(() => _state = _GateState.authenticated);
    }
  }

  void _onAuthenticated() {
    if (mounted) setState(() => _state = _GateState.authenticated);
  }

  @override
  Widget build(BuildContext context) {
    switch (_state) {
      case _GateState.loading:
        return const Scaffold(
          body: Center(child: CircularProgressIndicator.adaptive()),
        );
      case _GateState.firstBoot:
        return SetupPage(onSetupComplete: _onAuthenticated);
      case _GateState.login:
        return LoginPage(onLoginSuccess: _onAuthenticated);
      case _GateState.authenticated:
        return widget.child;
    }
  }
}

enum _GateState { loading, firstBoot, login, authenticated }
