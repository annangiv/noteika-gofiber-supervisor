import { Link } from 'react-router-dom';
import MarketingLayout from '../components/MarketingLayout';

export default function AboutPage() {
  return (
    <MarketingLayout>
      <section className="page-hero">
        <div className="container narrow">
          <p className="eyebrow">About</p>
          <h1>Built for builders, not archivists</h1>
          <p className="page-lead">
            Most note apps are great at saving and terrible at bringing notes back when they matter.
            Noteika does one job: save once, find later — even if you forgot the name.
          </p>
        </div>
      </section>

      <section className="section">
        <div className="container about-grid">
          <article className="about-block">
            <h2>The problem</h2>
            <p>
              Prompts live in AI chats. Links live in browser tabs. Decisions live in scratchpads.
              A week later you <em>know</em> you figured it out — but not where you put it.
            </p>
          </article>
          <article className="about-block">
            <h2>Our approach</h2>
            <p>
              One capture box. Silent type detection for display. Semantic search instead of keyword guessing.
              Proactive resurfacing so you see duplicates before you redo the work.
            </p>
          </article>
          <article className="about-block">
            <h2>What we&apos;re not</h2>
            <p>
              Not a graph view. Not team sharing. Not in-app AI chat.
              Noteika is the place you save from elsewhere and find when it counts.
            </p>
          </article>
        </div>
      </section>

      <section className="section section-muted">
        <div className="container narrow">
          <h2 className="section-title">How it works</h2>
          <ol className="steps-list">
            <li><strong>Paste</strong> anything — prompt, link, Q&amp;A, code snippet.</li>
            <li><strong>Tag a project</strong> optionally (Keller, client work, Inbox…).</li>
            <li><strong>Find by meaning</strong> when you need it — or get nudged if it already exists.</li>
          </ol>
          <Link to="/login" className="btn btn-primary">Try Noteika</Link>
        </div>
      </section>
    </MarketingLayout>
  );
}
