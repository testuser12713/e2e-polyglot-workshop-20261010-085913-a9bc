import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { ApiError, apiGet } from '../api/client';
import { ErrorBanner } from '../components/ErrorBanner';
import { SelectField } from '../components/Field';
import { StatusBadge } from '../components/StatusBadge';
import { ORDER_STATUSES, STATUS_LABELS, formatMoney } from '../lib/format';
import './OrdersPage.css';

export interface OrderRow {
  order_number: string;
  status: string;
  plate?: string | null;
  preferred_date?: string | null;
  customer_name?: string | null;
  total_cents?: number | null;
}

interface OrdersResponse {
  orders?: OrderRow[] | null;
}

const SEARCH_DEBOUNCE_MS = 250;

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

function rowSuffix(orderNumber: string): string {
  const digits = orderNumber.replace(/\D/g, '');
  return digits.slice(-4) || orderNumber;
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  return 'Es ist ein Fehler aufgetreten.';
}

export function OrdersPage() {
  const [status, setStatus] = useState('');
  const [plateInput, setPlateInput] = useState('');
  const [plate, setPlate] = useState('');
  const [orders, setOrders] = useState<OrderRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const requestId = useRef(0);

  useEffect(() => {
    const handle = window.setTimeout(() => {
      setPlate(plateInput.trim());
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(handle);
  }, [plateInput]);

  useEffect(() => {
    const id = ++requestId.current;
    setLoading(true);
    apiGet<OrdersResponse>('/api/workshop/orders', { status, plate })
      .then((response) => {
        if (id !== requestId.current) return;
        setOrders(response?.orders ?? []);
        setError(null);
      })
      .catch((cause) => {
        if (id !== requestId.current) return;
        setOrders([]);
        setError(errorMessage(cause));
      })
      .finally(() => {
        if (id !== requestId.current) return;
        setLoading(false);
      });
  }, [status, plate]);

  const hasFilter = status !== '' || plate !== '';
  const count = orders.length;
  const emptyTitle = hasFilter ? 'Keine Treffer für diesen Filter' : 'Noch keine Aufträge';
  const emptyBody = hasFilter
    ? 'Ändern Sie den Statusfilter oder die Kennzeichensuche.'
    : 'Sobald Kundenanfragen eingehen, erscheinen die Aufträge hier.';

  return (
    <section className="orders-page" data-testid="orders-page">
      <header className="orders-page__head row-between">
        <div className="orders-page__titles">
          <h1 className="page__title">Aufträge</h1>
          <p className="orders-page__count">
            <span className="numeric" data-testid="orders-count">
              {count}
            </span>{' '}
            {count === 1 ? 'Auftrag' : 'Aufträge'} · Treffer gesamt
          </p>
        </div>
      </header>

      <section className="card" data-od-id="orders-toolbar-card">
        <div className="toolbar">
          <div className="toolbar__filter">
            <SelectField
              label="Status"
              id="filter-status"
              data-od-id="filter-status"
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <option value="">Alle Status</option>
              {ORDER_STATUSES.map((value) => (
                <option key={value} value={value}>
                  {STATUS_LABELS[value]}
                </option>
              ))}
            </SelectField>
          </div>

          <div className="toolbar__search field">
            <label className="field__label" htmlFor="search-plate">
              Suche
            </label>
            <div className="search field__input-wrap">
              <svg
                className="search__lead"
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                aria-hidden="true"
              >
                <circle cx="11" cy="11" r="7" stroke="currentColor" strokeWidth="1.6" />
                <path
                  d="M16 16l4 4"
                  stroke="currentColor"
                  strokeWidth="1.6"
                  strokeLinecap="round"
                />
              </svg>
              <input
                className="field__control search__input"
                id="search-plate"
                data-od-id="search-plate"
                type="text"
                placeholder="Kennzeichen suchen"
                autoComplete="off"
                value={plateInput}
                onChange={(event) => setPlateInput(event.target.value)}
              />
              {plateInput ? (
                <button
                  type="button"
                  className="search__clear"
                  aria-label="Suche zurücksetzen"
                  onClick={() => setPlateInput('')}
                >
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path
                      d="M6 6l12 12M18 6L6 18"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                    />
                  </svg>
                </button>
              ) : null}
            </div>
          </div>
        </div>
      </section>

      <section className="card orders-page__list-card" data-od-id="orders-list-card">
        {error ? <ErrorBanner message={error} /> : null}

        {loading ? (
          <div className="orders-page__skeleton" aria-hidden="true">
            <div className="skeleton-line" />
            <div className="skeleton-line" />
            <div className="skeleton-line" />
          </div>
        ) : error ? null : count === 0 ? (
          <div className="empty" data-od-id="orders-empty" data-testid="orders-empty">
            <svg
              className="empty__icon"
              width="32"
              height="32"
              viewBox="0 0 24 24"
              fill="none"
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="7" stroke="currentColor" strokeWidth="1.6" />
              <path d="M16 16l4 4" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
            </svg>
            <p className="empty__title">{emptyTitle}</p>
            <p className="empty__body">{emptyBody}</p>
          </div>
        ) : (
          <ul className="order-list" data-testid="orders-list">
            {orders.map((order) => (
              <li
                className="order-row"
                data-od-id={`order-row-${rowSuffix(order.order_number)}`}
                data-status={order.status}
                key={order.order_number}
              >
                <Link className="order-row__link" to={`/werkstatt/auftraege/${order.order_number}`}>
                  <span className="order-row__num mono">{order.order_number}</span>
                  <span className="order-row__plate" data-cell="plate">
                    {(order.plate ?? '—').toUpperCase()}
                  </span>
                  <span className="order-row__customer">{order.customer_name ?? '—'}</span>
                  <span className="order-row__date numeric">
                    {formatDate(order.preferred_date)}
                  </span>
                  <span className="order-row__status">
                    <StatusBadge status={order.status} />
                  </span>
                  <span className="order-row__total numeric">
                    {formatMoney(order.total_cents ?? 0)}
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
        )}
      </section>
    </section>
  );
}
