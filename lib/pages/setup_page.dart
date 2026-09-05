import 'package:flutter/material.dart';
import 'package:quark/services/auth_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/setup/recovery_phrase_step.dart';
import 'package:quark/widgets/setup/setup_form.dart';
import 'package:quark/widgets/setup/theme_step.dart';

/// First-boot setup screen — creates the owner account on the quark.
///
/// Three steps:
///  1. Create account (username + password)
///  2. Acknowledge recovery phrase
///  3. Choose app theme (persisted immediately — live preview)
class SetupPage extends StatefulWidget {
  final VoidCallback onSetupComplete;

  const SetupPage({super.key, required this.onSetupComplete});

  @override
  State<SetupPage> createState() => _SetupPageState();
}

class _SetupPageState extends State<SetupPage> {
  final _formKey = GlobalKey<FormState>();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _confirmController = TextEditingController();
  final _usernameFocus = FocusNode();
  final _passwordFocus = FocusNode();
  final _confirmFocus = FocusNode();

  bool _loading = false;
  bool _obscurePassword = true;
  bool _obscureConfirm = true;
  String? _error;

  // Step 2: recovery phrase acknowledgement
  String? _recoveryPhrase;
  bool _phraseAcknowledged = false;

  // Step 3: theme selection
  bool _showThemeStep = false;

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    _confirmController.dispose();
    _usernameFocus.dispose();
    _passwordFocus.dispose();
    _confirmFocus.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await AuthService.setup(
        username: _usernameController.text.trim(),
        password: _passwordController.text,
      );
      if (!mounted) return;
      setState(() {
        _recoveryPhrase = result.recoveryPhrase;
        _loading = false;
      });
    } catch (e) {
      debugPrint('[setup_page.dart] Error: $e');
      if (!mounted) return;
      setState(() {
        // An unreachable Quark is not a rejected request; saying so plainly
        // beats a socket error in a form's error banner (#1637).
        _error = Errors.message(e, 'set up your Quark');
        _loading = false;
      });
    }
  }

  void _confirmPhraseAndProceed() {
    if (_phraseAcknowledged) {
      setState(() => _showThemeStep = true);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 440),
              child: _showThemeStep
                  ? ThemeStep(onContinue: widget.onSetupComplete)
                  : _recoveryPhrase != null
                  ? RecoveryPhraseStep(
                      phrase: _recoveryPhrase!,
                      acknowledged: _phraseAcknowledged,
                      onAcknowledgedChanged: (v) =>
                          setState(() => _phraseAcknowledged = v ?? false),
                      onContinue: _confirmPhraseAndProceed,
                    )
                  : SetupForm(
                      formKey: _formKey,
                      usernameController: _usernameController,
                      passwordController: _passwordController,
                      confirmController: _confirmController,
                      usernameFocus: _usernameFocus,
                      passwordFocus: _passwordFocus,
                      confirmFocus: _confirmFocus,
                      obscurePassword: _obscurePassword,
                      obscureConfirm: _obscureConfirm,
                      loading: _loading,
                      error: _error,
                      onTogglePassword: () =>
                          setState(() => _obscurePassword = !_obscurePassword),
                      onToggleConfirm: () =>
                          setState(() => _obscureConfirm = !_obscureConfirm),
                      onSubmit: _submit,
                    ),
            ),
          ),
        ),
      ),
    );
  }
}
