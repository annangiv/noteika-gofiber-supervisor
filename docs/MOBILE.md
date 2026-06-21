# Noteika — Mobile app requirements (Flutter)

Companion to [PRODUCT.md](./PRODUCT.md) (what/why), [E2E.md](./E2E.md) (the encryption + search architecture this must stay compatible with), and [ROADMAP.md](./ROADMAP.md) (Phase E2E-3 stub this expands on).

**Decided:** native Flutter app, not a PWA or React Native wrapper.

---

## Why this is harder than "just port the UI"

The web client isn't a thin frontend over a dumb API — most of the privacy-critical logic runs client-side in JS:

- Vault key derivation (PBKDF2 → AES-GCM key + HKDF key)
- The per-user LSH fingerprint matrix (HKDF-expanded from the vault key) and the project/binarize/pack math that turns a real embedding into the 32-byte fingerprint the server is allowed to see
- The embedding model itself (`BAAI/bge-small-en-v1.5` via transformers.js/ONNX in-browser)
- Encrypting/decrypting note ciphertext and the cached real-embedding blob

None of that exists on the server to fall back on. The mobile app has to reimplement all of it in Dart, and — critically — **bit-for-bit compatibly**, or a user's fingerprints/ciphertexts won't be comparable between their web session and their phone. See "Cross-platform parity" below; this is the main technical risk in this doc.

---

## Scope for v1 (mobile)

Mapped from [PRODUCT.md](./PRODUCT.md)'s "Must have" v1 list — mobile v1 is feature parity on the core loop, not everything the web app has grown since:

**Must have:**
- Vault unlock (passcode entry → derive keys, same as web)
- Quick capture — paste/type, optional project, save
- Semantic search (server-side Hamming narrowing + client exact re-rank, same protocol as web)
- Inbox / list view per project, newest first
- Capture detail — view, edit, delete (soft-delete/trash)
- Duplicate hint on save (reuse the same `findDuplicateMatches`-equivalent flow)

**Explicitly out of scope for v1 (mobile):**
- Account/billing screens (Stripe checkout, plan management) — link out to the web app for those
- Data export
- Project rename/management beyond create + select
- Offline *search* (see "Offline behavior" — offline browse yes, offline semantic search is a stretch goal, not v1)
- Dark mode, polish items already deferred on web

---

## Architecture

- **Framework:** Flutter (Dart), targeting iOS first (matches your dev machine), Android as a near-term follow-on since Flutter gives both from one codebase.
- **Backend:** same Go/Fiber REST API the web client already uses — no new endpoints planned unless auth requires it (see below). Mobile is just another client of `web/captures.go`'s existing contract (see E2E.md's "API sketch").
- **Local storage:** SQLite (`sqflite` package) caching the same shape the server returns — ciphertext, fingerprint, encrypted_vector, timestamps, ids. **Never cache the vault key, the passcode, or decrypted plaintext to disk.** Decryption happens in memory, on demand, only while the vault is unlocked in the current app session — same model as web (vault key lives in memory only, cleared on lock/background-timeout).
- **Crypto:** `cryptography` or `pointycastle` Dart package for AES-256-GCM, PBKDF2 (310,000 iterations, SHA-256), and HKDF — must match `frontend/src/lib/crypto.js`'s exact parameters.
- **Embedding model:** `BAAI/bge-small-en-v1.5` (384-dim) via ONNX, run on-device with `onnxruntime` (or `flutter_onnxruntime`/`fonnx`) — same model family already used in the browser via transformers.js, so the underlying weights match. **The tokenizer must match too** (see below).
- **Fingerprint matrix + binarization:** direct Dart port of `frontend/src/lib/fingerprint.js` — same HKDF info labels, same ±1 matrix construction, same sign/pack-into-32-bytes logic.

---

## Cross-platform parity (the real risk in this doc)

A user's fingerprint for "the same note text" must come out identical whether computed on web or mobile, because the server compares fingerprints from *any* client against each other via Hamming distance — that's the whole point of the architecture in E2E.md. If web and mobile disagree even slightly, search/duplicate-detection silently degrades per-platform instead of failing loudly, which is worse.

