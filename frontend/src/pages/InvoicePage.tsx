import { useState } from 'react';
import type { FormEvent } from 'react';
import { apiGet, ApiError } from '../api/client';
import { Button } from '../components/Button';
import { ErrorBanner } from '../components/ErrorBanner';
import { Field } from '../components/Field';
import { formatMoney } from '../lib/format';

interface InvoiceItem {
  kind: string;
  description: string;
  hours?: number | null;
  quantity?: number | null;
  unit_price_cents?: number | null;
  line_total_cents: number;
}

interface Invoice {
  items: InvoiceItem[];
  subtotal_cents: number;
  tax_cents: number;
  total_cents: number;
  created_at: string;
}

interface InvoiceResponse {
  invoice: Invoice;
}

interface OrderStatusResponse {
  order_number: string;
  status: string;
}

type Result =
  | { kind: 'idle' }
  | { kind: 'invoice'; invoice: Invoice; orderNumber: string; plate: string }
  | { kind: 'notReady' }
  | { kind: 'notFound' }
  | { kind: 'error'; message: string };

const NOT_READY_TITLE = 'Die Rechnung ist ab dem Status fertig verf\u00fcgbar.';
const NOT_READY_BODY = 'Sobald Ihr Auftrag fertiggestellt ist, erscheint die Rechnung hier.';
const NOT_FOUND_TITLE = 'Rechnung nicht gefunden';
const NOT_FOUND_BODY = 'Bitte Auftragsnummer und Kennzeichen pr\u00fcfen.';
const GENERIC_ERROR = 'Es ist ein Fehler aufgetreten.';

function formatDate(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (match) return `${match[3]}.${match[2]}.${match[1]}`;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const day = String(date.getDate()).padStart(2, '0');
  const month = String(date.getMonth() + 1).padStart(2, '0');
  return `${day}.${month}.${date.getFullYear()}`;
}

function formatHours(hours: number): string {
  return `${hours.toFixed(1).replace('.', ',')} h`;
}

function formatQuantity(quantity: number): string {
  return `${String(quantity).replace('.', ',')} Stk.`;
}

function MoneyText({ cents }: { cents: number }) {
  return <span className="money">{formatMoney(cents)}</span>;
}

function PositionRow({ item }: { item: InvoiceItem }) {
  const isLabour = item.kind === 'labour';
  const quantityText = isLabour
    ? typeof item.hours === 'number'
      ? formatHours(item.hours)
      : ''
    : typeof item.quantity === 'number'
      ? formatQuantity(item.quantity)
      : '';

  return (
    <div className="position-row">
      <div className="position-row__desc">
        <span className="position-row__title">{item.description}</span>
        <span className="position-row__meta">
          {isLabour ? 'Arbeitsposition' : 'Teilposition'}
        </span>
      </div>
      <div className="position-row__qty tnum" data-label="Menge/Zeit">
        {quantityText}
      </div>
      <div className="position-row__price tnum" data-label="Einzelpreis">
        {typeof item.unit_price_cents === 'number' ? (
          <MoneyText cents={item.unit_price_cents} />
        ) : (
          '\u2014'
        )}
      </div>
      <div className="position-row__sum tnum" data-label="Summe">
        <MoneyText cents={item.line_total_cents} />
      </div>
    </div>
  );
}

function TotalsBlock({ invoice }: { invoice: Invoice }) {
  return (
    <div className="totals">
      <div className="totals__inner">
        <div className="totals__row">
          <span>Zwischensumme</span>
          <MoneyText cents={invoice.subtotal_cents} />
        </div>
        <div className="totals__row">
          <span>19 % MwSt.</span>
          <MoneyText cents={invoice.tax_cents} />
        </div>
        <div className="totals__row totals__row--total">
          <span>Gesamtbetrag</span>
          <MoneyText cents={invoice.total_cents} />
        </div>
      </div>
    </div>
  );
}

function InvoiceView({
  invoice,
  orderNumber,
  plate,
}: {
  invoice: Invoice;
  orderNumber: string;
  plate: string;
}) {
  return (
    <section className="card" data-od-id="invoice-view">
      <div className="card__header">
        <div>
          <p className="caption muted">Rechnung</p>
          <p className="h3 mono tnum">{orderNumber}</p>
        </div>
      </div>
      <div className="meta" style={{ marginBottom: 'var(--space-3)' }}>
        <div className="meta__item">
          <span className="meta__label">Kennzeichen</span>
          <span className="meta__value tnum">{plate}</span>
        </div>
        <div className="meta__item">
          <span className="meta__label">Rechnungsdatum</span>
          <span className="meta__value tnum">{formatDate(invoice.created_at)}</span>
        </div>
      </div>
      <div className="position-list">
        {invoice.items.map((item, index) => (
          <PositionRow key={index} item={item} />
        ))}
      </div>
      <TotalsBlock invoice={invoice} />
    </section>
  );
}

