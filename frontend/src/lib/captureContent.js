export function generateAutoTitle(body) {
  const cleaned = (body ?? '').trim();
  if (!cleaned) return 'Untitled Capture';

  const lines = cleaned.split('\n');
  let firstLine = '';
  for (const l of lines) {
    const t = l.trim();
    if (t && !t.startsWith('```')) {
      firstLine = t;
      break;
    }
  }

  if (!firstLine) {
    for (const l of lines) {
      const t = l.trim();
      if (t) {
        firstLine = t.replace(/`/g, '');
        break;
      }
    }
  }

  if (!firstLine) firstLine = 'Untitled Capture';

  if (firstLine.length > 80) {
    const sub = firstLine.slice(0, 80);
    const lastSpace = sub.lastIndexOf(' ');
    if (lastSpace > 60) return `${firstLine.slice(0, lastSpace)}...`;
    return `${sub}...`;
  }
  return firstLine;
}

export function classifyContentType(body) {
  const trimmed = (body ?? '').trim();
  if (!trimmed) return 'note';

  if (
    (trimmed.startsWith('http://') || trimmed.startsWith('https://'))
    && !trimmed.includes('\n')
    && !trimmed.includes(' ')
  ) {
    return 'link';
  }
  if (trimmed.includes('```')) return 'code';

  const lower = trimmed.toLowerCase();
  if (
    (lower.includes('q:') && lower.includes('a:'))
    || (lower.includes('user:') && lower.includes('assistant:'))
    || (lower.includes('user:') && lower.includes('ai:'))
    || (lower.includes('human:') && lower.includes('assistant:'))
  ) {
    return 'qa';
  }
  return 'note';
}

export function parseTagsInput(value) {
  if (!value?.trim()) return [];
  return value.split(',').map((t) => t.trim()).filter(Boolean);
}

export function formatTagsInput(tags) {
  if (!Array.isArray(tags) || tags.length === 0) return '';
  return tags.join(', ');
}

export function parseHashtags(body) {
  const re = /#([a-zA-Z][a-zA-Z0-9_-]{1,31})/g;
  const seen = new Set();
  const tags = [];
  let m = re.exec(body ?? '');
  while (m) {
    const n = m[1].toLowerCase();
    if (!seen.has(n)) {
      seen.add(n);
      tags.push(n);
    }
    m = re.exec(body ?? '');
  }
  return tags.sort();
}

export function mergeTags(...lists) {
  const seen = new Set();
  const out = [];
  for (const list of lists) {
    for (const t of list ?? []) {
      const n = t.trim().toLowerCase().replace(/^#/, '');
      if (n.length >= 2 && !seen.has(n)) {
        seen.add(n);
        out.push(n);
      }
    }
  }
  return out.sort().slice(0, 12);
}
