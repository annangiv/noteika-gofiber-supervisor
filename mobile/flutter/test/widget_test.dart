// This is a basic Flutter widget test for NoteikaApp.

import 'package:flutter_test/flutter_test.dart';
import 'package:dio/dio.dart';
import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'package:noteika_mobile/main.dart';
import 'package:noteika_mobile/services/noteika_api.dart';

class FakeNoteikaApi extends NoteikaApi {
  FakeNoteikaApi() : super(Dio(), CookieJar());

  @override
  Future<Map<String, dynamic>?> currentUser() async {
    return null;
  }
}

void main() {
  setUp(() {
    // Initialize mock values for FlutterSecureStorage to prevent method channel calls from hanging
    FlutterSecureStorage.setMockInitialValues({});
  });

  testWidgets('Smoke test - App starts on LandingScreen, and moves to LoginScreen on button tap', (WidgetTester tester) async {
    final api = FakeNoteikaApi();

    await tester.pumpWidget(NoteikaApp(api: api));

    // Let the async auth bootstrap run its microtasks and rebuild the widget tree
    await tester.pump();
    // Settle the landing page entrance animations
    await tester.pumpAndSettle();

    // Verify we are on the Landing Screen
    expect(find.text('Noteika'), findsOneWidget);
    expect(find.text('Prevent Duplicates'), findsOneWidget);

    // Tap "Get Started" to proceed to login
    await tester.tap(find.text('Get Started'));
    await tester.pumpAndSettle();

    // Verify we are now on the Login Screen
    expect(find.text('Sign in to Noteika'), findsOneWidget);
    expect(find.text('Sign in with GitHub'), findsOneWidget);
    expect(find.text('Sign in with Google'), findsOneWidget);
  });
}
