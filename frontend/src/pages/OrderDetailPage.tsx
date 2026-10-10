import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ApiError, apiDelete, apiGet, apiPatch, apiPost } from '../api/client';
import { Button } from '../components/Button';
import { ErrorBanner } from '../components/ErrorBanner';
import { Field, SelectField } from '../components/Field';
import { StatusBadge } from '../components/StatusBadge';
import { formatMoney, formatStatusLabel } from '../lib/format';
import './OrderDetailPage.css';

interface OrderItem {
  id: number;
  kind: 'labour' | 'part';
  description: string;
  hours?: number | null;
  quantity?: number | null;
  unit_price_cents?: number | null;
  line_total_cents?: number | null;
}

interface OrderStatusHistoryEntry {
  status: string;
  changed_at: string;
}

interface OrderCustomer {
  name: string;
  email: string;
  phone: string;
}

interface OrderVehicle {
  plate: string;
  brand: string;
  model: string;
  mileage: number;
}

interface OrderDetail {
  order_number: string;
  status: string;
  plate?: string | null;
  preferred_date?: string | null;
  customer_name?: string | null;
  total_cents?: number | null;
  customer?: OrderCustomer | null;
  vehicle?: OrderVehicle | null;
  description?: string | null;
  items?: OrderItem[] | null;
  status_history?: OrderStatusHistoryEntry[] | null;
  subtotal_cents?: number | null;
}

interface OrderDetailResponse {
  order: OrderDetail;
}

interface StatusUpdateResponse {
  order_number: string;
  status: string;
  status_history: OrderStatusHistoryEntry[];
}

interface PositionPayload {
  kind: 'labour' | 'part';
  description: string;
  hours?: number;
  quantity?: number;
  unit_price_cents?: number;
}

type Feedback = { kind: 'success' | 'error'; message: string };

interface ConfirmState {
  title: string;
  body: string;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
}

const STATUS_CHAIN = ['angefragt', 'bestaetigt', 'in_arbeit', 'fertig', 'abgeholt'];
const TRANSITION_TARGETS = STATUS_CHAIN.slice(1);

const TRANSITION_TITLES: Record<string, string> = {
  bestaetigt: 'Auftrag bestätigen?',
  in_arbeit: 'In Arbeit setzen?',
  fertig: 'Auftrag fertig melden?',
  abgeholt: 'Als abgeholt markieren?',
};

const TRANSITION_LABELS: Record<string, string> = {
  bestaetigt: 'Bestätigen',
  in_arbeit: 'In Arbeit setzen',
  fertig: 'Fertig melden',
  abgeholt: 'Abgeholt',
};

const TRANSITION_SUCCESS: Record<string, string> = {
  bestaetigt: 'Auftrag bestätigt.',
  in_arbeit: 'Auftrag in Arbeit gesetzt.',
  fertig: 'Auftrag fertig gemeldet. Die Rechnung wird erstellt.',
  abgeholt: 'Auftrag als abgeholt markiert.',
};

function itemLineTotal(item: OrderItem): number {
  if (typeof item.line_total_cents === 'number') return item.line_total_cents;
  const rate = item.unit_price_cents ?? 0;
  if (item.kind === 'labour') return Math.round((item.hours ?? 0) * rate);
  return (item.quantity ?? 0) * rate;
}

function orderSubtotal(order: OrderDetail): number {
  if (typeof order.subtotal_cents === 'number') return order.subtotal_cents;
  return (order.items ?? []).reduce((sum, item) => sum + itemLineTotal(item), 0);
}

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

function formatDateTime(value?: string | null): string {
  const date = parseDate(value);
  if (!date) return '—';
  const day = date.toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    timeZone: 'Europe/Berlin',
  });
  const time = date.toLocaleTimeString('de-DE', {
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Europe/Berlin',
  });
  return `${day}, ${time} Uhr`;
}

function formatHours(value?: number | null): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '—';
  return `${value.toFixed(1).replace('.', ',')} h`;
}

function formatQuantity(value?: number | null): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '—';
  return `${value} Stk.`;
}

function formatMileage(value?: number | null): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '—';
  return `${value.toLocaleString('de-DE')} km`;
}

function parseDecimal(value: string): number {
  const trimmed = value.trim();
  if (!trimmed) return Number.NaN;
  const parsed = Number(trimmed.replace(',', '.'));
  return Number.isFinite(parsed) ? parsed : Number.NaN;
}

