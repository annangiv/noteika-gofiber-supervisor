import { Link, useLocation } from 'react-router-dom';
import MarketingLayout from '../components/MarketingLayout';

export default function NotFoundPage() {
  const { pathname } = useLocation();

  return (
    <MarketingLayout>
      <section className="page-hero">
        <div className="container narrow">
          <p className="eyebrow">404</p>
          <h1>Page not found</h1>
          <p className="page-lead">
            There is no route for <code>{pathname}</code>. Maybe the link moved, or the URL has a typo.
          </p>
          <div className="hero-actions" style={{ marginTop: '1.5rem' }}>
            <Link to="/" className="btn btn-primary">Go home</Link>
            <Link to="/notes" className="btn btn-secondary">My notes</Link>
          </div>
        </div>
      </section>
    </MarketingLayout>
  );
}
