import 'package:flutter/material.dart';

/// Password strength level, from weakest to strongest.
enum PasswordStrength { empty, weak, fair, strong, veryStrong }

extension PasswordStrengthDisplay on PasswordStrength {
  String get label => switch (this) {
    PasswordStrength.empty => '',
    PasswordStrength.weak => 'Weak',
    PasswordStrength.fair => 'Fair',
    PasswordStrength.strong => 'Strong',
    PasswordStrength.veryStrong => 'Very strong',
  };

  Color color(ColorScheme cs) => switch (this) {
    PasswordStrength.empty => cs.outline,
    PasswordStrength.weak => const Color(0xFFEF4444), // red-500
    PasswordStrength.fair => const Color(0xFFF97316), // orange-500
    PasswordStrength.strong => const Color(0xFF22C55E), // green-500
    PasswordStrength.veryStrong => const Color(0xFF16A34A), // green-600
  };

  double get fraction => switch (this) {
    PasswordStrength.empty => 0.0,
    PasswordStrength.weak => 0.25,
    PasswordStrength.fair => 0.5,
    PasswordStrength.strong => 0.75,
    PasswordStrength.veryStrong => 1.0,
  };
}

/// Computes a [PasswordStrength] for the given [password].
///
/// Scoring:
///  +1  ≥ 8 characters
///  +1  ≥ 12 characters
///  +1  contains both upper- and lower-case letters
///  +1  contains at least one digit
///  +1  contains at least one special character
///
/// Score 0 → weak, 1 → weak, 2 → fair, 3 → strong, 4–5 → very strong.
PasswordStrength scorePassword(String password) {
  if (password.isEmpty) return PasswordStrength.empty;

  int score = 0;
  if (password.length >= 8) {
    score++;
  }
  if (password.length >= 12) {
    score++;
  }
  if (password.contains(RegExp(r'[A-Z]')) &&
      password.contains(RegExp(r'[a-z]'))) {
    score++;
  }
  if (password.contains(RegExp(r'\d'))) {
    score++;
  }
  if (password.contains(RegExp(r'[^A-Za-z\d]'))) {
    score++;
  }

  return switch (score) {
    0 || 1 => PasswordStrength.weak,
    2 => PasswordStrength.fair,
    3 => PasswordStrength.strong,
    _ => PasswordStrength.veryStrong,
  };
}

/// Animated horizontal bar showing password strength.
///
/// Place this beneath the password field. It is purely visual and does not
/// affect form validation.
///
/// ```dart
/// PasswordStrengthBar(password: _passwordController.text),
/// ```
class PasswordStrengthBar extends StatelessWidget {
  const PasswordStrengthBar({super.key, required this.password});

  final String password;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final strength = scorePassword(password);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(2),
          child: TweenAnimationBuilder<double>(
            tween: Tween(begin: 0.0, end: strength.fraction),
            duration: const Duration(milliseconds: 250),
            curve: Curves.easeOut,
            builder: (context, value, _) {
              return LinearProgressIndicator(
                value: value,
                minHeight: 4,
                backgroundColor: cs.surfaceContainerHighest,
                valueColor: AlwaysStoppedAnimation(strength.color(cs)),
              );
            },
          ),
        ),
        if (strength != PasswordStrength.empty) ...[
          const SizedBox(height: 4),
          Text(
            strength.label,
            style: Theme.of(
              context,
            ).textTheme.labelSmall?.copyWith(color: strength.color(cs)),
          ),
        ],
      ],
    );
  }
}
