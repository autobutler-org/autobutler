import 'package:flutter/material.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/auth_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/core/copy_button.dart';
import 'package:quark/widgets/password_strength_bar.dart';
import 'package:quark_icons/quark_icons.dart';

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
                  ? _ThemeStep(onContinue: widget.onSetupComplete)
                  : _recoveryPhrase != null
                  ? _RecoveryPhraseStep(
                      phrase: _recoveryPhrase!,
                      acknowledged: _phraseAcknowledged,
                      onAcknowledgedChanged: (v) =>
                          setState(() => _phraseAcknowledged = v ?? false),
                      onContinue: _confirmPhraseAndProceed,
                    )
                  : _SetupForm(
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

class _SetupForm extends StatefulWidget {
  final GlobalKey<FormState> formKey;
  final TextEditingController usernameController;
  final TextEditingController passwordController;
  final TextEditingController confirmController;
  final FocusNode usernameFocus;
  final FocusNode passwordFocus;
  final FocusNode confirmFocus;
  final bool obscurePassword;
  final bool obscureConfirm;
  final bool loading;
  final String? error;
  final VoidCallback onTogglePassword;
  final VoidCallback onToggleConfirm;
  final VoidCallback onSubmit;

  const _SetupForm({
    required this.formKey,
    required this.usernameController,
    required this.passwordController,
    required this.confirmController,
    required this.usernameFocus,
    required this.passwordFocus,
    required this.confirmFocus,
    required this.obscurePassword,
    required this.obscureConfirm,
    required this.loading,
    required this.error,
    required this.onTogglePassword,
    required this.onToggleConfirm,
    required this.onSubmit,
  });

  @override
  State<_SetupForm> createState() => _SetupFormState();
}

class _SetupFormState extends State<_SetupForm> {
  void _onPasswordChanged() {
    // Re-validate so the confirm field updates its error state in real-time
    // when the password field changes after the confirm field has been touched.
    setState(() {});
  }

  @override
  void initState() {
    super.initState();
    widget.passwordController.addListener(_onPasswordChanged);
  }

  @override
  void dispose() {
    widget.passwordController.removeListener(_onPasswordChanged);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Form(
      key: widget.formKey,
      autovalidateMode: AutovalidateMode.onUserInteraction,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Icon(
            QuarkIcons.home_filled,
            size: 56,
            color: theme.colorScheme.primary,
            semanticLabel: 'Quark',
          ),
          const SizedBox(height: 16),
          Text(
            'Set up your quark',
            style: theme.textTheme.headlineMedium?.copyWith(
              fontWeight: FontWeight.bold,
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(
            'Create your owner account. This is the only account that can manage the quark.',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 32),

          if (widget.error != null) ...[
            _ErrorBanner(message: widget.error!),
            const SizedBox(height: 16),
          ],

          TextFormField(
            controller: widget.usernameController,
            focusNode: widget.usernameFocus,
            decoration: const InputDecoration(
              labelText: 'Username',
              border: OutlineInputBorder(),
              prefixIcon: Icon(QuarkIcons.person_outline),
            ),
            textInputAction: TextInputAction.next,
            autofillHints: const [AutofillHints.newUsername],
            autocorrect: false,
            onFieldSubmitted: (_) {
              FocusScope.of(context).requestFocus(widget.passwordFocus);
            },
            validator: (v) =>
                (v == null || v.trim().isEmpty) ? 'Username is required' : null,
          ),
          const SizedBox(height: 16),

          TextFormField(
            controller: widget.passwordController,
            focusNode: widget.passwordFocus,
            obscureText: widget.obscurePassword,
            decoration: InputDecoration(
              labelText: 'Password',
              helperText: 'At least 8 characters',
              border: const OutlineInputBorder(),
              prefixIcon: const Icon(QuarkIcons.lock_outline),
              suffixIcon: IconButton(
                icon: Icon(
                  widget.obscurePassword
                      ? QuarkIcons.visibility_outlined
                      : QuarkIcons.visibility_off_outlined,
                ),
                tooltip: widget.obscurePassword
                    ? 'Show password'
                    : 'Hide password',
                onPressed: widget.onTogglePassword,
              ),
            ),
            textInputAction: TextInputAction.next,
            autofillHints: const [AutofillHints.newPassword],
            onFieldSubmitted: (_) {
              FocusScope.of(context).requestFocus(widget.confirmFocus);
            },
            validator: (v) {
              if (v == null || v.isEmpty) return 'Password is required';
              if (v.length < 8) return 'Password must be at least 8 characters';
              return null;
            },
          ),
          const SizedBox(height: 8),
          PasswordStrengthBar(password: widget.passwordController.text),
          const SizedBox(height: 8),

          TextFormField(
            controller: widget.confirmController,
            focusNode: widget.confirmFocus,
            obscureText: widget.obscureConfirm,
            decoration: InputDecoration(
              labelText: 'Confirm password',
              border: const OutlineInputBorder(),
              prefixIcon: const Icon(QuarkIcons.lock_outline),
              suffixIcon: IconButton(
                icon: Icon(
                  widget.obscureConfirm
                      ? QuarkIcons.visibility_outlined
                      : QuarkIcons.visibility_off_outlined,
                ),
                tooltip: widget.obscureConfirm
                    ? 'Show password'
                    : 'Hide password',
                onPressed: widget.onToggleConfirm,
              ),
            ),
            textInputAction: TextInputAction.done,
            autofillHints: const [AutofillHints.newPassword],
            onFieldSubmitted: (_) => widget.loading ? null : widget.onSubmit(),
            validator: (v) {
              if (v == null || v.isEmpty) return 'Please confirm your password';
              if (v != widget.passwordController.text) {
                return 'Passwords do not match';
              }
              return null;
            },
          ),
          const SizedBox(height: 24),

          FilledButton(
            onPressed: widget.loading ? null : widget.onSubmit,
            child: widget.loading
                ? const SizedBox(
                    height: 20,
                    width: 20,
                    child: CircularProgressIndicator.adaptive(strokeWidth: 2),
                  )
                : const Text('Create account'),
          ),
        ],
      ),
    );
  }
}

/// Shows the recovery phrase and requires acknowledgement before proceeding.
class _RecoveryPhraseStep extends StatelessWidget {
  final String phrase;
  final bool acknowledged;
  final ValueChanged<bool?> onAcknowledgedChanged;
  final VoidCallback onContinue;

  const _RecoveryPhraseStep({
    required this.phrase,
    required this.acknowledged,
    required this.onAcknowledgedChanged,
    required this.onContinue,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Icon(
          QuarkIcons.key_rounded,
          size: 56,
          color: theme.colorScheme.primary,
          semanticLabel: 'Recovery phrase',
        ),
        const SizedBox(height: 16),
        Text(
          'Save your recovery phrase',
          style: theme.textTheme.headlineMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        Text(
          'This phrase is the only way to reset your password if you forget it. '
          "It will not be shown again. Write it down somewhere safe — don't store it digitally on this device.",
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 24),

        // Phrase display box
        Semantics(
          label: 'Recovery phrase: $phrase',
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: theme.colorScheme.outlineVariant),
            ),
            child: Column(
              children: [
                SelectableText(
                  phrase,
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontFamily: 'monospace',
                    fontWeight: FontWeight.bold,
                    letterSpacing: 1.2,
                    height: 1.8,
                  ),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 12),
                CopyButton(
                  text: phrase,
                  icon: QuarkIcons.copy_outlined,
                  variant: CopyButtonVariant.outlined,
                  successMessage: 'Recovery phrase copied to clipboard',
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 24),

        // Acknowledgement checkbox
        CheckboxListTile(
          value: acknowledged,
          onChanged: onAcknowledgedChanged,
          title: const Text(
            'I have written down my recovery phrase and stored it safely.',
          ),
          controlAffinity: ListTileControlAffinity.leading,
          contentPadding: EdgeInsets.zero,
        ),
        const SizedBox(height: 16),

        FilledButton(
          onPressed: acknowledged ? onContinue : null,
          child: const Text('Continue'),
        ),
      ],
    );
  }
}

/// Theme selection step — the user picks Light, Dark, or System.
///
/// Selecting an option immediately applies the theme (live preview) and
/// persists the preference via [AppSettings]. The user can proceed with
/// any selection; the default is whatever [AppSettings] loaded on startup
/// (i.e. System on first boot).
class _ThemeStep extends StatelessWidget {
  final VoidCallback onContinue;

  const _ThemeStep({required this.onContinue});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ValueListenableBuilder<ThemeMode>(
      valueListenable: AppSettings.instance.themeMode,
      builder: (context, currentMode, _) {
        return Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Icon(
              QuarkIcons.palette_outlined,
              size: 56,
              color: theme.colorScheme.primary,
              semanticLabel: 'Theme',
            ),
            const SizedBox(height: 16),
            Text(
              'Choose your theme',
              style: theme.textTheme.headlineMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              'You can change this at any time in Settings.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 32),

            _ThemeOption(
              icon: QuarkIcons.brightness_auto_rounded,
              label: 'System',
              description: 'Follows your device setting',
              isSelected: currentMode == ThemeMode.system,
              onTap: () => AppSettings.instance.setThemeMode(ThemeMode.system),
            ),
            const SizedBox(height: 12),
            _ThemeOption(
              icon: QuarkIcons.light_mode_rounded,
              label: 'Light',
              description: 'Always use light theme',
              isSelected: currentMode == ThemeMode.light,
              onTap: () => AppSettings.instance.setThemeMode(ThemeMode.light),
            ),
            const SizedBox(height: 12),
            _ThemeOption(
              icon: QuarkIcons.dark_mode_rounded,
              label: 'Dark',
              description: 'Always use dark theme',
              isSelected: currentMode == ThemeMode.dark,
              onTap: () => AppSettings.instance.setThemeMode(ThemeMode.dark),
            ),

            const SizedBox(height: 32),

            FilledButton(
              onPressed: onContinue,
              child: const Text('Get started'),
            ),
          ],
        );
      },
    );
  }
}

