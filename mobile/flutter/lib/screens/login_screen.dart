import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../state/app_state.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _email = TextEditingController(text: 'dev-user@example.com');
  final _name = TextEditingController(text: 'Developer User');
  bool _loading = false;
  String? _error;

  Future<void> _devSignIn() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();
      await state.api.devLogin(
        email: _email.text.trim(),
        name: _name.text.trim(),
      );
      await state.bootstrap();
      if (mounted && state.isLoggedIn) {
        Navigator.of(context).pushReplacementNamed('/vault');
      }
    } catch (e) {
      if (mounted) {
        setState(() => _error = e.toString());
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _email.dispose();
    _name.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final base = context.read<AppState>().api.baseUrl;

    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      appBar: AppBar(
        title: const Text('Sign in to Noteika'),
        backgroundColor: const Color(0xFF161B22),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'Google and GitHub sign-in do not work inside the mobile app WebView — '
              'Google blocks embedded browsers for security.',
              style: TextStyle(color: Color(0xFF8B949E), height: 1.45),
            ),
            const SizedBox(height: 8),
            Text(
              'Use dev sign-in below while testing against $base (requires ENVIRONMENT=development on the server).',
              style: const TextStyle(color: Color(0xFF8B949E), height: 1.45),
            ),
            const SizedBox(height: 24),
            TextField(
              controller: _email,
              keyboardType: TextInputType.emailAddress,
              decoration: const InputDecoration(labelText: 'Email'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _name,
              decoration: const InputDecoration(labelText: 'Name'),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.redAccent)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _loading ? null : _devSignIn,
              child: _loading
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Dev sign in'),
            ),
          ],
        ),
      ),
    );
  }
}
