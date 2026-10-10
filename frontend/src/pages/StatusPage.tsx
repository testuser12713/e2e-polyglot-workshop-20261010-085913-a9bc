import { useState } from 'react';
import type { FormEvent } from 'react';
import { ApiError, apiGet } from '../api/client';
import { Button } from '../components/Button';
import { ErrorBanner } from '../components/ErrorBanner';
import { Field } from '../components/Field';
import { StatusBadge } from '../components/StatusBadge';

interface StatusHistoryEntry {
  status: string;
  changed_at: string;
}

interface OrderStatusResponse {
  order_number: string;
  status: string;
  status_history: StatusHistoryEntry[];
}

type Result = { kind: 'found'; data: OrderStatusResponse } | { kind: 'empty' };

const NEUTRAL_MESSAGE =
  'Status nicht gefunden — bitte Auftragsnummer und Kennzeichen prüfen';

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const formatted = new Intl.DateTimeFormat('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
  return `${formatted} Uhr`;
}

function StatusTimeline({ history }: { history: StatusHistoryEntry[] }) {
  const entries = [...history].sort(
    (a, b) => new Date(a.changed_at).getTime() - new Date(b.changed_at).getTime(),
  );
  return (
    <ol className="timeline">
      {entries.map((entry, index) => (
        <li className="timeline__item" key={`${entry.status}-${entry.changed_at}-${index}`}>
          <StatusBadge status={entry.status} />
          <time className="timeline__time" dateTime={entry.changed_at}>
            {formatDateTime(entry.changed_at)}
          </time>
        </li>
      ))}
    </ol>
  );
}

export function StatusPage() {
  const [orderNumber, setOrderNumber] = useState('');
  const [plate, setPlate] = useState('');
  const [touched, setTouched] = useState({ order: false, plate: false });
  const [submitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<Result | null>(null);
  const [error, setError] = useState<string | null>(null);

  const order = orderNumber.trim();
  const plateValue = plate.trim();
  const showOrderError = (touched.order || submitted) && order.length < 3;
  const showPlateError = (touched.plate || submitted) && plateValue.length < 3;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);
    if (order.length < 3 || plateValue.length < 3) return;

    setLoading(true);
    setResult(null);
    setError(null);
    try {
      const data = await apiGet<OrderStatusResponse>(
        `/api/public/orders/${encodeURIComponent(order)}`,
        { plate: plateValue },
      );
      setResult({ kind: 'found', data });
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setResult({ kind: 'empty' });
      } else {
        setError(err instanceof Error ? err.message : 'Es ist ein Fehler aufgetreten.');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="page stack">
      <header data-od-id="page-head">
        <h1 className="page__title">Status abfragen</h1>
        <p className="page__text">
          Geben Sie Ihre Auftragsnummer und das Kennzeichen ein, um den aktuellen Status und den
          Verlauf zu sehen.
        </p>
      </header>

      <form
        className="card form-col"
        id="status-form"
        data-od-id="status-form"
        noValidate
        onSubmit={handleSubmit}
      >
        <Field
          id="order"
          name="order"
          label="Auftragsnummer"
          required
          autoCapitalize="characters"
          className="mono"
          value={orderNumber}
          onChange={(event) => setOrderNumber(event.target.value.toUpperCase())}
          onBlur={() => setTouched((prev) => ({ ...prev, order: true }))}
          helper="z. B. AUF-2025-0187"
          error={showOrderError ? 'Bitte geben Sie die Auftragsnummer an.' : null}
        />
        <Field
          id="plate"
          name="plate"
          label="Kennzeichen"
          required
          autoCapitalize="characters"
          value={plate}
          onChange={(event) => setPlate(event.target.value.toUpperCase())}
          onBlur={() => setTouched((prev) => ({ ...prev, plate: true }))}
          helper="z. B. B-AB 1234"
          error={showPlateError ? 'Bitte geben Sie das Kennzeichen an.' : null}
        />
        <div className="form-actions">
          <Button type="submit" data-od-id="submit-status" loading={loading}>
            Status abfragen
          </Button>
        </div>
      </form>

      <div id="status-result" aria-live="polite" data-od-id="status-result">
        {error ? <ErrorBanner message={error} /> : null}

        {result?.kind === 'found' ? (
          <div className="card" data-od-id="status-found">
            <div className="card__header">
              <div>
                <p className="caption muted">Auftragsnummer</p>
                <p className="h3 mono">{result.data.order_number}</p>
              </div>
              <StatusBadge status={result.data.status} />
            </div>
            <h3 className="h3">Statusverlauf</h3>
            <StatusTimeline history={result.data.status_history} />
          </div>
        ) : null}

        {result?.kind === 'empty' ? (
          <div className="card">
            <div className="empty" data-od-id="status-empty">
              <p className="empty__body">{NEUTRAL_MESSAGE}</p>
            </div>
          </div>
        ) : null}
      </div>
    </section>
  );
}
