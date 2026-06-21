#!/usr/bin/env bash
# One-time setup: generate android/ + ios/ and allow HTTP to local Noteika server.
set -euo pipefail
cd "$(dirname "$0")"

echo "Generating Flutter platform projects (android, ios)..."
flutter create . --project-name noteika_mobile

ANDROID_MANIFEST="android/app/src/main/AndroidManifest.xml"
if [[ -f "$ANDROID_MANIFEST" ]] && ! grep -q 'usesCleartextTraffic' "$ANDROID_MANIFEST"; then
  echo "Enabling cleartext HTTP for Android emulator (10.0.2.2:8080)..."
  sed -i '' 's/<application/<application android:usesCleartextTraffic="true"/' "$ANDROID_MANIFEST"
fi

IOS_PLIST="ios/Runner/Info.plist"
if [[ -f "$IOS_PLIST" ]] && ! grep -q 'NSAppTransportSecurity' "$IOS_PLIST"; then
  echo "Enabling local HTTP for iOS simulator..."
  /usr/libexec/PlistBuddy -c 'Add :NSAppTransportSecurity dict' "$IOS_PLIST" 2>/dev/null || true
  /usr/libexec/PlistBuddy -c 'Add :NSAppTransportSecurity:NSAllowsLocalNetworking bool true' "$IOS_PLIST" 2>/dev/null || true
fi

echo ""
echo "Done. Run the app:"
echo "  Android emulator: flutter run --dart-define=NOTEIKA_API_URL=http://10.0.2.2:8080"
echo "  iOS simulator:    flutter run --dart-define=NOTEIKA_API_URL=http://localhost:8080"
