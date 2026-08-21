import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/router.dart';
import 'package:quark/services/app_settings.dart';

/// Terms and Conditions acceptance gate.
///
/// Shown on first launch (or after a reset) before the user can access the app.
/// Tapping "I Agree" persists the acceptance via [AppSettings] and navigates
/// to the main file-browser view.
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
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    await AppSettings.instance.acceptTerms();
                    if (context.mounted) {
                      context.go(AppRoutes.cirrus);
                    }
                  },
                  child: const Padding(
                    padding: EdgeInsets.symmetric(vertical: 4),
                    child: Text('I Agree', style: TextStyle(fontSize: 16)),
                  ),
                ),
              ),
            ),
          ),
        ],
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
