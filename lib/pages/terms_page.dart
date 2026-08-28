import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';

/// Terms and Conditions acceptance gate.
///
/// Shown on first launch (or after a reset) before the user can access the app.
/// Tapping "I Agree" persists the acceptance via [AppSettings] and navigates
/// straight to wherever the user belongs next — setup, login, or the file
/// browser.
class TermsPage extends StatelessWidget {
  const TermsPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Terms & Conditions')),
      body: Column(
        children: [
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: const [
                  Text(
                    'Terms and Conditions',
                    style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
                  ),
                  SizedBox(height: 16),
                  Text(
                    'Last updated: June 2026',
                    style: TextStyle(color: Colors.grey),
                  ),
                  SizedBox(height: 24),
                  _TermsSection(
                    title: '1. Personal Use',
                    body:
                        'Quark is designed for personal use to manage '
                        'and back up your own photos and files. You may not '
                        'use the software to store or distribute content that '
                        'violates applicable laws or the rights of others.',
                  ),
                  _TermsSection(
                    title: '2. Your Data',
                    body:
                        'You retain full ownership of all data stored through '
                        'Quark. The app operates on your own hardware; no '
                        'data is uploaded to third-party servers without your '
                        'explicit action. You are solely responsible for the '
                        'security and backup of your data.',
                  ),
                  _TermsSection(
                    title: '3. No Warranty',
                    body:
                        'Quark is provided "as is", without warranty of '
                        'any kind, express or implied. The authors make no '
                        'guarantees regarding uptime, data integrity, or '
                        'fitness for a particular purpose. Use at your own risk.',
                  ),
                  _TermsSection(
                    title: '4. Acceptable Use',
                    body:
                        'You agree not to use Quark for any unlawful '
                        'purpose, to attempt to gain unauthorized access to '
                        'other systems, or to interfere with the operation of '
                        'the software for other users. Because Quark '
                        'runs entirely on your own hardware and we have no '
                        'access to your data or device, these terms are '
                        'legally binding but not technically enforceable by '
                        'us. You are solely responsible for your own '
                        'compliance with applicable laws.',
                  ),
                  _TermsSection(
                    title: '5. Limitation of Liability',
                    body:
                        'To the maximum extent permitted by law, the '
                        'developers of Quark shall not be liable for any '
                        'indirect, incidental, special, or consequential '
                        'damages arising from your use of the software, even '
                        'if advised of the possibility of such damages.',
                  ),
                  _TermsSection(
                    title: '6. Changes to These Terms',
                    body:
                        'We may update these Terms from time to time. '
                        'Continued use of the application after changes are '
                        'posted constitutes your acceptance of the revised '
                        'Terms.',
                  ),
                ],
              ),
            ),
          ),
          const SafeArea(
            child: Padding(
              padding: EdgeInsets.all(24),
              child: SizedBox(width: double.infinity, child: _AgreeButton()),
            ),
          ),
        ],
      ),
    );
  }
}

/// The "I Agree" button.
///
/// Stateful only to hold the in-flight state: accepting terms resolves the
/// next route, which asks the Quark whether it has been set up, and that is a
/// network round-trip the user should see happening.
class _AgreeButton extends StatefulWidget {
  const _AgreeButton();

  @override
  State<_AgreeButton> createState() => _AgreeButtonState();
}

class _AgreeButtonState extends State<_AgreeButton> {
  bool _accepting = false;

  Future<void> _accept() async {
    setState(() => _accepting = true);
    await AppSettings.instance.acceptTerms();
    // Resolved here rather than by going to /files and leaving it to the
    // router's redirect: that redirect silently did nothing when the status
    // call failed, stranding the user on a signed-out file browser (#1624).
    final destination = await destinationAfterAcceptingTerms();
    if (!mounted) return;
    context.go(destination);
  }

  @override
  Widget build(BuildContext context) {
    return FilledButton(
      onPressed: _accepting ? null : _accept,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: _accepting
            ? const SizedBox(
                height: 20,
                width: 20,
                child: CircularProgressIndicator.adaptive(strokeWidth: 2),
              )
            : const Text('I Agree', style: TextStyle(fontSize: 16)),
      ),
    );
  }
}

class _TermsSection extends StatelessWidget {
  final String title;
  final String body;

  const _TermsSection({required this.title, required this.body});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 6),
          Text(body, style: const TextStyle(height: 1.5)),
        ],
      ),
    );
  }
}
