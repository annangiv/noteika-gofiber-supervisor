# Noteika — Manual test scenarios

Fresh start: wipe local data, rebuild, then walk through these cases.

```bash
rm -rf data && mkdir -p data
docker compose up -d --build
```

Open **http://localhost:8080** — hard refresh (`Cmd+Shift+R`) after each deploy.

---

## Setup (every run)

1. Sign in with GitHub or Google.
2. On `/notes`, **create a vault passcode** (8+ chars). Remember it — not recoverable.
3. Wait for **“Loading semantic search model…”** to finish (first time downloads ~30MB).
4. Confirm header badge: **Encrypted · semantic search**.

---

## Scenario 1 — Basic encrypted save

**Goal:** Note body never stored readable on server; card shows after decrypt.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Project: `Inbox`. Tags: `groceries, meal-planning`. Paste body below. | Auto-title = first line |
| 2 | Click **Capture** | Toast success; card in feed |
| 3 | Refresh page, unlock vault | Same note visible |

**Body (note A):**
```
Weekly grocery run — breakfast shopping list
- eggs, oat milk, berries
- coffee, butter
- spinach for smoothies
#groceries #breakfast
```

---

## Scenario 2 — Semantic search (search bar)

**Goal:** Client embeds query; server ranks encrypted vectors; no query text stored server-side.

| Step | Action | Expected |
|------|--------|----------|
| 1 | With note A saved, search: `breakfast shopping` | Note A in results, high % match |
| 2 | Search: `breakfast` | Note A likely appears (may need lower sensitivity — see Scenario 8) |
| 3 | Search: `RustFS S3 IAM` | No results (nothing saved yet) |

---

## Scenario 3 — True duplicate (should warn)

**Goal:** Embedding similarity between paraphrased grocery notes.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Start typing note B (below) in capture form | Yellow **Already in your docket** panel while typing (≥65%) |
| 2 | Click **Capture** | Save blocked or **Already in your docket** modal (≥78%) |
| 3 | Choose **Save anyway** (if modal) | Saves; both notes in feed |

**Body (note B):**
```
Need to pick up stuff for morning meals — eggs, oat milk, fruit, coffee
#groceries #breakfast
```

---

## Scenario 4 — Unrelated note (should NOT duplicate)

**Goal:** Unrelated content must not flag against note A.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Project: `RustFS`. Paste note C. Capture. | Saves with **no** duplicate warning |
| 2 | While typing another unrelated note D | No warning pointing at note A |

**Body (note C):**
```
Fixed RustFS ListObjects returning 403 on prefix listing
Root cause: IAM policy missing s3:ListBucket on bucket ARN
Test with aws s3 ls s3://my-bucket/prefix/
#rustfs #s3 #iam
```

**Body (note D — optional):**
```
Landing page hero: "Notes that resurface before you duplicate yourself."
Sub: Save once. Find by meaning, not filename.
#copy #landing
```
Project: `Prompts`

---

## Scenario 5 — Projects & tags

**Goal:** Plain project names on server; tags inside encrypted blob.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Save note E in project `Client-X` | Sidebar shows `Client-X` |
| 2 | Tags field: `oauth, github`. Body with `#oauth` | Tag chips on card after save |
| 3 | Switch sidebar to `Inbox` vs `Client-X` | Feed filters correctly |

**Body (note E):**
```
GitHub OAuth app setup for Noteika staging
Callback URL: http://localhost:8080/auth/github/callback
Scopes: read:user, user:email
#oauth #github #staging
```

---

## Scenario 6 — Content types (client auto-detect)

| Type | Body | Expected `type` on card |
|------|------|-------------------------|
| Link | `https://docs.github.com/en/apps/oauth-apps` | link card |
| Code | fenced Go snippet | code block preview |
| Q&A | `User: …` / `Assistant: …` | Q&A layout |

---

## Scenario 7 — Edit & trash

| Step | Action | Expected |
|------|--------|----------|
| 1 | Open note A → edit body → save | Updates; search still finds it |
| 2 | Delete note → soft delete | Moves to Trash |
| 3 | Restore from Trash | Back in project feed |
| 4 | Delete again → Empty Trash | Gone permanently |

---

## Scenario 8 — Search sensitivity (Account)

**Goal:** User-tunable min similarity (50–85%, default 70%).

| Step | Action | Expected |
|------|--------|----------|
| 1 | `/account` → lower slider to ~55% | Broader search results |
| 2 | Search `breakfast` | More likely to surface note A |
| 3 | Raise to ~80% | Stricter; fewer results |

---

## Scenario 9 — Vault lock behavior

| Step | Action | Expected |
|------|--------|----------|
| 1 | Refresh `/notes` | Unlock passcode prompt |
| 2 | Wrong passcode | Error; notes stay hidden |
| 3 | Correct passcode | Feed decrypts and loads |

---

## Scenario 10 — Cross-project search

**Goal:** Search scans **whole docket**; sidebar only filters feed.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Save notes in `Inbox`, `RustFS`, `Client-X` | — |
| 2 | Select `Inbox` in sidebar | Feed shows Inbox only |
| 3 | Search `OAuth GitHub` | Can return `Client-X` note even while Inbox selected |
| 4 | Banner text | “Searching all projects…” |

---

## Quick checklist

- [ ] Vault create + unlock
- [ ] Save encrypted note; refresh + unlock shows content
- [ ] Search finds related note by meaning
- [ ] Paraphrase duplicate warns/blocks
- [ ] Unrelated note saves clean
- [ ] Projects + tag chips
- [ ] Trash restore / empty
- [ ] Account search sensitivity
- [ ] Delete button text visible (modals / account)

---

## Troubleshooting

| Symptom | Check |
|---------|--------|
| Search always empty | Model loaded? Re-save notes after fresh `data/` wipe (fingerprints stored on save). |
| Every note flags as duplicate | Duplicate check uses **query↔passage** (not passage↔passage). Re-save notes after deploy. Unrelated notes (scenario 4) should not warn. |
| Stale UI | Hard refresh after `docker compose up -d --build`. |

---

## Architecture reminder (what you’re testing)

- **Encrypted on client:** title, body, tags, source URL  
- **Plain over TLS:** opaque binarized embedding fingerprint only — never the real embedding  
- **No server-side embedding key:** fingerprints are non-reversible by construction, nothing to encrypt at rest  
- **Search / duplicates:** client embed + fingerprint → server Hamming-distance candidate narrowing → client decrypt + re-embed + exact cosine re-rank  

See [E2E.md](./E2E.md) for full design.
