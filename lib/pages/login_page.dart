import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/auth_service.dart';
import 'package:quark/utils/connection_error.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/host_manager.dart';
import 'package:quark/widgets/quark_connect_form.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The app's landing page (#1639).
///
/// It doubles as the escape hatch from a Quark you can't reach: every other
/// route is behind the auth gate, so if the configured host is wrong or
/// unreachable this is the only page the user can get to. It therefore owns
/// host management too — connect the first Quark, switch between saved ones,
/// add, edit and remove.
class LoginPage extends StatefulWidget {
  final VoidCallback onLoginSuccess;

  const LoginPage({super.key, required this.onLoginSuccess});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _usernameFocus = FocusNode();
  final _passwordFocus = FocusNode();

  bool _loading = false;
  bool _obscurePassword = true;
  String? _error;

  /// Set when the last sign-in attempt never reached the Quark, so the page
  /// says so plainly instead of rendering a socket error (#1637). This is the
  /// only page that can be reached with an unreachable Quark configured, so it
  /// is where the explanation matters most.
  bool _disconnected = false;

  /// Whether the inline host list is expanded.
  ///
  /// Inline rather than in a dialog or sheet on purpose: switching hosts can
  /// send the router to the terms page, and a route sitting above this one
  /// would be torn down mid-transition (#1623).
  bool _managingHosts = false;

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    _usernameFocus.dispose();
    _passwordFocus.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _loading = true;
      _error = null;
      _disconnected = false;
    });
    try {
      await AuthService.login(
        username: _usernameController.text.trim(),
        password: _passwordController.text,
      );
      if (!mounted) return;
      widget.onLoginSuccess();
    } catch (e) {
      debugPrint('[login_page.dart] Error: $e');
      if (!mounted) return;
      setState(() {
        // An unreachable Quark is not a failed sign-in, and saying so in the
        // credentials banner reads as "wrong password". It gets its own state.
        _disconnected = isQuarkUnreachableError(e);
        _error = _disconnected ? null : Errors.message(e, 'sign in');
        _loading = false;
      });
      // Announce error to screen readers
    }
  }

  void _goToRecover() {
    context.push(AppRoutes.recover);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 400),
              // Rebuilds when a host is connected, switched or edited, so the
              // page flips between "connect a Quark" and "sign in" on its own.
              child: ValueListenableBuilder<String?>(
                valueListenable: AppSettings.instance.activeHostNotifier,
                builder: (context, activeHost, _) => activeHost == null
                    ? _buildConnect()
                    : _buildSignIn(context),
              ),
            ),
          ),
        ),
      ),
    );
  }

  /// Nothing configured yet, so the only useful thing on this page is pointing
  /// the app at a Quark.
  Widget _buildConnect() {
    return QuarkConnectForm(
      onConnected: () {
        if (mounted) setState(() {});
      },
    );
  }

  Widget _buildSignIn(BuildContext context) {
    final theme = Theme.of(context);
    return Form(
      key: _formKey,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Logo / title
          Icon(
            QuarkIcons.home_filled,
            size: 56,
            color: theme.colorScheme.primary,
            semanticLabel: 'Quark',
          ),
          const SizedBox(height: 16),
          Text(
            'Sign in',
            style: theme.textTheme.headlineMedium?.copyWith(
              fontWeight: FontWeight.bold,
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(
            'Enter your credentials.',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 24),

          _buildActiveHostCard(context),
          if (_managingHosts) ...[
            const SizedBox(height: 8),
            HostManager(
              onChanged: () {
                if (mounted) setState(() {});
              },
            ),
          ],
          const SizedBox(height: 24),

          // Error banner
          if (_disconnected) ...[
            QuarkDisconnectedBanner(onRetry: _loading ? null : _submit),
            const SizedBox(height: 16),
          ] else if (_error != null) ...[
            _ErrorBanner(message: _error!),
            const SizedBox(height: 16),
          ],

          // Username
          TextFormField(
            controller: _usernameController,
            focusNode: _usernameFocus,
            decoration: const InputDecoration(
              labelText: 'Username',
              border: OutlineInputBorder(),
              prefixIcon: Icon(QuarkIcons.person_outline),
            ),
            textInputAction: TextInputAction.next,
            autofillHints: const [AutofillHints.username],
            autocorrect: false,
            onFieldSubmitted: (_) {
              FocusScope.of(context).requestFocus(_passwordFocus);
            },
            validator: (v) =>
                (v == null || v.trim().isEmpty) ? 'Username is required' : null,
          ),
          const SizedBox(height: 16),

          // Password
          TextFormField(
            controller: _passwordController,
            focusNode: _passwordFocus,
            obscureText: _obscurePassword,
            decoration: InputDecoration(
              labelText: 'Password',
              border: const OutlineInputBorder(),
              prefixIcon: const Icon(QuarkIcons.lock_outline),
              suffixIcon: IconButton(
                icon: Icon(
                  _obscurePassword
                      ? QuarkIcons.visibility_outlined
                      : QuarkIcons.visibility_off_outlined,
                ),
                tooltip: _obscurePassword ? 'Show password' : 'Hide password',
                onPressed: () {
                  setState(() => _obscurePassword = !_obscurePassword);
                },
              ),
            ),
            textInputAction: TextInputAction.done,
            autofillHints: const [AutofillHints.password],
            onFieldSubmitted: (_) => _loading ? null : _submit(),
            validator: (v) =>
                (v == null || v.isEmpty) ? 'Password is required' : null,
          ),
          const SizedBox(height: 24),

          // Sign in button
          FilledButton(
            onPressed: _loading ? null : _submit,
            child: _loading
                ? const SizedBox(
                    height: 20,
                    width: 20,
                    child: CircularProgressIndicator.adaptive(strokeWidth: 2),
                  )
                : const Text('Sign in'),
          ),
          const SizedBox(height: 12),

          // Forgot password
          TextButton(
            onPressed: _loading ? null : _goToRecover,
            child: const Text('Forgot password?'),
          ),
        ],
      ),
    );
  }

  /// Which Quark these credentials are for, plus the way out if it is the
  /// wrong one — without this the user is stuck on this page (#1639).
  Widget _buildActiveHostCard(BuildContext context) {
    final theme = Theme.of(context);
    final settings = AppSettings.instance;
    final hosts = settings.hosts;
    final index = settings.activeIndex;
    final active = (index >= 0 && index < hosts.length) ? hosts[index] : null;

    return Card(
      margin: EdgeInsets.zero,
      child: ListTile(
        leading: Icon(
          QuarkIcons.storage_outlined,
          color: theme.colorScheme.primary,
        ),
        title: Text(
          active?.name ?? 'Quark',
          style: const TextStyle(fontWeight: FontWeight.w600),
        ),
        subtitle: Text(active?.hostAddress ?? ''),
        trailing: TextButton(
          onPressed: () => setState(() => _managingHosts = !_managingHosts),
          child: Text(_managingHosts ? 'Done' : 'Change'),
        ),
      ),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  final String message;
  const _ErrorBanner({required this.message});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Semantics(
      liveRegion: true,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: theme.colorScheme.errorContainer,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(
              QuarkIcons.error_outline,
              color: theme.colorScheme.onErrorContainer,
              size: 20,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                message,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onErrorContainer,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
