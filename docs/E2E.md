# Noteika — Client encryption & semantic search

> **Status:** Phase E2E-1 in progress (client encrypt + client embed + server encrypted vectors).  
> **Baseline:** The commit immediately before the first E2E implementation PR is the last **plaintext-on-server** snapshot. Look for commit message: `docs: E2E architecture (pre-E2E plaintext baseline)`.

This is **not** Obsidian-style true E2E (server never sees semantic data). It is **client-encrypted content + server-side semantic index** — chosen so Noteika can keep its differentiator: *resurface before you duplicate yourself*.

See also: [PRODUCT.md](./PRODUCT.md), [ROADMAP.md](./ROADMAP.md).

---

## What we protect (and what we don’t)

| Data | On server | Readable by operator? | Purpose |
|------|-----------|------------------------|---------|
| Note title & body | **Ciphertext** (client key) | **No** (without user passcode) | Display after client decrypt |
| Embedding vector | Never sent to server — only an opaque binarized fingerprint (32 bytes) | **No** (fingerprint is non-reversible, no server key exists) | Semantic search, duplicates, resurfacing — see "Embedding privacy v2" below |
| Tags | Ciphertext (with note) | **No** | Filters, display |
| Project name | **Ciphertext** (client key); only an opaque `project_id` is plain | **No** (only that two captures share a project) | Sidebar filter, grouping |
| Timestamps, IDs | Plain | Yes | Sync, ordering |
| Search query | **Query fingerprint only** (not text, not the real embedding) | Approximate similarity shape, not exact string or content | Rank captures |

**Honest pitch:** *We cannot read your notes in our database. Search uses one-way binarized fingerprints computed on your device from your notes' embeddings; we match those fingerprints to power resurfacing, but we never see the real embedding or the note content.*

**Not claimed:** Zero-knowledge, Signal-grade E2E, or “we learn nothing about your notes.”

---

## Current architecture (plaintext baseline)

Before E2E implementation:

- Title, body, tags stored **plaintext** in BadgerDB (`db.Capture`).
- Embeddings computed on **Go backend** via Python sidecar (`POST /embed`).
- Tag suggestions via Python (`POST /suggest-tags`).
- Search: client sends **query text** → server embeds → cosine + text similarity.
- Auth: session cookie; no note encryption key.

---

## Target architecture

### Save flow

```
Client                          Server                         Python sidecar
  │                               │                                  │
  │ 1. auto-title (first line)    │                                  │
  │ 2. encrypt {title, body, …}   │                                  │
  │ 3. embed(title + body) locally│                                  │
  │    (transformers.js / ONNX)   │                                  │
  │                               │                                  │
  │ POST /api/captures            │                                  │
  │   ciphertext + vector ───────►│ store ciphertext                 │
  │                               │ encrypt(vector, server DEK)      │
  │                               │ persist                          │
  │◄── { id, … metadata } ────────│                                  │
```

- **No plaintext note fields** on the wire or disk.
- Python `/embed` on server becomes **optional/dev-only** or removed from save path.
- Tag suggestion: client-side later, or server never sees tags until encrypted blob exists (Phase 2).

### Search / duplicate flow

```
Client                          Server
  │                               │
  │ embed(query) locally          │
  │ fingerprint(embedding)        │
  │ POST /api/captures/search     │
  │   { query_fingerprint, … }    │
  │ ─────────────────────────────►│ no key, no real vector ever held
  │                               │ rank by Hamming distance (approx.)
  │◄── [{ id, similarity (approx),│
  │      ciphertext }] ───────────│
  │ decrypt + re-embed locally    │
  │ re-rank by true cosine        │
  │ show cards / resurface UI     │
```

- **Query text and real embedding never sent** to server — only an opaque fingerprint (privacy win vs today; see "Embedding privacy v2" below for the full mechanism).
- Duplicate-while-typing: client embeds draft → same search endpoint with `exclude_id` / low limit.

### Key management (v1)

| Key | Derivation | Stored where |
|-----|------------|--------------|
| **Note key** | User passcode + per-user salt (scrypt or Argon2id) | Never on server |
| **Server DEK** | Per-user or per-vault data key | Wrapped by server KEK (env/KMS) |

- **Separate encryption password** from OAuth login (Obsidian pattern).
- Lose passcode → data unrecoverable (state clearly in UI).
- Optional export of recovery key (Phase 2).

### Embedding privacy v2 (implemented) — binarized fingerprints, no server key

**Problem with v1**: `NOTEIKA_EMBEDDING_KEY` was one server-held secret that could decrypt every stored embedding back to a real float vector. Embedding-inversion research shows real text content is approximately reconstructable from a raw embedding — so one leaked key was a content-confidentiality failure, not just a metadata one. v1 has been removed entirely (no migration needed — no real users existed yet).

