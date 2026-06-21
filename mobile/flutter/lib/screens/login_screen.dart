import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:path_provider/path_provider.dart';
import 'package:provider/provider.dart';

import '../config/api_config.dart';
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

  Future<void> _startOAuth(String provider) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();

      if (provider == 'google') {
        final googleSignIn = GoogleSignIn(
          scopes: ['email', 'profile'],
          serverClientId: ApiConfig.googleServerClientId.isEmpty ? null : ApiConfig.googleServerClientId,
        );

        debugPrint('LoginScreen: Initiating native Google Sign-In...');
        final GoogleSignInAccount? account = await googleSignIn.signIn();
        if (account == null) {
          // User cancelled login
          setState(() => _loading = false);
          return;
        }

        final GoogleSignInAuthentication auth = await account.authentication;
        final String? idToken = auth.idToken;
        if (idToken == null) {
          throw Exception('Could not retrieve Google ID Token.');
        }

        debugPrint('LoginScreen: Native Google auth successful, exchanging token with backend...');
        final res = await state.api.googleNativeLogin(idToken);
        final token = res['token'] as String?;

        if (token != null && token.isNotEmpty) {
          final dir = await getApplicationDocumentsDirectory();
          final jar = PersistCookieJar(storage: FileStorage('${dir.path}/.cookies/'));
          final apiUri = Uri.parse(state.api.baseUrl);
          final cookie = Cookie('keller_session', token)
            ..domain = apiUri.host
            ..path = '/';
          await jar.saveFromResponse(apiUri, [cookie]);

          await state.bootstrap();
          if (mounted && state.isLoggedIn) {
            Navigator.of(context).pushReplacementNamed('/vault');
          }
        } else {
          setState(() => _error = 'Native Google login failed: Session token not received.');
        }
        return;
      }

      final url = '${state.api.baseUrl}/auth/login/$provider?mobile=true';
      debugPrint('LoginScreen: Initiating native OAuth login at URL: $url');
      
      final result = await FlutterWebAuth2.authenticate(
        url: url,
        callbackUrlScheme: 'noteika',
      );
      
      debugPrint('LoginScreen: Native OAuth login completed, result: $result');
      
      final uri = Uri.parse(result);
      final token = uri.queryParameters['token'];

      if (token != null && token.isNotEmpty) {
        final dir = await getApplicationDocumentsDirectory();
        final jar = PersistCookieJar(storage: FileStorage('${dir.path}/.cookies/'));
        final apiUri = Uri.parse(state.api.baseUrl);
        final cookie = Cookie('keller_session', token)
          ..domain = apiUri.host
          ..path = '/';
        await jar.saveFromResponse(apiUri, [cookie]);

        await state.bootstrap();
        if (mounted && state.isLoggedIn) {
          Navigator.of(context).pushReplacementNamed('/vault');
        }
      } else {
        setState(() => _error = 'OAuth login failed: Session token not received.');
      }
    } catch (e) {
      if (mounted) {
        setState(() => _error = 'OAuth login failed: $e');
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
    return Scaffold(
      backgroundColor: const Color(0xFF090A0C),
      appBar: AppBar(
        title: const Text('Sign in to Noteika'),
        backgroundColor: const Color(0xFF090A0C),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const SizedBox(height: 16),
            Center(
              child: Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: const Color(0xFF13151A),
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(color: const Color(0xFF1F2228)),
                ),
                child: ShaderMask(
                  shaderCallback: (bounds) => const LinearGradient(
                    colors: [Color(0xFF8B5CF6), Color(0xFFA78BFA)],
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                  ).createShader(bounds),
                  child: const Icon(
                    Icons.blur_on_rounded,
                    size: 36,
                    color: Colors.white,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 24),
            const Text(
              'Welcome to Noteika',
              style: TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.bold),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            const Text(
              'Sign in to sync your encrypted vault',
              style: TextStyle(color: Color(0xFF9CA3AF), fontSize: 14),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 32),
            OutlinedButton.icon(
              icon: const Icon(Icons.code_rounded, size: 20),
              label: const Text('Sign in with GitHub'),
              onPressed: _loading ? null : () => _startOAuth('github'),
              style: OutlinedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
                side: const BorderSide(color: Color(0xFF1F2228)),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              ),
            ),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              icon: const Icon(Icons.g_mobiledata_rounded, size: 30),
              label: const Text('Sign in with Google'),
              onPressed: _loading ? null : () => _startOAuth('google'),
              style: OutlinedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
                side: const BorderSide(color: Color(0xFF1F2228)),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              ),
            ),
            const SizedBox(height: 40),
            const Row(
              children: [
                Expanded(child: Divider(color: Color(0xFF1F2228))),
                Padding(
                  padding: EdgeInsets.symmetric(horizontal: 16),
                  child: Text('OR DEV SIGN IN', style: TextStyle(color: Color(0xFF6B7280), fontSize: 11, fontWeight: FontWeight.bold, letterSpacing: 0.8)),
                ),
                Expanded(child: Divider(color: Color(0xFF1F2228))),
              ],
            ),
            const SizedBox(height: 28),
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: const Color(0xFF13151A),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFF1F2228)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Local developer sign in:',
                    style: TextStyle(color: Colors.white, fontSize: 14, fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _email,
                    keyboardType: TextInputType.emailAddress,
                    decoration: const InputDecoration(
                      labelText: 'Email',
                      hintText: 'dev-user@example.com',
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _name,
                    decoration: const InputDecoration(
                      labelText: 'Name',
                      hintText: 'Developer User',
                    ),
                  ),
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(_error!, style: const TextStyle(color: Colors.redAccent, fontSize: 13)),
                  ],
                  const SizedBox(height: 20),
                  SizedBox(
                    height: 48,
                    child: ElevatedButton(
                      onPressed: _loading ? null : _devSignIn,
                      style: ElevatedButton.styleFrom(
                        backgroundColor: const Color(0xFF7C3AED),
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      ),
                      child: _loading
                          ? const SizedBox(
                              height: 18,
                              width: 18,
                              child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                            )
                          : const Text('Dev sign in'),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
