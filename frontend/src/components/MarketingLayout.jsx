import { Link, NavLink, useNavigate } from 'react-router-dom';
import { useEffect, useState } from 'react';

export default function MarketingLayout({ children }) {
  const [user, setUser] = useState(null);
  const navigate = useNavigate();

  useEffect(() => {
    fetch('/api/auth/me')
      .then((r) => (r.ok ? r.json() : null))
      .then(setUser)
      .catch(() => setUser(null));
  }, []);

  return (
    <div className="site">
      <header className="site-header">
        <div className="container site-header-inner">
          <Link to="/" className="brand">
            <span className="brand-mark">n</span>
            <span className="brand-name">noteika</span>
          </Link>
          <nav className="site-nav">
            <NavLink to="/pricing" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              Pricing
            </NavLink>
            <NavLink to="/about" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              About
            </NavLink>
            {user ? (
              <>
                <NavLink to="/notes" className="nav-link">Notes</NavLink>
                <button type="button" className="btn btn-primary btn-sm" onClick={() => navigate('/notes')}>
                  Open app
                </button>
              </>
            ) : (
              <Link to="/login" className="btn btn-primary btn-sm">Sign in</Link>
            )}
          </nav>
        </div>
      </header>
      <main>{children}</main>
      <footer className="site-footer">
        <div className="container site-footer-inner">
          <div className="footer-brand">
            <span className="brand-mark">n</span>
            <span>noteika</span>
          </div>
          <p className="footer-tagline">Notes that resurface before you duplicate yourself.</p>
          <div className="footer-links">
            <Link to="/about">About</Link>
            <Link to="/pricing">Pricing</Link>
            <Link to="/login">Sign in</Link>
          </div>
          <p className="footer-copy">© {new Date().getFullYear()} Noteika</p>
        </div>
      </footer>
    </div>
  );
}