**Two distinct leaks, two different costs to fix** — don't conflate them:
- **Risk A — single-key compromise reveals approximate content for every note.** Fixed by v2 below.
- **Risk B — the party doing the similarity comparison learns the relative similarity structure of your notes** (which notes are alike), from data at rest and from live queries. This is inherent to *any* scheme where a third party narrows candidates by similarity — true zero-knowledge here would need FHE/MPC, which are impractical for real-time search today. Every encrypted-search product on the market (Bitwarden, ProtonMail search, etc.) accepts this same leak. **Not fixed by v2.** The only real fix for Risk B is doing the comparison entirely client-side (no server-side narrowing at all) — a separate, larger project (local-first cache, delta sync, interacts with the full-text search path below); queued separately, not part of v2.

**v2 pipeline** (random-hyperplane LSH / SimHash — similarity-*preserving*, not a cryptographic hash):

1. Passcode + salt → vault key (existing, PBKDF2 — `deriveVaultKeys`).
2. Vault key → a per-user matrix seed, via HKDF (fast expand from the existing strong key; don't re-derive from the raw passcode a second time).
3. Seed → deterministic k×d random matrix `R` (k=256 bits, d=384 for BGE-small), generated by HKDF-expanding the seed directly into matrix bytes (`lib/fingerprint.js`). **Never stored or transmitted** — regenerated on the fly every time it's needed, same as the vault key itself.
4. Multiply: `y = R · embedding` (384-dim embedding → 256 real-valued projections onto random hyperplanes).
5. Binarize: `sign(y)` → a 256-bit vector.
6. Pack those 256 bits into 32 bytes. **This is what gets sent to and stored on the server, as plaintext** — there is no encryption/decryption step at all anymore. Storing it unencrypted is fine specifically because it's the product of a matrix unique per user and derived from the user's passcode: the server never has the matrix, so it can't compute a fingerprint for a guessed phrase itself (no offline dictionary attack on a DB dump), and two different users' fingerprints are never comparable to each other (no cross-user leakage).

**Search**: server does Hamming distance (XOR + popcount) between the query's fingerprint and all of a user's stored fingerprints, returns top-K candidates ranked by `cos(π·hamming/256)` (`rankCapturesByFingerprint` in `web/captures.go`). Client decrypts those K and re-ranks by true cosine similarity for the final precise order and threshold filter (`searchCaptures` in `notesApi.js`) — an approximate-then-exact two-step, same shape as the existing full-text false-positive pass.

**Getting the real embedding back for the exact step without re-running the model**: re-embedding every candidate client-side (the first version of this) cost one ML inference per result on every search — noticeable latency even with a single note, since the server-side narrowing step alone can't avoid it (there's nothing for the server to give back except ciphertext). Fix: alongside the fingerprint, also store the real embedding **encrypted with the vault key** (`db.Capture.EncryptedVector`, AES-GCM over the raw `Float32Array` bytes rather than JSON text — ~4x smaller on the wire). This is still fully E2E — the server stores it but has no key that can decrypt it, so Risk A stays fixed exactly as before. Search candidates now just get decrypted (microseconds) instead of re-embedded (model inference); the fingerprint/Hamming-narrowing step is unchanged and still does the actual job of letting the server bound the candidate set without ever seeing real content.

