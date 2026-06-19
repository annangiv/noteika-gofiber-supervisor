import { Link } from 'react-router-dom';
import MarketingLayout from '../components/MarketingLayout';

export default function LandingPage() {
  return (
    <MarketingLayout>
      <section className="hero">
        <div className="container hero-grid">
          <div className="hero-copy">
            <p className="eyebrow">For builders juggling many projects</p>
            <h1>Notes that resurface before you duplicate yourself</h1>
            <p className="hero-lead">
              Paste anything once — AI prompts, links, Q&amp;A, scratchpad thoughts.
              Find it later by meaning, even when you forgot what you called it.
            </p>
            <div className="hero-actions">
              <Link to="/login" className="btn btn-primary btn-lg">Get started free</Link>
              <Link to="/about" className="btn btn-secondary btn-lg">How it works</Link>
            </div>
            <p className="hero-note">No type picker. No folder rituals. Just paste and move on.</p>
          </div>
          <div className="hero-visual">
            <div className="demo-card">
              <div className="demo-card-header">
                <span className="demo-dot" />
                <span className="demo-dot" />
                <span className="demo-dot" />
              </div>
              <div className="demo-search">
                <span className="demo-search-icon">⌕</span>
                <span>landing page prompt for AI app builders…</span>
              </div>
              <div className="demo-result demo-result-highlight">
                <div className="demo-result-top">
                  <strong>Emergent vs Lovable landing copy</strong>
                  <span className="demo-badge">92% match</span>
                </div>
                <p>Playful hero variant with social proof section…</p>
                <span className="demo-meta">Noteika · 6 days ago</span>
              </div>
              <div className="demo-result">
                <div className="demo-result-top">
                  <strong>OAuth redirect handling notes</strong>
                  <span className="demo-badge muted">71% match</span>
                </div>
                <p>Callback URL setup for GitHub and Google…</p>
              </div>
              <div className="demo-alert">
                ⚠ Similar capture already in your docket
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="section">
        <div className="container">
          <h2 className="section-title">The graveyard problem</h2>
          <p className="section-lead">
            You don&apos;t lose the thought — you lose where you put it. Noteika brings captures back when they matter.
          </p>
          <div className="feature-grid">
            <article className="feature-card">
              <div className="feature-icon">📋</div>
              <h3>One box, one save</h3>
              <p>Paste from ChatGPT, Claude, or anywhere. Optional project tag. No type picker at save time.</p>
            </article>
            <article className="feature-card">
              <div className="feature-icon">🧠</div>
              <h3>Search by meaning</h3>
              <p>Find &ldquo;that OAuth thing I researched&rdquo; — not exact keywords or filenames you never chose.</p>
            </article>
            <article className="feature-card">
              <div className="feature-icon">💡</div>
              <h3>Resurface before duplicate</h3>
              <p>While you type or search, Noteika shows if you already saved something similar.</p>
            </article>
          </div>
        </div>
      </section>

      <section className="section section-muted">
        <div className="container cta-band">
          <div>
            <h2>Stop re-asking AI the same question</h2>
            <p>Your past work should show up before you start over.</p>
          </div>
          <Link to="/login" className="btn btn-primary btn-lg">Start capturing</Link>
        </div>
      </section>
    </MarketingLayout>
  );
}
