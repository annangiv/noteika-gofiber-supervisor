# Noteika — Build roadmap

Derived from [PRODUCT.md](./PRODUCT.md). Check items off as shipped.

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

- [ ] On project open: show "Recent in {project}" + "Similar to your last search"
- [ ] On search: if top result similarity > threshold, banner: "You saved this on {date}"
- [ ] Optional: browser extension or bookmarklet for one-click capture (later)

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
