import { useEffect, useState } from 'react';
import { Navigate } from 'react-router-dom';

export default function ProtectedRoute({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function check() {
      try {
        const res = await fetch('/api/auth/me');
        if (res.ok) setUser(await res.json());
      } catch {
        setUser(null);
      } finally {
        setLoading(false);
      }
    }
    check();
  }, []);

  if (loading) {
    return (
      <div className="page-loader">
        <div className="page-loader-inner">
          <div className="spinner" />
          <p>Loading…</p>
        </div>
      </div>
    );
  }

  if (!user) return <Navigate to="/login" replace />;
  return children;
}