function parseInteger(value: string): number {
  const trimmed = value.trim();
  if (!/^\d+$/.test(trimmed)) return Number.NaN;
  return Number.parseInt(trimmed, 10);
}

function parseEuroToCents(value: string): number | null {
  const euros = parseDecimal(value);
  if (!Number.isFinite(euros)) return null;
  return Math.round(euros * 100);
}

function centsToEuroInput(cents: number): string {
  return (cents / 100).toFixed(2).replace('.', ',');
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  return 'Es ist ein Fehler aufgetreten.';
}

function ConfirmDialog({ state, onCancel }: { state: ConfirmState; onCancel: () => void }) {
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onCancel();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onCancel]);

  return (
    <div
      className="overlay"
      role="presentation"
      onClick={(event) => {
        if (event.target === event.currentTarget) onCancel();
      }}
    >
      <div className="dialog" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
        <h2 className="dialog__title" id="confirm-title">
          {state.title}
        </h2>
        {state.body ? <p className="dialog__body">{state.body}</p> : null}
        <div className="dialog__footer">
          <Button variant="secondary" onClick={onCancel}>
            Abbrechen
          </Button>
          <Button
            variant={state.destructive ? 'destructive' : 'primary'}
            onClick={state.onConfirm}
          >
            {state.confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}

function PositionDialog({
  item,
  submitting,
  error,
  onCancel,
  onSubmit,
}: {
  item: OrderItem | null;
  submitting: boolean;
  error: string | null;
  onCancel: () => void;
  onSubmit: (payload: PositionPayload) => void;
}) {
  const editing = item !== null;
  const [kind, setKind] = useState<'labour' | 'part'>(
    item?.kind === 'part' ? 'part' : 'labour',
  );
  const [description, setDescription] = useState(item?.description ?? '');
  const [hours, setHours] = useState(
    item && item.kind === 'labour' && item.hours != null ? String(item.hours) : '',
  );
  const [quantity, setQuantity] = useState(
    item && item.kind === 'part' && item.quantity != null ? String(item.quantity) : '',
  );
  const [unitPrice, setUnitPrice] = useState(
    item && item.kind === 'part' && item.unit_price_cents != null
      ? centsToEuroInput(item.unit_price_cents)
      : '',
  );
  const [errors, setErrors] = useState<Record<string, string>>({});

  const isLabour = kind === 'labour';
  const descriptionLabel = isLabour ? 'Beschreibung' : 'Bezeichnung';

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const next: Record<string, string> = {};
    const trimmed = description.trim();
    if (trimmed.length < 2) {
      next.description = isLabour
        ? 'Bitte eine Beschreibung angeben.'
        : 'Bitte eine Bezeichnung angeben.';
    }

    let payload: PositionPayload;
    if (isLabour) {
      const parsedHours = parseDecimal(hours);
      if (!(parsedHours > 0)) {
        next.hours = 'Bitte die Arbeitszeit in Stunden angeben.';
      }
      payload = { kind: 'labour', description: trimmed, hours: parsedHours };
    } else {
      const parsedQuantity = parseInteger(quantity);
      if (!(parsedQuantity >= 1)) {
        next.quantity = 'Bitte eine Menge von mindestens 1 angeben.';
      }
      const parsedCents = parseEuroToCents(unitPrice);
      if (parsedCents === null || parsedCents < 0) {
        next.unit_price = 'Bitte einen Einzelpreis in Euro angeben.';
      }
      payload = {
        kind: 'part',
        description: trimmed,
        quantity: parsedQuantity,
        unit_price_cents: parsedCents ?? 0,
      };
    }

    setErrors(next);
    if (Object.keys(next).length > 0) return;
    onSubmit(payload);
  }

  return (
    <div
      className="overlay"
      role="presentation"
      onClick={(event) => {
        if (event.target === event.currentTarget) onCancel();
      }}
    >
      <div className="dialog" role="dialog" aria-modal="true" aria-labelledby="position-title">
        <h2 className="dialog__title" id="position-title">
          {editing ? 'Position bearbeiten' : 'Position hinzufügen'}
        </h2>
        {error ? <ErrorBanner message={error} /> : null}
        <form className="position-form" onSubmit={handleSubmit} noValidate>
          <SelectField
            label="Typ"
            value={kind}
            onChange={(event) => setKind(event.target.value === 'part' ? 'part' : 'labour')}
          >
            <option value="labour">Arbeitsposition</option>
            <option value="part">Teilposition</option>
          </SelectField>

          <Field
            label={descriptionLabel}
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            error={errors.description}
            required
          />

          {isLabour ? (
            <Field
              label="Arbeitszeit"
              type="number"
              step="0.5"
              min="0"
              inputMode="decimal"
              value={hours}
              onChange={(event) => setHours(event.target.value)}
              error={errors.hours}
              helper="Stunden mit einer Dezimalstelle, z. B. 2,5."
              required
            />
          ) : (
            <div className="field-2">
              <Field
                label="Menge"
                type="number"
                min="1"
                step="1"
                inputMode="numeric"
                value={quantity}
                onChange={(event) => setQuantity(event.target.value)}
                error={errors.quantity}
                required
              />
              <Field
                label="Einzelpreis (€)"
                type="text"
                inputMode="decimal"
                placeholder="z. B. 78,50"
                value={unitPrice}
                onChange={(event) => setUnitPrice(event.target.value)}
                error={errors.unit_price}
                required
              />
            </div>
          )}

          <div className="dialog__footer">
            <Button variant="secondary" onClick={onCancel} disabled={submitting}>
              Abbrechen
            </Button>
            <Button type="submit" variant="primary" loading={submitting}>
              Speichern
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

export function OrderDetailPage() {
  const { orderNumber } = useParams<{ orderNumber: string }>();
  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);
  const [positionDialog, setPositionDialog] = useState<{ item: OrderItem | null } | null>(null);
  const [positionError, setPositionError] = useState<string | null>(null);
  const [positionSubmitting, setPositionSubmitting] = useState(false);

  const path = `/api/workshop/orders/${encodeURIComponent(orderNumber ?? '')}`;

  const loadOrder = useCallback(
    async (silent = false) => {
      if (!orderNumber) {
        setLoadError('Auftragsnummer fehlt.');
        setLoading(false);
        return;
      }
      if (!silent) setLoading(true);
      try {
        const response = await apiGet<OrderDetailResponse>(
          `/api/workshop/orders/${encodeURIComponent(orderNumber)}`,
        );
        setOrder(response.order);
        setLoadError(null);
      } catch (error) {
        if (!silent) setLoadError(errorMessage(error));
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [orderNumber],
  );

  useEffect(() => {
    void loadOrder();
  }, [loadOrder]);

  async function applyTransition(target: string) {
    setConfirm(null);
    try {
      const response = await apiPost<StatusUpdateResponse>(`${path}/status`, { status: target });
      setOrder((previous) =>
        previous
          ? {
              ...previous,
              status: response.status,
              status_history: response.status_history,
            }
          : previous,
      );
      setFeedback({ kind: 'success', message: TRANSITION_SUCCESS[target] ?? 'Status gesetzt.' });
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setFeedback({
          kind: 'error',
          message: `Dieser Übergang ist nicht erlaubt. ${error.message}`,
        });
      } else {
        setFeedback({ kind: 'error', message: errorMessage(error) });
      }
    }
  }

  function requestTransition(target: string) {
    setFeedback(null);
    setConfirm({
      title: TRANSITION_TITLES[target] ?? 'Status setzen?',
      body: `Der Status des Auftrags ${order?.order_number ?? orderNumber ?? ''} wird auf „${formatStatusLabel(target)}“ gesetzt. Dies wird im Statusverlauf festgehalten.`,
      confirmLabel: TRANSITION_LABELS[target] ?? 'Bestätigen',
      onConfirm: () => {
        void applyTransition(target);
      },
    });
  }

  async function savePosition(payload: PositionPayload, itemId: number | null) {
    setPositionSubmitting(true);
    setPositionError(null);
    try {
      if (itemId !== null) {
        await apiPatch(`${path}/items/${itemId}`, payload);
      } else {
        await apiPost(`${path}/items`, payload);
      }
      setPositionDialog(null);
      setFeedback({
        kind: 'success',
        message: itemId !== null ? 'Position geändert.' : 'Position hinzugefügt.',
      });
      await loadOrder(true);
    } catch (error) {
      setPositionError(errorMessage(error));
    } finally {
      setPositionSubmitting(false);
    }
  }

  async function deleteItem(item: OrderItem) {
    setConfirm(null);
    try {
      await apiDelete(`${path}/items/${item.id}`);
      setFeedback({ kind: 'success', message: 'Position gelöscht.' });
      await loadOrder(true);
    } catch (error) {
      setFeedback({ kind: 'error', message: errorMessage(error) });
    }
  }

  function requestDelete(item: OrderItem) {
    setFeedback(null);
    setConfirm({
      title: 'Position löschen?',
      body: `„${item.description}“ wird entfernt. Die Beträge des Auftrags werden neu berechnet.`,
      confirmLabel: 'Löschen',
      destructive: true,
      onConfirm: () => {
        void deleteItem(item);
      },
    });
  }

  const displayNumber = order?.order_number ?? orderNumber ?? '';

  if (loading && !order) {
    return (
      <section className="order-detail" data-testid="order-detail">
        <p className="muted">Auftrag wird geladen …</p>
        <div className="skeleton-line" />
        <div className="skeleton-line" />
      </section>
    );
  }

  if (!order) {
    return (
      <section className="order-detail" data-testid="order-detail">
        <h1 className="order-detail__title mono">{displayNumber || 'Auftrag'}</h1>
        <ErrorBanner message={loadError ?? 'Der Auftrag konnte nicht geladen werden.'} />
        <Link className="caption" to="/werkstatt/auftraege">
          ← Zur Auftragsliste
        </Link>
      </section>
    );
  }

  const items = order.items ?? [];
  const subtotal = orderSubtotal(order);
  const vat = Math.round(subtotal * 0.19);
  const total = subtotal + vat;
  const history = order.status_history ?? [];
  const currentIndex = STATUS_CHAIN.indexOf(order.status);
  const nextStatus = currentIndex >= 0 ? STATUS_CHAIN[currentIndex + 1] : undefined;
  const locked = order.status === 'fertig' || order.status === 'abgeholt';
  const customer = order.customer ?? null;
  const vehicle = order.vehicle ?? null;

  return (
    <section className="order-detail" data-testid="order-detail">
      <div id="action-feedback" aria-live="polite" data-testid="action-feedback">
        {feedback?.kind === 'error' ? (
          <ErrorBanner message={feedback.message} onClose={() => setFeedback(null)} />
        ) : feedback ? (
          <div className="inline-alert inline-alert--success" role="status" aria-live="polite">
            <span className="inline-alert__message">{feedback.message}</span>
          </div>
        ) : null}
      </div>

      <header className="row-between">
        <div className="order-detail__head">
          <Link className="caption" to="/werkstatt/auftraege">
            ← Zur Auftragsliste
          </Link>
          <div className="order-detail__heading">
            <h1 className="order-detail__title mono">{displayNumber}</h1>
            <span data-testid="order-status">
              <StatusBadge status={order.status} />
            </span>
          </div>
        </div>
      </header>

      <section className="card" data-od-id="order-meta">
        <div className="card__header">
          <h2 className="card__title">Auftragsdaten</h2>
        </div>
        <div className="meta">
          <div className="meta__item">
            <span className="meta__label">Kunde</span>
            <span className="meta__value">{customer?.name ?? order.customer_name ?? '—'}</span>
          </div>
          <div className="meta__item">
            <span className="meta__label">Kennzeichen</span>
            <span className="meta__value mono">
              {(vehicle?.plate ?? order.plate ?? '—').toUpperCase()}
            </span>
          </div>
          <div className="meta__item">
            <span className="meta__label">Fahrzeug</span>
            <span className="meta__value">
              {vehicle ? `${vehicle.brand} ${vehicle.model}` : '—'}
            </span>
          </div>
          <div className="meta__item">
            <span className="meta__label">Kilometerstand</span>
            <span className="meta__value mono">{formatMileage(vehicle?.mileage)}</span>
          </div>
          <div className="meta__item">
            <span className="meta__label">Wunschtermin</span>
            <span className="meta__value mono">{formatDate(order.preferred_date)}</span>
          </div>
          <div className="meta__item">
            <span className="meta__label">Kontakt</span>
            <span className="meta__value">
              {customer ? `${customer.email} · ${customer.phone}` : '—'}
            </span>
          </div>
        </div>
        {order.description ? (
          <div className="meta__problem">
            <span className="meta__label">Problembeschreibung</span>
            <p className="meta__value">{order.description}</p>
          </div>
        ) : null}
      </section>

      <div className="detail-grid">
        <section className="card" data-od-id="positions-card">
          <div className="card__header">
            <h2 className="card__title">Positionen</h2>
            <Button
              variant="secondary"
              onClick={() => {
                setPositionError(null);
                setPositionDialog({ item: null });
              }}
              disabled={locked}
            >
              Position hinzufügen
            </Button>
          </div>

          {items.length === 0 ? (
            <div className="empty" data-od-id="positions-empty">
              <p className="empty__title">Noch keine Positionen</p>
              <p className="empty__body">Fügen Sie Arbeits- und Teilpositionen hinzu.</p>
            </div>
          ) : (
            <ul className="position-list" data-od-id="positions-list">
              {items.map((item) => {
                const isLabour = item.kind === 'labour';
                const unitPrice =
                  typeof item.unit_price_cents === 'number'
                    ? isLabour
                      ? `${formatMoney(item.unit_price_cents)}/h`
                      : formatMoney(item.unit_price_cents)
                    : '—';
                return (
                  <li
                    className="position-row"
                    data-od-id={`position-${item.id}`}
                    data-testid={`position-${item.id}`}
                    key={item.id}
                  >
                    <div className="position-row__desc">
                      <span className="position-row__title">{item.description}</span>
                      <span className="position-row__meta">
                        {isLabour ? 'Arbeitsposition' : 'Teilposition'}
                      </span>
                    </div>
                    <div className="position-row__cell" data-label="Menge/Zeit">
                      {isLabour ? formatHours(item.hours) : formatQuantity(item.quantity)}
                    </div>
                    <div className="position-row__cell" data-label="Einzelpreis">
                      {unitPrice}
                    </div>
                    <div className="position-row__cell position-row__sum" data-label="Summe">
                      {formatMoney(itemLineTotal(item))}
                    </div>
                    <div className="position-row__actions">
                      {locked ? (
                        <span className="caption muted">gesperrt</span>
                      ) : (
                        <>
                          <Button
                            variant="ghost"
                            onClick={() => {
                              setPositionError(null);
                              setPositionDialog({ item });
                            }}
                          >
                            Bearbeiten
                          </Button>
                          <Button variant="destructive" onClick={() => requestDelete(item)}>
                            Löschen
                          </Button>
                        </>
                      )}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}

          <div className="totals">
            <div className="totals__inner">
              <div className="totals__row">
                <span>Zwischensumme</span>
                <span className="numeric">{formatMoney(subtotal)}</span>
              </div>
              <div className="totals__row">
                <span>19 % MwSt.</span>
                <span className="numeric">{formatMoney(vat)}</span>
              </div>
              <div className="totals__row totals__row--total">
                <span>Gesamtbetrag</span>
                <span className="numeric">{formatMoney(total)}</span>
              </div>
            </div>
          </div>
          <p className="caption muted totals__note">
            {locked
              ? 'Der Auftrag ist abgeschlossen — Positionen sind gesperrt.'
              : 'Alle Beträge werden in ganzen Cent geführt; die Rechnung entsteht automatisch beim Status „fertig“.'}
          </p>
        </section>

        <section className="card" data-od-id="status-card">
          <div className="card__header">
            <h2 className="card__title">Status setzen</h2>
          </div>
          <div className="status-actions">
            {TRANSITION_TARGETS.map((target) => {
              const isNext = target === nextStatus;
              return (
                <Button
                  key={target}
                  variant={isNext ? 'primary' : 'secondary'}
                  block
                  onClick={() => requestTransition(target)}
                >
                  {TRANSITION_LABELS[target]}
                </Button>
              );
            })}
            <p className="caption muted">
              Erlaubt sind nur die Übergänge angefragt → bestätigt → in Arbeit → fertig →
              abgeholt.
            </p>
          </div>

          <h3 className="card__subtitle">Statusverlauf</h3>
          <ol className="timeline" data-od-id="timeline">
            {history.map((entry, index) => (
              <li className="timeline__item" key={`${entry.status}-${index}`}>
                <StatusBadge status={entry.status} />
                <time className="timeline__time">{formatDateTime(entry.changed_at)}</time>
              </li>
            ))}
          </ol>
        </section>
      </div>

      {confirm ? <ConfirmDialog state={confirm} onCancel={() => setConfirm(null)} /> : null}
      {positionDialog ? (
        <PositionDialog
          item={positionDialog.item}
          submitting={positionSubmitting}
          error={positionError}
          onCancel={() => {
            setPositionError(null);
            setPositionDialog(null);
          }}
          onSubmit={(payload) => {
            void savePosition(payload, positionDialog.item?.id ?? null);
          }}
        />
      ) : null}
    </section>
  );
}