function InvoiceEmpty({ title, body }: { title: string; body: string }) {
  return (
    <div className="card">
      <div className="empty" data-od-id="invoice-empty">
        <svg
          className="empty__icon"
          width="32"
          height="32"
          viewBox="0 0 24 24"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M6 3h9l3 3v15H6z"
            stroke="currentColor"
            strokeWidth="1.6"
            strokeLinejoin="round"
          />
          <path d="M9 12h6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
        </svg>
        <p className="empty__title">{title}</p>
        <p className="empty__body">{body}</p>
      </div>
    </div>
  );
}

async function orderIsPending(orderNumber: string, plate: string): Promise<boolean> {
  try {
    const order = await apiGet<OrderStatusResponse>(
      `/api/public/orders/${encodeURIComponent(orderNumber)}`,
      { plate },
    );
    return order.status !== 'fertig' && order.status !== 'abgeholt';
  } catch {
    return false;
  }
}

export function InvoicePage() {
  const [orderNumber, setOrderNumber] = useState('');
  const [plate, setPlate] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<Result>({ kind: 'idle' });

  const orderError =
    submitted && orderNumber.trim().length < 3 ? 'Bitte geben Sie die Auftragsnummer an.' : null;
  const plateError =
    submitted && plate.trim().length < 3 ? 'Bitte geben Sie das Kennzeichen an.' : null;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);
    const number = orderNumber.trim();
    const normalizedPlate = plate.trim();
    if (number.length < 3 || normalizedPlate.length < 3) return;

    setLoading(true);
    try {
      const data = await apiGet<InvoiceResponse>(
        `/api/public/orders/${encodeURIComponent(number)}/invoice`,
        { plate: normalizedPlate },
      );
      if (data && data.invoice) {
        setResult({ kind: 'invoice', invoice: data.invoice, orderNumber: number, plate: normalizedPlate });
      } else {
        setResult({ kind: 'notFound' });
      }
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        const pending = await orderIsPending(number, normalizedPlate);
        setResult({ kind: pending ? 'notReady' : 'notFound' });
      } else {
        setResult({
          kind: 'error',
          message: error instanceof ApiError ? error.message : GENERIC_ERROR,
        });
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="page">
      <header>
        <h1 className="page__title" data-od-id="page-head">
          Rechnung
        </h1>
        <p className="page__text">
          Geben Sie Ihre Auftragsnummer und das Kennzeichen ein, um die Rechnung abzurufen.
        </p>
      </header>

      <form
        className="card form-col"
        id="invoice-form"
        data-od-id="invoice-form"
        onSubmit={handleSubmit}
        noValidate
      >
        <Field
          label="Auftragsnummer"
          required
          helper="z. B. AUF-2025-0192"
          error={orderError}
          className="mono"
          name="order"
          autoCapitalize="characters"
          value={orderNumber}
          onChange={(event) => setOrderNumber(event.target.value.toUpperCase())}
        />
        <Field
          label="Kennzeichen"
          required
          helper="z. B. B-CD 5678"
          error={plateError}
          name="plate"
          autoCapitalize="characters"
          value={plate}
          onChange={(event) => setPlate(event.target.value.toUpperCase())}
        />
        <div className="form-actions">
          <Button
            type="submit"
            variant="primary"
            loading={loading}
            className="btn--form"
            data-od-id="submit-invoice"
          >
            Rechnung ansehen
          </Button>
        </div>
      </form>

      <div
        id="invoice-result"
        className="invoice-result"
        aria-live="polite"
        data-od-id="invoice-result"
      >
        {result.kind === 'invoice' ? (
          <InvoiceView
            invoice={result.invoice}
            orderNumber={result.orderNumber}
            plate={result.plate}
          />
        ) : null}
        {result.kind === 'notReady' ? (
          <InvoiceEmpty title={NOT_READY_TITLE} body={NOT_READY_BODY} />
        ) : null}
        {result.kind === 'notFound' ? (
          <InvoiceEmpty title={NOT_FOUND_TITLE} body={NOT_FOUND_BODY} />
        ) : null}
        {result.kind === 'error' ? <ErrorBanner message={result.message} /> : null}
      </div>
    </section>
  );
}
