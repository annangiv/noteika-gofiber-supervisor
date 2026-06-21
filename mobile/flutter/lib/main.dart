import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'screens/inbox_screen.dart';
import 'screens/login_screen.dart';
import 'screens/vault_unlock_screen.dart';
import 'screens/about_screen.dart';
import 'screens/pricing_screen.dart';
import 'screens/account_screen.dart';
import 'screens/landing_screen.dart';
import 'services/noteika_api.dart';
import 'state/app_state.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final api = await NoteikaApi.create();
  runApp(NoteikaApp(api: api));
}

class NoteikaApp extends StatelessWidget {
  const NoteikaApp({super.key, required this.api});

  final NoteikaApi api;

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (_) => AppState(api)..bootstrap(),
      child: MaterialApp(
        title: 'Noteika',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          brightness: Brightness.dark,
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF8B5CF6),
            brightness: Brightness.dark,
            primary: const Color(0xFF8B5CF6),
            surface: const Color(0xFF13151A),
          ),
          scaffoldBackgroundColor: const Color(0xFF0A0B0D),
          cardColor: const Color(0xFF13151A),
          appBarTheme: const AppBarTheme(
            backgroundColor: Color(0xFF0A0B0D),
            elevation: 0,
            centerTitle: false,
            titleTextStyle: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold),
            iconTheme: IconThemeData(color: Colors.white),
          ),
          inputDecorationTheme: InputDecorationTheme(
            filled: true,
            fillColor: const Color(0xFF13151A),
            contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(color: Color(0xFF1F2228), width: 1),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(color: Color(0xFF1F2228), width: 1),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(color: Color(0xFF8B5CF6), width: 1.5),
            ),
            labelStyle: const TextStyle(color: Color(0xFF9CA3AF), fontSize: 14),
            hintStyle: const TextStyle(color: Color(0xFF6B7280), fontSize: 14),
          ),
          elevatedButtonTheme: ElevatedButtonThemeData(
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFF7C3AED),
              foregroundColor: Colors.white,
              elevation: 0,
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 20),
              textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
            ),
          ),
          outlinedButtonTheme: OutlinedButtonThemeData(
            style: OutlinedButton.styleFrom(
              foregroundColor: Colors.white,
              side: const BorderSide(color: Color(0xFF1F2228)),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 20),
              textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
            ),
          ),
          textButtonTheme: TextButtonThemeData(
            style: TextButton.styleFrom(
              foregroundColor: const Color(0xFFA78BFA),
              textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
            ),
          ),
        ),
        home: const _RootGate(),
        routes: {
          '/landing': (_) => const LandingScreen(),
          '/login': (_) => const LoginScreen(),
          '/vault': (_) => const VaultUnlockScreen(),
          '/inbox': (_) => const InboxScreen(),
          '/about': (_) => const AboutScreen(),
          '/pricing': (_) => const PricingScreen(),
          '/account': (_) => const AccountScreen(),
        },
      ),
    );
  }
}

class _RootGate extends StatelessWidget {
  const _RootGate();

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();

    if (state.authLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }
    if (!state.isLoggedIn) {
      if (!state.hasSeenLanding) {
        return const LandingScreen();
      }
      return const LoginScreen();
    }
    if (!state.isUnlocked) return const VaultUnlockScreen();
    return const InboxScreen();
  }
}
