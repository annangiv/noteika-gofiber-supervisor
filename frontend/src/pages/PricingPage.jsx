import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
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
    highlighted: false,
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
    highlighted: true,
  },
];

export default function PricingPage() {
  const [user, setUser] = useState(null);
  const [authLoading, setAuthLoading] = useState(true);
  const [billingLoading, setBillingLoading] = useState(false);
  const [billingNotice, setBillingNotice] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    async function checkAuth() {
      try {
        const res = await fetch('/api/auth/me');
        if (res.ok) {
          const data = await res.json();
          setUser(data);
        }
      } catch (err) {
        console.warn('Failed to check auth status:', err);
      } finally {
        setAuthLoading(false);
      }
    }
    checkAuth();
  }, []);

  const handleFreeAction = () => {
    if (user) {
      navigate('/notes');
    } else {
      navigate('/login');
    }
  };

  const handleProAction = async () => {
    if (!user) {
      navigate('/login');
      return;
    }

    if (user.pro_access) {
      navigate('/account');
      return;
    }

    if (!user.stripe_enabled) {
      setBillingNotice('Billing is not configured on this server yet.');
      return;
    }

    setBillingLoading(true);
    setBillingNotice('');
    try {
      const res = await fetch('/api/billing/checkout', { method: 'POST' });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'checkout failed');
      window.location.href = data.url;
    } catch (err) {
      setBillingNotice(err.message || 'Could not start checkout.');
      setBillingLoading(false);
    }
  };

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
        <div className="container pricing-grid" style={{ position: 'relative' }}>
          {billingNotice && (
            <p className="settings-saved" style={{
              position: 'absolute',
              top: '-48px',
              left: '50%',
              transform: 'translateX(-50%)',
              width: '100%',
              maxWidth: '400px',
              textAlign: 'center',
              color: 'var(--danger)',
              background: 'rgba(239, 68, 68, 0.1)',
              border: '1px solid rgba(239, 68, 68, 0.2)',
              padding: '8px 16px',
              borderRadius: '6px',
              zIndex: 10
            }}>
              {billingNotice}
            </p>
          )}

          {plans.map((plan) => {
            const isFreePlan = plan.name === 'Free';
            let buttonText = plan.cta;
            let onClickAction = isFreePlan ? handleFreeAction : handleProAction;
            let isDisabled = false;

            if (!authLoading && user) {
              if (isFreePlan) {
                buttonText = 'Go to notes';
              } else {
                if (user.pro_access) {
                  buttonText = 'Manage plan';
                } else if (!user.stripe_enabled) {
                  buttonText = 'Billing not configured';
                  isDisabled = true;
                } else {
                  buttonText = billingLoading ? 'Redirecting…' : 'Upgrade to Pro';
                }
              }
            }

            return (
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
                <button
                  type="button"
                  className={`btn ${plan.highlighted ? 'btn-primary' : 'btn-secondary'} btn-block`}
                  onClick={onClickAction}
                  disabled={isDisabled || (plan.name === 'Pro' && billingLoading)}
                >
                  {buttonText}
                </button>
              </article>
            );
          })}
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