/// A selectable card representing a single theme option.
class _ThemeOption extends StatelessWidget {
  final IconData icon;
  final String label;
  final String description;
  final bool isSelected;
  final VoidCallback onTap;

  const _ThemeOption({
    required this.icon,
    required this.label,
    required this.description,
    required this.isSelected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primaryColor = theme.colorScheme.primary;

    return Semantics(
      button: true,
      selected: isSelected,
      label: '$label theme${isSelected ? ', selected' : ''}',
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          decoration: BoxDecoration(
            color: isSelected
                ? primaryColor.withValues(alpha: 0.08)
                : theme.colorScheme.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: isSelected
                  ? primaryColor
                  : theme.colorScheme.outlineVariant,
              width: isSelected ? 2 : 1,
            ),
          ),
          child: Row(
            children: [
              Icon(
                icon,
                size: 28,
                color: isSelected
                    ? primaryColor
                    : theme.colorScheme.onSurfaceVariant,
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      label,
                      style: theme.textTheme.bodyLarge?.copyWith(
                        fontWeight: FontWeight.w600,
                        color: isSelected
                            ? primaryColor
                            : theme.colorScheme.onSurface,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      description,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              AnimatedOpacity(
                opacity: isSelected ? 1.0 : 0.0,
                duration: const Duration(milliseconds: 150),
                child: Icon(
                  QuarkIcons.check_circle_rounded,
                  color: primaryColor,
                  size: 22,
                ),
              ),
            ],
          ),
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
