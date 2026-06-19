import { Link } from 'react-router-dom';
import MarketingLayout from '../components/MarketingLayout';

const plans = [
  {
    name: 'Free',
    price: '$0',
    period: 'forever',
    description: 'Everything you need to stop losing prompts and notes.',
    features: [
      'Unlimited captures',
      'Semantic search',
      'Project folders',
      'Duplicate resurfacing',
      'JSON export',
    ],
    cta: 'Get started',
    highlighted: true,
  },
  {
    name: 'Pro',
    price: '$8',
    period: '/ month',
    description: 'For power users who live across many projects.',
    features: [
      'Everything in Free',
      'Priority embedding queue',
      'Advanced export formats',
      'Email support',
      'Early access features',
    ],
    cta: 'Coming soon',
    highlighted: false,
    disabled: true,
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
            Start free. Upgrade when Noteika becomes part of your daily workflow.
          </p>
        </div>
      </section>

      <section className="section">
        <div className="container pricing-grid">
          {plans.map((plan) => (
            <article key={plan.name} className={`pricing-card ${plan.highlighted ? 'highlighted' : ''}`}>
              {plan.highlighted && <span className="pricing-badge">Most popular</span>}
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
                <Link to="/login" className="btn btn-primary btn-block">{plan.cta}</Link>
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
            <p>Yes. Your captures are stored in your own instance. Export or delete your account anytime.</p>
          </div>
          <div className="faq-item">
            <h3>Do I need to organize everything upfront?</h3>
            <p>No. Paste first. Projects are optional. Noteika finds notes by meaning later.</p>
          </div>
        </div>
      </section>
    </MarketingLayout>
  );
}
