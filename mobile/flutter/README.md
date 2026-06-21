# Noteika Flutter

Native iOS + Android client for [Noteika](../../docs/PRODUCT.md). Matches the web app's E2E encryption protocol; semantic embeddings (BGE ONNX) are planned for a follow-up.

## Prerequisites

1. [Flutter SDK](https://docs.flutter.dev/get-started/install) 3.19+
2. Noteika backend running (`docker compose up` → `http://localhost:8080`)

## Setup

```bash
cd mobile/flutter
flutter pub get

# Required once — generates android/ + ios/ (the scaffold only had lib/ code)
chmod +x setup_platforms.sh
./setup_platforms.sh
```

Then launch an emulator and run:

```bash
# Android (emulator already running on emulator-5554)
flutter run --dart-define=NOTEIKA_API_URL=http://10.0.2.2:8080

# iOS simulator
flutter emulators --launch apple_ios_simulator
flutter run --dart-define=NOTEIKA_API_URL=http://localhost:8080
```

If you see *"devices were found, but are not supported by this project"*, you skipped `setup_platforms.sh` (or `flutter create .`).

## Server URL

Default: `http://localhost:8080`. Override at build/run time:

```bash
flutter run --dart-define=NOTEIKA_API_URL=http://10.0.2.2:8080   # Android emulator
flutter run --dart-define=NOTEIKA_API_URL=https://your-domain.com
```

## Features (v0.1)

- OAuth login via in-app WebView (mock GitHub/Google in dev)
- Vault passcode unlock (PBKDF2 + AES-GCM, same as web)
- Inbox list, quick capture, view/edit/delete
- Project filter

## Not yet implemented

- On-device BGE embeddings (saves work; semantic search needs vectors — use web app for full search until ONNX lands)
- Offline SQLite cache ([ROADMAP Phase E2E-3](../../docs/ROADMAP.md))
- Share extension / system share sheet

## Why Flutter over separate native apps?

Single codebase for iOS and Android, already planned in the product roadmap, and crypto/API logic ports cleanly from the React client.
