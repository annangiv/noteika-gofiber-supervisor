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
| Embedding vector | **Encrypted at rest** (server key); plain in RAM during search | **Yes at search time** | Semantic search, duplicates, resurfacing |
| Tags (optional) | Ciphertext or client-only | Prefer encrypt with note | Filters, display |
| Project name | Plain (v1) | Yes | Sidebar filter |
| Timestamps, IDs | Plain | Yes | Sync, ordering |
| Search query | **Query embedding only** (not text) | Semantic shape, not exact string | Rank captures |

**Honest pitch:** *We cannot read your notes in our database. Search uses embedding fingerprints computed on your device; we match them to power resurfacing. Fingerprints are encrypted at rest.*

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
  │ POST /api/captures/search     │
  │   { query_embedding, … }      │
  │ ─────────────────────────────►│ decrypt stored vectors (server key)
  │                               │ rank by cosine (+ optional text hash)
  │◄── [{ id, similarity,         │
  │      ciphertext }] ───────────│
  │ decrypt locally               │
  │ show cards / resurface UI     │
```

- **Query text never sent** to server (privacy win vs today).
- Duplicate-while-typing: client embeds draft → same search endpoint with `exclude_id` / low limit.

### Key management (v1)

| Key | Derivation | Stored where |
|-----|------------|--------------|
| **Note key** | User passcode + per-user salt (scrypt or Argon2id) | Never on server |
| **Server DEK** | Per-user or per-vault data key | Wrapped by server KEK (env/KMS) |

- **Separate encryption password** from OAuth login (Obsidian pattern).
- Lose passcode → data unrecoverable (state clearly in UI).
- Optional export of recovery key (Phase 2).

### Crypto (concrete)

- **Notes:** AES-256-GCM, random nonce per field or single JSON blob per capture.
- **Embeddings at rest:** AES-256-GCM with server-managed DEK (protects DB backup theft, not operator at runtime).
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

- [ ] Encrypt tags with note blob; encrypt project names (optional).
- [ ] Client-side tag suggestion or drop server tag endpoint for encrypted mode.
- [ ] Query embedding only (remove query text path).
- [ ] Audit: no plaintext in logs, crash dumps, error reports.
- [ ] Account export decrypts client-side.

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
  "embedding": [0.12, -0.34, …],
  "project": "Inbox",
  "type": "note"
}
```

Server stores `ciphertext`, encrypts `embedding` for disk, stores `type`/`project`/timestamps plain.

### `POST /api/captures/search`

```json
{
  "query_embedding": […],
  "project": "",
  "min_similarity": 0.7,
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
