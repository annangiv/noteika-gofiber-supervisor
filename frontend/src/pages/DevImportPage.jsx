import { useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import VaultGate from '../components/VaultGate';
import { useVault } from '../context/VaultContext';
import { saveCapture } from '../lib/saveCapture';
import { filterMarkdownFiles, prepareEvergreenNote } from '../lib/evergreenImport';
import '../notes.css';
import '../dev-import.css';

const IMPORT_PROJECT = 'Evergreen';
const EVERGREEN_FIXTURE_URL =
  'https://github.com/flowing-abyss/obsidian-hybrid-search/tree/master/fixtures/evergreen-notes';

function DevImportInner() {
  const { vaultKey } = useVault();
  const abortRef = useRef(false);

  const [files, setFiles] = useState([]);
  const [limit, setLimit] = useState('');
  const [running, setRunning] = useState(false);
  const [progress, setProgress] = useState({ done: 0, total: 0, current: '' });
  const [errors, setErrors] = useState([]);
  const [finished, setFinished] = useState(false);

  const [goldenQueries, setGoldenQueries] = useState([]);
  const [goldenFilter, setGoldenFilter] = useState('');

  const effectiveTotal = limit ? Math.min(parseInt(limit, 10) || 0, files.length) : files.length;

  const onPickNotes = (e) => {
    const picked = filterMarkdownFiles(e.target.files ?? []);
    setFiles(picked);
    setProgress({ done: 0, total: 0, current: '' });
    setErrors([]);
    setFinished(false);
    abortRef.current = false;
  };

  const onPickGolden = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const data = JSON.parse(await file.text());
      setGoldenQueries(Array.isArray(data) ? data : []);
    } catch {
      setGoldenQueries([]);
      alert('Could not parse golden-set.json');
    }
    e.target.value = '';
  };

  const stopImport = () => {
    abortRef.current = true;
  };

  const runImport = async () => {
    if (!vaultKey || files.length === 0 || running) return;

    const batch = limit ? files.slice(0, parseInt(limit, 10) || files.length) : files;
    abortRef.current = false;
    setRunning(true);
    setFinished(false);
    setErrors([]);
    setProgress({ done: 0, total: batch.length, current: '' });

    const failLog = [];

    for (let i = 0; i < batch.length; i++) {
      if (abortRef.current) break;

      const file = batch[i];
      const label = file.webkitRelativePath || file.name;
      setProgress({ done: i, total: batch.length, current: label });

      try {
        const raw = await file.text();
        const { body, sourceUrl } = prepareEvergreenNote(file.name, raw);
        await saveCapture(vaultKey, {
          body,
          project: IMPORT_PROJECT,
          sourceUrl,
        });
      } catch (err) {
        failLog.push({ file: label, message: err.message || String(err) });
        setErrors([...failLog]);
      }

      setProgress({ done: i + 1, total: batch.length, current: label });
    }

    setRunning(false);
    setFinished(!abortRef.current);
  };

  const filteredGolden = goldenQueries.filter((q) => {
    if (!goldenFilter) return true;
    const f = goldenFilter.toLowerCase();
    return (
      q.query?.toLowerCase().includes(f)
      || q.category?.toLowerCase().includes(f)
      || q.id?.toLowerCase().includes(f)
    );
  });

  const pct = progress.total ? Math.round((progress.done / progress.total) * 100) : 0;

  return (
    <div className="dev-import-page">
      <header className="dev-import-header">
        <div>
          <p className="dev-import-badge">Dev testing only — not for production</p>
          <h1>Evergreen notes import</h1>
          <p className="dev-import-lead">
            Bulk-load the Andy Matuschak evergreen fixture through the real save pipeline
            (client encrypt + embed → server). Project: <strong>{IMPORT_PROJECT}</strong>.
          </p>
        </div>
        <Link to="/notes" className="btn btn-primary">← Back to notes</Link>
      </header>

      <section className="glass-card dev-import-section">
        <h2>1. Get the fixture</h2>
        <p>
          The repo ships <code>golden-set.json</code> only — markdown notes are <strong>not</strong> in git.
          Generate them by crawling Andy Matuschak&apos;s public site (from the repo root):
        </p>
        <pre className="dev-import-cmd">{`cd obsidian-hybrid-search
npm install
npm run eval:prepare-evergreen-notes -- --force --no-images
# creates: fixtures/evergreen-notes/dataset/notes/*.md (~1,357 files, ~10–20 min)`}</pre>
        <p>
          Then on this page click <strong>Choose notes folder</strong> and select{' '}
          <code>fixtures/evergreen-notes/dataset/notes</code>.
        </p>
      </section>

      <section className="glass-card dev-import-section">
        <h2>2. Import notes</h2>
        <div className="dev-import-controls">
          <label className="btn btn-primary dev-import-file-btn">
            Choose notes folder
            <input
              type="file"
              multiple
              webkitdirectory=""
              directory=""
              onChange={onPickNotes}
              disabled={running}
            />
          </label>
          {files.length > 0 && (
            <span className="dev-import-meta">{files.length} .md files selected</span>
          )}
          <label className="dev-import-limit">
            Limit (blank = all):
            <input
              type="number"
              min="1"
              max={files.length || 1357}
              placeholder="1357"
              value={limit}
              onChange={(e) => setLimit(e.target.value)}
              disabled={running}
            />
          </label>
        </div>

        {files.length > 0 && (
          <p className="dev-import-hint">
            Will import {effectiveTotal} note{effectiveTotal !== 1 ? 's' : ''}.
            Full vault takes ~30–45 min in the browser (one embed per note). Keep this tab open.
          </p>
        )}

        {files.length === 0 && (
          <p className="dev-import-hint">
            Start import unlocks after you pick a folder containing .md files. Golden-set.json alone is not enough.
          </p>
        )}

        <div className="dev-import-actions">
          <button
            type="button"
            className="btn btn-primary"
            onClick={runImport}
            disabled={running || !vaultKey || files.length === 0}
          >
            {running ? 'Importing…' : `Start import (${effectiveTotal || '…'})`}
          </button>
          {running && (
            <button type="button" className="btn" onClick={stopImport}>
              Stop
            </button>
          )}
        </div>

        {(running || progress.done > 0) && (
          <div className="dev-import-progress">
            <div className="dev-import-progress-bar" style={{ width: `${pct}%` }} />
            <p>
              {progress.done} / {progress.total} ({pct}%)
              {progress.current && <> — <code>{progress.current}</code></>}
            </p>
          </div>
        )}

        {finished && errors.length === 0 && (
          <p className="dev-import-success">
            Done. Open <Link to="/notes">Notes</Link>, select project {IMPORT_PROJECT}, and search.
          </p>
        )}
        {finished && errors.length > 0 && (
          <p className="dev-import-warn">
            Stopped with {errors.length} error(s). {progress.done - errors.length} notes may have saved.
          </p>
        )}

        {errors.length > 0 && (
          <ul className="dev-import-errors">
            {errors.slice(-10).map((e) => (
              <li key={e.file}><code>{e.file}</code>: {e.message}</li>
            ))}
            {errors.length > 10 && <li>…and {errors.length - 10} more</li>}
          </ul>
        )}
      </section>

      <section className="glass-card dev-import-section">
        <h2>3. Golden-set queries (optional)</h2>
        <p>
          Load <code>golden-set.json</code> from the same fixture. Copy queries into the Notes search bar
          and check whether expected notes appear.
        </p>
        <label className="btn dev-import-file-btn">
          Load golden-set.json
          <input type="file" accept=".json,application/json" onChange={onPickGolden} />
        </label>
        {goldenQueries.length > 0 && (
          <>
            <input
              className="dev-import-filter"
              placeholder="Filter by query, category, or id…"
              value={goldenFilter}
              onChange={(e) => setGoldenFilter(e.target.value)}
            />
            <p className="dev-import-meta">{filteredGolden.length} of {goldenQueries.length} queries</p>
            <div className="dev-import-golden-list">
              {filteredGolden.map((item) => (
                <div key={item.id} className="dev-import-golden-item">
                  <div className="dev-import-golden-top">
                    <span className="dev-import-golden-id">{item.id}</span>
                    <span className="dev-import-golden-cat">{item.category}</span>
                    <button
                      type="button"
                      className="btn btn-sm"
                      onClick={() => navigator.clipboard.writeText(item.query)}
                    >
                      Copy query
                    </button>
                  </div>
                  <p className="dev-import-golden-query">{item.query}</p>
                  <p className="dev-import-golden-paths">
                    <strong>Expect:</strong>{' '}
                    {(item.relevant_paths ?? []).map((p) => p.replace(/^notes\//, '').replace(/\.md$/, '')).join(', ')}
                  </p>
                  {item.notes && <p className="dev-import-golden-notes">{item.notes}</p>}
                </div>
              ))}
            </div>
          </>
        )}
      </section>
    </div>
  );
}

export default function DevImportPage() {
  return (
    <VaultGate>
      <DevImportInner />
    </VaultGate>
  );
}
