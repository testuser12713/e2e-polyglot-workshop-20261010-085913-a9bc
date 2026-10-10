import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { ApiError, apiGet } from '../api/client';
import { Button } from '../components/Button';
import { ErrorBanner } from '../components/ErrorBanner';
import { StatusBadge } from '../components/StatusBadge';
import { formatMoney } from '../lib/format';
import './DashboardPage.css';

export interface DashboardOrder {
  order_number: string;
  status: string;
  plate?: string | null;
  preferred_date?: string | null;
  customer_name?: string | null;
  total_cents?: number | null;
}

export interface DashboardResponse {
  open: DashboardOrder[];
  finished_today: DashboardOrder[];
  revenue_month_cents: number;
}

const DASHBOARD_PATH = '/api/workshop/dashboard';
const ORDERS_PATH = '/werkstatt/auftraege';

const OPEN_STATUSES = ['angefragt', 'bestaetigt', 'in_arbeit'];

const OPEN_CAPTION = 'angefragt, bestätigt, in Arbeit';
const DONE_CAPTION = 'nach Zeitpunkt des Wechsels nach „fertig“';
const REVENUE_CAPTION = 'Gesamtbeträge fertiger Rechnungen (ganze Cent)';

function parseDate(value?: string | null): Date | null {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

function formatDate(value?: string | null): string {
  const date = parseDate(value);
  if (!date) return '—';
  return date.toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    timeZone: 'Europe/Berlin',
  });
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  return 'Es ist ein Fehler aufgetreten.';
}

function normalize(response: DashboardResponse): DashboardResponse {
  return {
    open: Array.isArray(response?.open) ? response.open : [],
    finished_today: Array.isArray(response?.finished_today) ? response.finished_today : [],
    revenue_month_cents:
      typeof response?.revenue_month_cents === 'number' ? response.revenue_month_cents : 0,
  };
}

function DashboardStatCard({
  testId,
  valueTestId,
  label,
  caption,
  value,
  money = false,
}: {
  testId: string;
  valueTestId: string;
  label: string;
  caption: string;
  value: string;
  money?: boolean;
}) {
  return (
    <div className="card stat-card" data-testid={testId}>
      <span className="stat-card__label">{label}</span>
      <span
        className={money ? 'stat-card__value money tnum' : 'stat-card__value tnum'}
        data-testid={valueTestId}
      >
        {value}
      </span>
      <span className="stat-card__caption">{caption}</span>
    </div>
  );
}

function EmptyState({ title, body }: { title: string; body: string }) {
  return (
    <div className="dashboard__empty">
      <p className="dashboard__empty-title">{title}</p>
      <p className="dashboard__empty-body">{body}</p>
    </div>
  );
}

function OrderRows({ orders }: { orders: DashboardOrder[] }) {
  return (
    <ul className="order-list">
      {orders.map((order) => (
        <li
          className="order-row"
          key={order.order_number}
          data-testid={`dash-order-${order.order_number}`}
        >
          <Link
            className="order-row__link"
            to={`${ORDERS_PATH}/${encodeURIComponent(order.order_number)}`}
          >
            <span className="order-row__num tnum">{order.order_number}</span>
            <span className="order-row__plate tnum">
              {(order.plate ?? '—').toUpperCase()}
            </span>
            <span className="order-row__customer">{order.customer_name ?? '—'}</span>
            <span className="order-row__date tnum">{formatDate(order.preferred_date)}</span>
            <span className="order-row__status">
              <StatusBadge status={order.status} />
            </span>
            <svg
              className="order-row__chev"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              aria-hidden="true"
            >
              <path
                d="M9 6l6 6-6 6"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </Link>
        </li>
      ))}
    </ul>
  );
}

export function DashboardPage() {
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadDashboard = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const response = await apiGet<DashboardResponse>(DASHBOARD_PATH);
      setData(normalize(response));
      setError(null);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      if (!silent) setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadDashboard();
  }, [loadDashboard]);

  async function refresh() {
    setRefreshing(true);
    try {
      await loadDashboard(true);
    } finally {
      setRefreshing(false);
    }
  }

  const openOrders = (data?.open ?? []).filter((order) => OPEN_STATUSES.includes(order.status));
  const doneToday = data?.finished_today ?? [];
  const revenueCents = data?.revenue_month_cents ?? 0;

  if (loading && !data) {
    return (
      <section className="dashboard" data-testid="dashboard">
        <h1 className="page__title">Dashboard</h1>
        <div className="skeleton-line" />
        <div className="skeleton-line" />
        <div className="skeleton-line" />
      </section>
    );
  }

  return (
    <section className="dashboard" data-testid="dashboard">
      <header className="row-between dashboard__head">
        <div>
          <h1 className="page__title">Dashboard</h1>
          <p className="dashboard__subtitle">Überblick über den laufenden Werkstattbetrieb.</p>
        </div>
        <div className="dashboard__actions">
          <Button variant="secondary" onClick={refresh} loading={refreshing}>
            Aktualisieren
          </Button>
          <Link className="btn btn--secondary" to={ORDERS_PATH}>
            Alle Aufträge
          </Link>
        </div>
      </header>

      {error ? (
        <ErrorBanner message={error} />
      ) : null}

      <section className="grid-3" aria-label="Kennzahlen" data-testid="dashboard-stats">
        <DashboardStatCard
          testId="stat-open"
          valueTestId="stat-open-value"
          label="Offene Aufträge"
          caption={OPEN_CAPTION}
          value={String(openOrders.length)}
        />
        <DashboardStatCard
          testId="stat-done-today"
          valueTestId="stat-done-value"
          label="Heute fertiggestellt"
          caption={DONE_CAPTION}
          value={String(doneToday.length)}
        />
        <DashboardStatCard
          testId="stat-revenue"
          valueTestId="stat-revenue-value"
          label="Umsatz laufender Monat"
          caption={REVENUE_CAPTION}
          value={formatMoney(revenueCents)}
          money
        />
      </section>

      <div className="grid-cards">
        <section className="card" data-testid="dashboard-open-orders">
          <div className="card__header">
            <h2 className="card__title">Offene Aufträge</h2>
            <Link className="btn btn--ghost" to={ORDERS_PATH}>
              Alle anzeigen
            </Link>
          </div>
          {openOrders.length === 0 ? (
            <EmptyState
              title="Keine offenen Aufträge"
              body="Sobald ein Auftrag eingeht, erscheint er hier."
            />
          ) : (
            <OrderRows orders={openOrders} />
          )}
        </section>

        <section className="card" data-testid="dashboard-done-today">
          <div className="card__header">
            <h2 className="card__title">Heute fertiggestellt</h2>
          </div>
          {doneToday.length === 0 ? (
            <EmptyState
              title="Heute noch nichts fertiggestellt"
              body="Fertig gemeldete Aufträge des heutigen Tages erscheinen hier."
            />
          ) : (
            <OrderRows orders={doneToday} />
          )}
        </section>
      </div>
    </section>
  );
}
