export function titleFromFilename(filename) {
  const base = (filename ?? '').split('/').pop() || filename;
  return base.replace(/\.md$/i, '').trim();
}

/** Strip optional YAML frontmatter, pulling out a `url:` field if present. */
export function parseFrontmatter(rawContent) {
  const raw = rawContent ?? '';
  if (!raw.startsWith('---')) {
    return { sourceUrl: '', body: raw.trim() };
  }

  const end = raw.indexOf('\n---', 3);
  if (end === -1) {
    return { sourceUrl: '', body: raw.trim() };
  }

  const frontmatter = raw.slice(3, end);
  const body = raw.slice(end + 4).trimStart();
  const urlMatch = frontmatter.match(/^url:\s*["']?([^"'\n]+)/m);
  return { sourceUrl: urlMatch?.[1]?.trim() ?? '', body: body.trim() };
}

/** Normalize an arbitrary markdown file (title from filename, optional frontmatter) for Noteika save. */
export function prepareMarkdownNote(filename, rawContent) {
  const title = titleFromFilename(filename);
  const { sourceUrl, body: stripped } = parseFrontmatter(rawContent);
  let body = stripped;

  if (body && !body.toLowerCase().startsWith(title.toLowerCase())) {
    body = `${title}\n\n${body}`;
  } else if (!body) {
    body = title;
  }

  return { title, body, sourceUrl };
}

export function filterMarkdownFiles(fileList) {
  return Array.from(fileList)
    .filter((f) => f.name.toLowerCase().endsWith('.md'))
    .sort((a, b) => (a.webkitRelativePath || a.name).localeCompare(b.webkitRelativePath || b.name));
}
