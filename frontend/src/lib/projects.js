import { encryptCapturePayload, decryptCapturePayload } from './crypto';

const INBOX = { id: 'inbox', name: 'Inbox' };

/** Fetch + decrypt the user's projects, with the Inbox sentinel pinned first. */
export async function fetchProjects(vaultKey) {
  if (!vaultKey) return [INBOX];
  const res = await fetch('/api/projects');
  if (!res.ok) return [INBOX];
  const raw = await res.json();
  const decrypted = await Promise.all(
    (Array.isArray(raw) ? raw : []).map(async (p) => {
      const { name } = await decryptCapturePayload(vaultKey, p.ciphertext);
      return { id: p.id, name };
    }),
  );
  decrypted.sort((a, b) => a.name.localeCompare(b.name));
  return [INBOX, ...decrypted];
}

/** Create a new encrypted project and return its {id, name}. */
export async function createProject(vaultKey, name) {
  const trimmed = name.trim();
  const id = crypto.randomUUID();
  const ciphertext = await encryptCapturePayload(vaultKey, { name: trimmed });
  const res = await fetch('/api/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, ciphertext }),
  });
  if (!res.ok) throw new Error('Failed to create project');
  return { id, name: trimmed };
}

/**
 * Resolve a typed project name to an id, creating a new project if no
 * existing one matches (case-insensitive). Empty input resolves to Inbox.
 */
export async function resolveProjectId(vaultKey, name) {
  const trimmed = (name ?? '').trim();
  if (!trimmed || trimmed.toLowerCase() === 'inbox') return 'inbox';

  const projects = await fetchProjects(vaultKey);
  const existing = projects.find((p) => p.name.toLowerCase() === trimmed.toLowerCase());
  if (existing) return existing.id;

  const created = await createProject(vaultKey, trimmed);
  return created.id;
}