Three things must match exactly between the JS and Dart implementations:
1. **PBKDF2 → HKDF → matrix derivation** — same iteration count, same HKDF info label strings, same byte-packing order. Mechanical to get right; needs a shared test vector (same passcode + salt → same vault key, same hkdf key, same matrix bytes, checked against both implementations).
2. **The embedding model's tokenizer** — BGE uses a BERT-style WordPiece tokenizer. transformers.js's JS tokenizer and whatever Dart/ONNX tokenizer library is used must produce identical token ids for identical input text (including the `query: `/`passage: ` prefixes — see `frontend/src/lib/embeddings.js`), or the embeddings (and therefore fingerprints) will differ for the same text on different platforms.
3. **The embedding model's pre/post-processing** — same pooling (`mean`) and normalization as `embeddings.js`'s `pooling: 'mean', normalize: true`.

**Recommended first milestone, before building any UI:** a standalone Dart script that takes a fixed passcode + salt + note text and prints the fingerprint bytes, run against the same inputs through the existing JS modules, and diffed byte-for-byte. Don't proceed past this until it matches — everything else in this doc assumes it does.

---

## Auth

Web uses an httpOnly cookie session (`keller_session`, set by `web/auth.go` after OAuth redirect). Native apps don't get free cookie-jar behavior the way a browser tab does. Two options, pick one before building the login screen:

- **(A) Cookie jar in the HTTP client.** Use `dio` + `cookie_jar`/`dio_cookie_manager`, drive the OAuth redirect through an in-app browser (`flutter_web_auth_2`, which uses `ASWebAuthenticationSession` on iOS), capture the `Set-Cookie` from the callback response, and persist it in the cookie jar for subsequent requests. **No backend changes needed.**
- **(B) Bearer token.** Add a token-issuing path to the existing OAuth callback (alongside, not instead of, the cookie) so mobile can store a long-lived token via `flutter_secure_storage` and send `Authorization: Bearer <token>` instead of relying on cookies. Requires a small backend change but is more idiomatic for native apps and avoids cookie-jar fragility (e.g. cookie expiry/renewal edge cases).

Recommendation: start with (A) since it needs zero backend changes and this is v1 of a personal-use app; revisit (B) if cookie handling turns out flaky in practice.

**Open decision — flag before building:** is this app going on the App Store/Play Store, or is it for your own device(s) only (sideload via Xcode + a free/personal Apple Developer account, or TestFlight to your own device)? This affects nothing architecturally but affects whether you need privacy-policy/App-Store-review-grade polish at all for v1.

---

## Offline behavior

- **Browse offline:** yes — SQLite caches ciphertext/fingerprint/encrypted_vector for captures already synced; once the vault is unlocked (passcode entered this session), cached captures decrypt and display without network.
- **Capture offline:** yes — save locally to SQLite with a `pending_sync` flag, encrypt+fingerprint locally (everything needed for that already runs on-device), push to the server when connectivity returns.
- **Search offline:** stretch goal, not v1. Full semantic search offline would mean replicating the server's Hamming-narrowing step locally over the SQLite cache (cheap — same XOR+popcount math, just in Dart instead of Go) — actually feasible since the fingerprint comparison itself is trivial; the harder part is keeping the local fingerprint index in sync with everything the user has saved from *other* devices. Decide later whether this is worth it for v1.5.
- **Conflict resolution:** last-write-wins on `updated_at`, same as the implicit behavior today — no multi-device concurrent-edit story planned for v1 (matches "personal use, mostly one device active at a time" framing).

---

## Suggested phased plan

1. **Crypto/embedding parity spike** (see above) — prove fingerprint byte-equality with web before anything else.
2. **Auth** — OAuth login working end-to-end against the existing backend.
3. **Read-only sync** — fetch + decrypt + list captures, no save yet.
4. **Capture + save** — quick capture screen, encrypt + fingerprint + encrypted_vector, POST to existing API.
5. **Search** — wire up the two-stage Hamming-narrow + exact-rerank flow against `/api/captures/search`.
6. **Offline cache + background sync.**
7. **Polish** — duplicate hints, trash/delete, project management parity.

---

## Open questions to resolve before/while building

- App Store distribution vs personal-device-only (see Auth section).
- Auth: cookie jar (A) vs bearer token (B) — leaning (A) for v1.
- Which Dart ONNX runtime package actually runs BGE-small acceptably on-device (model size, cold-load time, inference latency) — needs a real-device spike, not just a simulator check.
- Which Dart tokenizer library can be made to match transformers.js's WordPiece tokenization exactly — may need a custom/vendored tokenizer rather than trusting an off-the-shelf package's defaults.
- Whether to add the bearer-token backend path now or defer until cookie-jar approach proves insufficient.
