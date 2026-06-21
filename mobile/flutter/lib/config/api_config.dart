/// Base URL for the Noteika Go server.
/// Override: flutter run --dart-define=NOTEIKA_API_URL=https://example.com
class ApiConfig {
  static const String baseUrl = String.fromEnvironment(
    'NOTEIKA_API_URL',
    defaultValue: 'http://localhost:8080',
  );

  static const sessionCookieName = 'keller_session';
}