**Implementation checklist:**
- [x] Client: HKDF-expand vault key → matrix seed → matrix generation; project/binarize/pack helpers (`lib/fingerprint.js`).
- [x] API: capture create/update send the packed fingerprint instead of a raw embedding.
- [x] Server: store fingerprint bytes as-is (`db.Capture.Fingerprint`, replaces `EncryptedEmbedding`/`LegacyEmbedding`); Hamming-distance ranking in Go (XOR + popcount) replaces the old cosine-on-decrypted-vector step.
- [x] Client re-fetches top-K ciphertexts, decrypts a vault-key-encrypted copy of the real embedding sent alongside each candidate (`db.Capture.EncryptedVector` / `lib/crypto.js`'s `encryptEmbedding`/`decryptEmbedding`), and re-ranks by true cosine similarity — no re-embedding model call needed at search time.
- [x] Removed `web/embedding_crypto.go`, `NOTEIKA_EMBEDDING_KEY`, `db.Capture.EncryptedEmbedding`/`LegacyEmbedding` entirely.

### Crypto (concrete)

- **Notes:** AES-256-GCM, random nonce per field or single JSON blob per capture.
- **Embedding fingerprints:** no encryption — opaque 32-byte binarized LSH fingerprint, never reversible to the real embedding even with full server/DB access.
- **Model:** `BAAI/bge-small-en-v1.5` (384-d) on client — same family as today for comparable search quality.

---

## What stays the same (UX)

- Semantic search with % match scores.
- Resurface banners, duplicate warnings, save gate.
- Project sidebar, tags, trash, account export (export = client decrypt → JSON download).
- Docker deploy; Go + React stack.

---

## Implementation phases

### Phase E2E-1 — Crypto envelope (MVP)

- [x] Client: passcode setup/unlock, key derivation, encrypt/decrypt helpers.
- [x] Client: load embedding model (transformers.js); embed on save and search.
- [x] API: accept `ciphertext` + `embedding` on create/update; stop persisting plaintext body/title (new captures).
- [x] API: search accepts `query_embedding` instead of (or in addition to) query text.
- [x] Server: encrypt embeddings at rest before Badger write.
- [x] Server: `Ciphertext` field on `db.Capture` (legacy plaintext fields kept for migration).
- [x] Frontend: unlock gate; decrypt list/detail for display.
- [ ] Migration: re-encrypt existing plaintext captures on first unlock.

### Phase E2E-2 — Hardening

- [x] Encrypt tags with note blob; encrypt project names (opaque `project_id` + encrypted `project:` collection).
- [ ] Client-side tag suggestion or drop server tag endpoint for encrypted mode.
- [x] Query fingerprint only (real embedding and query text never sent to server).
- [x] Embeddings: binarized fingerprints, remove server embedding key — see "Embedding privacy v2" above.
- [ ] Audit: no plaintext in logs, crash dumps, error reports.
- [x] Account export decrypts client-side (now also exports the encrypted `projects` collection).

### Phase E2E-3 — Mobile (Flutter)

- [ ] SQLite local cache + ONNX/TFLite embed.
- [ ] Same ciphertext + vector sync protocol.
- [ ] Background sync; optional offline search against local index.

### Phase E2E-4 — Polish

- [ ] Optional small on-client title model (~20–30MB).
- [ ] Recovery key backup.
- [ ] Independent crypto review / pentest.

---

## API sketch (delta from today)

### `POST /api/captures`

```json
{
  "ciphertext": "<base64 AES-GCM blob: title, body, source_url, tags>",
  "fingerprint": "<base64, 32 bytes — binarized LSH fingerprint, not a real embedding>",
  "project_id": "inbox",
  "type": "note"
}
```

Server stores `ciphertext` and `fingerprint` as-is — no encryption, no server-held key, since the fingerprint is non-reversible. `type`/`project_id`/timestamps stored plain.

### `POST /api/captures/search`

```json
{
  "query_fingerprint": "<base64, 32 bytes>",
  "project_id": "",
  "limit": 20,
  "exclude_id": ""
}
```

Response:

```json
[{
  "capture": {
    "id": "…",
    "ciphertext": "…",
    "project": "Inbox",
    "similarity": 0.82
  }
}]
```

Client decrypts `ciphertext` for UI.

---

## Files likely touched (first PR)

| Area | Files |
|------|--------|
| Docs | `docs/E2E.md`, `docs/ROADMAP.md` |
| Client crypto | `frontend/src/lib/crypto.js` (new) |
| Client embed | `frontend/src/lib/embeddings.js` (new) |
| Notes UI | `frontend/src/pages/NotesPage.jsx`, unlock modal |
| API | `web/captures.go`, `web/server.go` |
| Models | `db/db.go` — ciphertext fields, encrypted embedding storage |
| Python | `embeddings/main.py` — demote from save path; keep for dev/tools |
| Remove | Server `generateAutoTitle` on create; client owns title |

---

## Why not true E2E (Obsidian)?

Obsidian Sync: encrypt → dumb cloud → decrypt → **local search only**. No server vectors.

That forfeits hosted semantic search at scale. Noteika’s job requires **similarity across the vault** without downloading everything to the browser on every search. The chosen compromise:

- **Words:** client-encrypted, server blind in DB.
- **Meaning:** client computes vectors; server matches them; vectors encrypted at rest.

Users who need “server learns nothing” should use Obsidian. Users who want **“intimate notes + resurfacing”** get this model.

---

## Effort estimate

| Task | Time |
|------|------|
| This doc + pre-E2E git baseline | ~15 min |
| Phase E2E-1 (working encrypt + search) | **3–5 days** focused |
| Hardening + migration + edge cases | +2–3 days |
| Flutter parity | separate milestone |

**15 minutes:** documentation and checkpoint commit only — not a working E2E build.

---

## References

- Obsidian Sync E2E: encrypt locally (AES-256-GCM, scrypt), sync ciphertext, search local only.
- Apple Notes ADP: E2E + on-device index.
- Embedding inversion: vectors leak approximate semantics; not exact note recovery — still not equivalent to encrypting bodies.
