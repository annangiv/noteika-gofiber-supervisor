# Noteika — Product vision

> **Notes that resurface before you duplicate yourself**

This document captures the product direction agreed in planning (June 2026). It is the source of truth for *what* Noteika is and *why* — not how it is built.

---

## The problem

Most note apps are good at **saving**. They are bad at **bringing notes back when they matter**.

People don't fail because they lack folders or a graph. They fail because:

| Where things go today | Why they disappear |
|----------------------|-------------------|
| AI chats | No memory across sessions; buried in chat history |
| Scratchpads / "bible" notes | Flat text; search only works if you remember the exact keyword |
| Many projects / apps | Same question asked in different places; nothing connects |

You don't lose the *thought* — you lose **where you put it** and **what you already decided**.

That's the **graveyard problem**.

---

## One job (v1)

**Anything you might want again later — save once, find later even if you forgot the name.**

### Real scenario (founder use case)

1. You ask AI for prompts (e.g. landing page for Emergent vs Lovable).
2. You get useful output but don't save it in one trusted place — or you save it without a name you'd remember.
3. A week later a friend wants something similar. You *know* it exists but can't find it.

You're not looking for a file called `emergent-lovable-v2-final`. You're looking for:

> "That landing page prompt I got for those AI app builders… one was more polished, one was more playful…"

That is **meaning search**, not keyword search.

---

## Core principles

### 1. No type picker at save time

When you're copying a prompt from ChatGPT, you won't stop to choose "Prompt vs Q&A vs Link." You paste and move on.

**v1 rule: one box, one save.**

```
[Paste anything here]
[Project ▼]  [Save]
```

Types (prompt, Q&A, link, note) are inferred silently for display only — never required from the user.

### 2. Organize by intent, not folders

You dump things in; the app organizes **by meaning**, not by you picking folders, types, or perfect titles.

**You do:**

- Paste (or type) whatever you got
- Optionally pick a **project** (Keller, Noteika, client work…) — the only org choice that really matters

**The app does:**

- Understands what it's *about* (landing page, OAuth, prompts for Emergent…)
- **Titles it** from the content — you don't name `emergent-v2-final`
- Groups by project + time
- **Surfaces related stuff** when you're working on something similar
- **Finds it when you search loosely** — "that AI builder landing page thing" even if you never wrote "Lovable"

### 3. Resurface before you duplicate

When you open a project or start typing a search, Noteika should answer:

> "You already looked into this — here's what you saved."

Not a passive archive. An active memory that shows up *before* you ask AI again or re-research the same thing.

---

## What you can save (capture types)

All of these are the same thing internally: **a capture** (text you might need again).

| Content | User action | App behavior (silent) |
|---------|-------------|------------------------|
| AI prompt | Paste | Shows as prompt-style card |
| Q&A from AI | Paste | Shows as Q&A layout |
| URL | Paste link | Shows as link, stores URL |
| Random note | Type/paste | Plain note card |
| Mixed / messy | Paste | Still saves; meaning extracted later |

---

## v1 feature set

### Must have

- **Quick capture** — paste anything, optional project tag, save in one click
- **Semantic search** — find by intent ("that OAuth thing I researched") not exact keywords
- **Auto title** — generated from content; user can edit later
- **Project filter** — group captures by project (Keller, Noteika, etc.)
- **Inbox / list view** — recent captures per project, newest first
- **Capture detail** — view full content, edit, delete

### Should have (v1.1)

- **Resurface panel** — "Related to what you're working on" when viewing a project
- **Duplicate hint** — "You saved something similar 5 days ago" when search matches closely
- **Source hint** — optional "where did this come from?" (ChatGPT, Claude, manual)

### Explicitly not v1

- Graph view / bidirectional links (Obsidian territory)
- Team collaboration / sharing
- Full AI chat inside Noteika (save *from* AI elsewhere; find *in* Noteika)
- Complex folder hierarchies
- Requiring tags, types, or perfect titles at capture time

---

## How Noteika is different

| Others | Noteika |
|--------|---------|
| "Second brain" / graph / folders | **Resurface** — active memory before you duplicate work |
| Search = keywords | Search = **meaning** |
| You organize at save time | App organizes by **intent** |
| Built for note-takers who love systems | Built for **builders juggling many projects** who forget where they put things |

---

## Success criteria (personal v1)

Noteika succeeds when the founder can:

1. Paste an AI prompt while switching between 10 apps and find it a week later without remembering the filename.
2. Search "landing page emergent" and get the right capture even if those words weren't in the original text.
3. Open the Noteika project and see recent work without maintaining a folder structure.
4. Stop re-asking AI the same question because Noteika surfaced the old answer first.

---

## Technical note (for builders only)

- Single binary, embedded DB (Stoolap) — no PostgreSQL/Redis required
- Vector search (HNSW) for semantic retrieval
- OAuth auth from Keller Actix template

End users never see or care about this stack.
