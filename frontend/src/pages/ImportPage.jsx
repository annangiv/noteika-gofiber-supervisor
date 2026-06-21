import { useEffect, useRef, useState } from 'react';
import { Link, Navigate } from 'react-router-dom';
import VaultGate from '../components/VaultGate';
import { useVault } from '../context/VaultContext';
import { importCapture } from '../lib/saveCapture';
import { filterMarkdownFiles, prepareMarkdownNote } from '../lib/markdownImport';
import '../notes.css';
import '../import.css';

const MAX_FILES = 100;
const DEFAULT_PROJECT = 'Imports';

function ImportInner({ user }) {
  const { vaultKey } = useVault();
  const abortRef = useRef(false);

  const [files, setFiles] = useState([]);
  const [project, setProject] = useState(DEFAULT_PROJECT);
  const [running, setRunning] = useState(false);
  const [progress, setProgress] = useState({ done: 0, total: 0, current: '' });
  const [errors, setErrors] = useState([]);
  const [finished, setFinished] = useState(false);

  const onPickFiles = (e) => {
    const picked = filterMarkdownFiles(e.target.files ?? []);
    if (picked.length > MAX_FILES) {
      alert(`Pick up to ${MAX_FILES} .md files at a time (you selected ${picked.length}).`);
      e.target.value = '';
      return;
    }
    setFiles(picked);
    setProgress({ done: 0, total: 0, current: '' });
    setErrors([]);
    setFinished(false);
    abortRef.current = false;
  };

  const stopImport = () => {
    abortRef.current = true;
  };

  const runImport = async () => {
    if (!vaultKey || files.length === 0 || running) return;

    abortRef.current = false;
    setRunning(true);
    setFinished(false);
    setErrors([]);
    setProgress({ done: 0, total: files.length, current: '' });

    const failLog = [];
    const targetProject = project.trim() || DEFAULT_PROJECT;

    for (let i = 0; i < files.length; i++) {
      if (abortRef.current) break;

      const file = files[i];
      setProgress({ done: i, total: files.length, current: file.name });

      try {
        const raw = await file.text();
        const { body, sourceUrl } = prepareMarkdownNote(file.name, raw);
        await importCapture(vaultKey, { body, project: targetProject, sourceUrl });
      } catch (err) {
        failLog.push({ file: file.name, message: err.message || String(err) });
        setErrors([...failLog]);
      }

      setProgress({ done: i + 1, total: files.length, current: file.name });
    }

    setRunning(false);
    setFinished(!abortRef.current);
  };

  const pct = progress.total ? Math.round((progress.done / progress.total) * 100) : 0;

  return (
    <div className="import-page">
      <header className="import-header">
        <div>
          <h1>Import notes</h1>
          <p className="import-lead">
            Upload up to {MAX_FILES} markdown (.md) files. Each one goes through the same
            client-side encryption and embedding pipeline as a normal save — Noteika never
            sees the plaintext.
          </p>
        </div>
        <Link to="/notes" className="btn btn-primary">← Back to notes</Link>
      </header>

      <section className="glass-card import-section">
        <div className="import-controls">
          <label className="btn btn-primary import-file-btn">
            Choose .md files
            <input type="file" multiple accept=".md" onChange={onPickFiles} disabled={running} />
          </label>
          {files.length > 0 && (
            <span className="import-meta">{files.length} .md file{files.length !== 1 ? 's' : ''} selected</span>
          )}
          <label className="import-project">
            Project:
            <input
              type="text"
              value={project}
              onChange={(e) => setProject(e.target.value)}
              placeholder={DEFAULT_PROJECT}
              disabled={running}
            />
          </label>
        </div>

        {files.length > 0 && (
          <p className="import-hint">
            Will import {files.length} note{files.length !== 1 ? 's' : ''} into "{project.trim() || DEFAULT_PROJECT}".
            One embed per note — keep this tab open until done.
          </p>
        )}

        {files.length === 0 && (
          <p className="import-hint">Start import unlocks after you pick at least one .md file.</p>
        )}

        <div className="import-actions">
          <button
            type="button"
            className="btn btn-primary"
            onClick={runImport}
            disabled={running || !vaultKey || files.length === 0}
          >
            {running ? 'Importing…' : `Start import (${files.length || '…'})`}
          </button>
          {running && (
            <button type="button" className="btn" onClick={stopImport}>
              Stop
            </button>
          )}
        </div>

        {(running || progress.done > 0) && (
          <div className="import-progress">
            <div className="import-progress-bar" style={{ width: `${pct}%` }} />
            <p>
              {progress.done} / {progress.total} ({pct}%)
              {progress.current && <> — <code>{progress.current}</code></>}
            </p>
          </div>
        )}

        {finished && errors.length === 0 && (
          <p className="import-success">
            Done. Open <Link to="/notes">Notes</Link> and select project "{project.trim() || DEFAULT_PROJECT}".
          </p>
        )}
        {finished && errors.length > 0 && (
          <p className="import-warn">
            Stopped with {errors.length} error(s). {progress.done - errors.length} notes may have saved.
          </p>
        )}

        {errors.length > 0 && (
          <ul className="import-errors">
            {errors.slice(-10).map((e) => (
              <li key={e.file}><code>{e.file}</code>: {e.message}</li>
            ))}
            {errors.length > 10 && <li>…and {errors.length - 10} more</li>}
          </ul>
        )}
      </section>
    </div>
  );
}

export default function ImportPage() {
  const [user, setUser] = useState(null);
  const [authLoading, setAuthLoading] = useState(true);

  useEffect(() => {
    async function checkAuth() {
      try {
        const res = await fetch('/api/auth/me');
        setUser(res.ok ? await res.json() : null);
      } catch (err) {
        setUser(null);
        console.error('Auth verification failed:', err);
      } finally {
        setAuthLoading(false);
      }
    }
    checkAuth();
  }, []);

  if (authLoading) return null;
  if (user && !user.pro_access) {
    return <Navigate to="/account" replace />;
  }

  return (
    <VaultGate>
      <ImportInner user={user} />
    </VaultGate>
  );
}
