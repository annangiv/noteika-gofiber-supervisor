# Noteika — Build roadmap

Derived from [PRODUCT.md](./PRODUCT.md). Check items off as shipped.

---

## Phase E2E — Client encryption + server semantic index

**Goal:** Notes encrypted on client before sync; server never stores readable title/body. Keep semantic search, duplicate detection, and resurfacing via client-generated embeddings (see [E2E.md](./E2E.md)).

**Baseline commit:** `docs: E2E architecture (pre-E2E plaintext baseline)` — last snapshot before implementation.

- [x] Phase E2E-1: crypto envelope, client embed, encrypted storage, search by query vector
- [x] Search: BGE query/passage, client text re-rank for single-word queries
- [x] Duplicate detection: asymmetric query↔passage, raised thresholds
- [ ] Phase E2E-2: hardening, migration, encrypt tags/projects
- [ ] Phase E2E-3: Flutter + SQLite
- [ ] Phase E2E-4: optional title model, recovery key

---

## Phase 0 — Foundation ✅

- [x] Scaffold app from Keller Actix template (React)
- [x] OAuth auth (GitHub + Google)
- [x] Supervisor + VaultActor plumbing
- [x] Product vision documented

---

## Phase 1 — Capture & store (MVP)

**Goal:** Save anything in one box; list captures by project.

### Backend

- [x] `captures` table — id, user_id, project, title, body, source_url, created_at, updated_at
- [x] `projects` table (or enum on capture) — user-defined project names
- [x] VaultActor messages: `CreateCapture`, `ListCaptures`, `GetCapture`, `UpdateCapture`, `DeleteCapture`
- [x] REST API:
  - `POST /api/captures`
  - `GET /api/captures?project=`
  - `GET /api/captures/:id`
  - `PATCH /api/captures/:id`
  - `DELETE /api/captures/:id`

### Frontend

- [x] **Inbox page** (`/inbox`) — capture form + recent list
- [x] **Capture detail** — view / edit / delete
- [x] Project selector (dropdown + "new project")
- [x] Replace template home page marketing with logged-in redirect to inbox

### UX rules

- No type picker on save
- Auto-generate title from first ~80 chars of body (simple heuristic first; AI later)
- Default project: "Inbox" or last-used project

---

## Phase 2 — Semantic search

**Goal:** Find captures by meaning, not keywords.

### Backend

- [ ] Generate embedding on capture create/update (local model or API — TBD)
- [ ] Store embedding in Stoolap vector column (HNSW)
- [ ] `SearchCaptures` message + `GET /api/captures/search?q=`
- [ ] Scope search by project (optional filter)

### Frontend

- [ ] Search bar on inbox — "What are you looking for?"
- [ ] Results ranked by similarity with snippet + project + date
- [ ] Empty state: "Nothing yet — paste something above"

---

## Phase 3 — Resurface

**Goal:** Proactive "you already have this" before duplicate work.

- [x] While typing / on save: duplicate warning ("Already in your docket")
- [x] Search results ranked by similarity
- [ ] ~~On project open: feed resurface panels~~ — removed (too noisy; similarity only on save + search)
- [ ] Optional: browser extension or bookmarklet for one-click capture (later)

---

## Phase Launch — Hosting, gate & billing

**Goal:** Deploy on your server; prevent abuse; optional paid upgrade. No decryption for limits — count rows per `user_id` only.

### Abuse gate (ship first)

- [ ] `User.Plan` (`free` | `pro`) + optional `stripe_customer_id` on user record
- [ ] `FREE_CAPTURE_LIMIT` env (e.g. 5 or 10 active captures, trash excluded)
- [ ] Enforce on `POST /api/captures` (and restore?) → `402` + `{ error, upgrade_url, used, limit }`
- [ ] Owner bypass: env `NOTEIKA_OWNER_EMAIL` or manual `plan: pro` in DB
- [ ] Frontend: show "X of N free notes" + upgrade CTA when blocked

### Stripe

- [ ] Stripe Checkout (subscription) or one-time — TBD price
- [ ] Webhook: `checkout.session.completed` / `customer.subscription.updated` → set `plan: pro`
- [ ] Customer Portal link on `/account` for cancel/manage
- [ ] `.env.example`: `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PRICE_ID`, public publishable key for frontend

### Production hosting

- [ ] Real GitHub/Google OAuth apps (redirect URIs for prod domain)
- [ ] `ENCRYPTION_KEY` — unique 32-byte secret (not example value)
- [ ] HTTPS reverse proxy (Caddy/nginx) → container `:8080`
- [ ] Persistent `./data` volume backup strategy
- [ ] Hide or remove `/dev/import` in production (eval only)

### Polish before public

- [ ] Landing + `/pricing` copy aligned with free tier limit
- [ ] Terms / Privacy (ciphertext + vectors stored; vault not recoverable)
- [ ] Error toasts for 402 upgrade path
- [ ] Account page: plan status, usage count, billing link

---

## Phase 4 — Polish

- [ ] Silent content-type detection (link vs Q&A vs prompt) for card layout
- [ ] Dark mode
- [ ] Export captures (JSON / markdown)
- [ ] Mobile-friendly capture UI

---

## Open decisions

| Question | Options | Notes |
|----------|---------|-------|
| Embeddings | Local (fastembed) vs OpenAI API | Local = no API key, one binary |
| Auto-title | Truncate vs small LLM call | Start with truncate |
| Projects | Free text vs managed list | Free text + autocomplete first |

---

## Out of scope (for now)

- Multi-user teams / sharing
- Graph / backlinks view
- In-app AI chat
- Mobile native apps
- noteika.com domain (unavailable as of planning)
