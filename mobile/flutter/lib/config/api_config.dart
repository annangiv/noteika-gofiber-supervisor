import 'dart:io';
import 'package:flutter/foundation.dart';

/// Base URL for the Noteika Go server.
/// Override: flutter run --dart-define=NOTEIKA_API_URL=https://example.com
class ApiConfig {
  static const String _rawUrl = String.fromEnvironment(
    'NOTEIKA_API_URL',
    defaultValue: 'https://noteika.com',
  );

  static String get baseUrl {
    // Android emulator requires 10.0.2.2 to access localhost of the host machine
    if (!kIsWeb && Platform.isAndroid && _rawUrl.contains('localhost')) {
      return _rawUrl.replaceFirst('localhost', '10.0.2.2');
    }
    return _rawUrl;
  }

  static const sessionCookieName = 'keller_session';

  /// Web Client ID from the Google Developer Console, needed by the native Google Sign-in SDK to request a valid backend ID token.
  /// Override: flutter run --dart-define=GOOGLE_SERVER_CLIENT_ID=your-id-here
  static const String googleServerClientId = String.fromEnvironment(
    'GOOGLE_SERVER_CLIENT_ID',
    defaultValue: '545277470147-mpu9i12kdfpmah6hea2qg4gcee5gj32o.apps.googleusercontent.com',
  );

  /// Product identifier for Noteika Pro Monthly Subscription on Google Play Console and App Store Connect.
  /// Override: flutter run --dart-define=PRO_SUBSCRIPTION_ID=noteika_pro_monthly
  static const String proSubscriptionId = String.fromEnvironment(
    'PRO_SUBSCRIPTION_ID',
    defaultValue: 'noteika_pro_monthly',
  );
}
