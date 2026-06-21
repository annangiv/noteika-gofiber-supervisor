# Noteika Mobile

Two paths to get Noteika on your phone:

| Approach | Effort | Best for |
|----------|--------|----------|
| **[Flutter](./flutter/)** (recommended) | Medium — native UI, ports crypto/API | Long-term; already on [roadmap Phase E2E-3](../docs/ROADMAP.md) |
| **Capacitor** (optional) | Low — wraps existing React app | Fastest full-feature parity (embeddings + search work today) |

## What the app does (v1)

- **Quick capture** — paste anything, optional project, one tap save
- **Encrypted vault** — same PBKDF2 + AES-GCM as the web client
- **Inbox list** — recent captures, view, trash
- **OAuth login** — GitHub/Google (mock in dev)

Semantic search needs on-device BGE embeddings (web has this via transformers.js; Flutter ONNX is next).

## Quick start — Flutter

```bash
# Backend must be running
docker compose up -d

# Install Flutter: https://docs.flutter.dev/get-started/install
cd mobile/flutter
flutter pub get
flutter run
```

See [flutter/README.md](./flutter/README.md) for emulator URLs and build flags.

## Quick start — Capacitor (optional)

Capacitor config lives in `frontend/capacitor.config.cjs`. After installing Xcode/Android Studio:

```bash
cd frontend
npm install
npm run cap:init    # first time: adds ios/ + android/
npm run cap:ios     # or cap:android
```

Uses **remote-server mode** — the shell loads `http://localhost:8080` so the full React app (including semantic search) runs unchanged.

## Your server is running

The Go backend is at **http://localhost:8080** (Docker). Use mock OAuth on login for local testing.
