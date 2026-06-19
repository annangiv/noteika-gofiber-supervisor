import { Link } from 'react-router-dom';
import MarketingLayout from '../components/MarketingLayout';

const plans = [
  {
    name: 'Free',
    price: '$0',
    period: 'forever',
    description: 'Try Noteika — encrypted notes with semantic search.',
    features: [
      '10 encrypted captures',
      'Semantic + exact search',
      'Project folders',
      'Duplicate warnings',
      'JSON export',
    ],
    cta: 'Get started',
    href: '/login',
    highlighted: false,
    disabled: false,
  },
  {
    name: 'Pro',
    price: '$8',
    period: '/ month',
    description: 'Unlimited saves for daily use across projects.',
    features: [
      'Unlimited encrypted captures',
      'Semantic + exact search',
      'Everything in Free',
      'Priority support',
      'Cancel anytime',
    ],
    cta: 'Sign in to upgrade',
    href: '/login',
    highlighted: true,
    disabled: false,
  },
];

export default function PricingPage() {
  return (
    <MarketingLayout>
      <section className="page-hero">
        <div className="container narrow">
          <p className="eyebrow">Pricing</p>
          <h1>Simple, honest pricing</h1>
          <p className="page-lead">
            Start free with 10 notes. Upgrade when Noteika becomes part of your daily workflow.
          </p>
        </div>
      </section>

      <section className="section">
        <div className="container pricing-grid">
          {plans.map((plan) => (
            <article key={plan.name} className={`pricing-card ${plan.highlighted ? 'highlighted' : ''}`}>
              {plan.highlighted && <span className="pricing-badge">Unlimited saves</span>}
              <h2>{plan.name}</h2>
              <div className="pricing-price">
                <span className="price">{plan.price}</span>
                <span className="period">{plan.period}</span>
              </div>
              <p className="pricing-desc">{plan.description}</p>
              <ul className="pricing-features">
                {plan.features.map((f) => (
                  <li key={f}>✓ {f}</li>
                ))}
              </ul>
              {plan.disabled ? (
                <button className="btn btn-secondary btn-block" disabled>{plan.cta}</button>
              ) : (
                <Link to={plan.href} className={`btn ${plan.highlighted ? 'btn-primary' : 'btn-secondary'} btn-block`}>
                  {plan.cta}
                </Link>
              )}
            </article>
          ))}
        </div>
      </section>

      <section className="section section-muted">
        <div className="container narrow faq">
          <h2>FAQ</h2>
          <div className="faq-item">
            <h3>Is my data private?</h3>
            <p>Your note text is encrypted on your device before it reaches our servers. We store embedding vectors for search, encrypted at rest.</p>
          </div>
          <div className="faq-item">
            <h3>What counts toward the free limit?</h3>
            <p>Saved captures (not trash). Search works on all your notes. Deleted notes in Trash don&apos;t count toward the active limit.</p>
          </div>
        </div>
      </section>
    </MarketingLayout>
  );
}
