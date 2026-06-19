import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

export default function AccountPage() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleteInput, setDeleteInput] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    fetch('/api/auth/me')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then(setUser)
      .catch(() => navigate('/login'))
      .finally(() => setLoading(false));
  }, [navigate]);

  const handleExport = async () => {
    setExporting(true);
    try {
      const res = await fetch('/api/account/export');
      if (!res.ok) throw new Error('export failed');
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'noteika-export.json';
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      alert('Export failed. Please try again.');
    } finally {
      setExporting(false);
    }
  };

  const handleDelete = async () => {
    if (deleteInput !== 'DELETE') return;
    setDeleting(true);
    try {
      const res = await fetch('/api/account', { method: 'DELETE' });
      if (!res.ok) throw new Error('delete failed');
      navigate('/');
    } catch {
      alert('Could not delete account. Please try again.');
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="page-loader">
        <div className="page-loader-inner"><div className="spinner" /></div>
      </div>
    );
  }

  return (
    <div className="site">
      <header className="site-header">
        <div className="container site-header-inner">
          <Link to="/" className="brand">
            <span className="brand-mark">n</span>
            <span className="brand-name">noteika</span>
          </Link>
          <nav className="site-nav">
            <Link to="/notes" className="nav-link">← Back to notes</Link>
          </nav>
        </div>
      </header>

      <main className="section">
        <div className="container narrow">
          <p className="eyebrow">Account</p>
          <h1>Your account</h1>

          <div className="account-card">
            <h2>Profile</h2>
            <dl className="account-dl">
              <dt>Name</dt>
              <dd>{user?.full_name || '—'}</dd>
              <dt>Email</dt>
              <dd>{user?.email || '—'}</dd>
              <dt>Plan</dt>
              <dd className="capitalize">{user?.tier || 'free'}</dd>
              <dt>Member since</dt>
              <dd>{user?.created_at ? new Date(user.created_at * 1000).toLocaleDateString() : '—'}</dd>
            </dl>
          </div>

          <div className="account-card">
            <h2>Data</h2>
            <p className="account-desc">Download all your captures as a JSON file.</p>
            <button type="button" className="btn btn-secondary" onClick={handleExport} disabled={exporting}>
              {exporting ? 'Preparing export…' : 'Download my data'}
            </button>
          </div>

          <div className="account-card">
            <h2>Session</h2>
            <p className="account-desc">Sign out on this device.</p>
            <a href="/auth/logout" className="btn btn-secondary">Log out</a>
          </div>

          <div className="account-card account-danger">
            <h2>Delete account</h2>
            <p className="account-desc">
              Permanently delete your account and all captures. This cannot be undone.
            </p>
            {!confirmDelete ? (
              <button type="button" className="btn btn-danger" onClick={() => setConfirmDelete(true)}>
                Delete my account
              </button>
            ) : (
              <div className="delete-confirm">
                <p>Type <strong>DELETE</strong> to confirm:</p>
                <input
                  type="text"
                  value={deleteInput}
                  onChange={(e) => setDeleteInput(e.target.value)}
                  placeholder="DELETE"
                  className="input"
                />
                <div className="delete-actions">
                  <button type="button" className="btn btn-secondary" onClick={() => { setConfirmDelete(false); setDeleteInput(''); }}>
                    Cancel
                  </button>
                  <button
                    type="button"
                    className="btn btn-danger"
                    disabled={deleteInput !== 'DELETE' || deleting}
                    onClick={handleDelete}
                  >
                    {deleting ? 'Deleting…' : 'Permanently delete'}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
