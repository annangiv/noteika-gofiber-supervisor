import 'dart:io';
import 'package:flutter/foundation.dart';

/// Base URL for the Noteika Go server.
/// Override: flutter run --dart-define=NOTEIKA_API_URL=https://example.com
class ApiConfig {
  static const String _rawUrl = String.fromEnvironment(
    'NOTEIKA_API_URL',
    defaultValue: 'http://localhost:8080',
  );

  static String get baseUrl {
    // Android emulator requires 10.0.2.2 to access localhost of the host machine
    if (!kIsWeb && Platform.isAndroid && _rawUrl.contains('localhost')) {
      return _rawUrl.replaceFirst('localhost', '10.0.2.2');
    }
    return _rawUrl;
  }

  static const sessionCookieName = 'keller_session';
}
