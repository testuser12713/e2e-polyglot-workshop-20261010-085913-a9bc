import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiGet: vi.fn(),
    apiPost: vi.fn(),
    apiPatch: vi.fn(),
    apiDelete: vi.fn(),
  };
});

import { ApiError, apiDelete, apiGet, apiPatch, apiPost } from '../api/client';
import type { Mock } from 'vitest';
import { OrderDetailPage } from './OrderDetailPage';

const ORDER_NUMBER = 'AUF-2025-0187';

function baseOrder(overrides: Record<string, unknown> = {}) {
  return {
    order_number: ORDER_NUMBER,
    status: 'angefragt',
    plate: 'B-AB 1234',
    preferred_date: '2025-03-14T00:00:00Z',
    customer_name: 'Anna Schmidt',
    total_cents: 35819,
    customer: { name: 'Anna Schmidt', email: 'anna.schmidt@example.de', phone: '0151 23456789' },
    vehicle: {
      plate: 'B-AB 1234',
      brand: 'Volkswagen',
      model: 'Golf 8',
      mileage: 128450,
    },
    description: 'Bremsbeläge vorne verschlissen, quietscht beim Bremsen.',
    items: [
      {
        id: 1,
        kind: 'labour',
        description: 'Bremsbeläge vorne erneuern',
        hours: 2.5,
        unit_price_cents: 8900,
        line_total_cents: 22250,
      },
      {
        id: 2,
        kind: 'part',
        description: 'Bremsbelag-Set vorne (ATE)',
        quantity: 1,
        unit_price_cents: 7850,
        line_total_cents: 7850,
      },
    ],
    status_history: [{ status: 'angefragt', changed_at: '2025-03-10T09:12:00Z' }],
    subtotal_cents: 30100,
    ...overrides,
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={[`/werkstatt/auftraege/${ORDER_NUMBER}`]}>
      <Routes>
        <Route path="/werkstatt/auftraege/:orderNumber" element={<OrderDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

async function loadOrderInPage() {
  renderPage();
  await screen.findByRole('heading', { name: ORDER_NUMBER });
}

beforeEach(() => {
  vi.resetAllMocks();
});

afterEach(() => {
  cleanup();
});

describe('OrderDetailPage', () => {
  it('loads the order via the fixed detail path and renders the totals', async () => {
    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });

    await loadOrderInPage();

    expect(apiGet).toHaveBeenCalledWith(`/api/workshop/orders/${ORDER_NUMBER}`);
    expect(screen.getByText('301,00 €')).toBeInTheDocument();
    expect(screen.getByText('57,19 €')).toBeInTheDocument();
    expect(screen.getByText('358,19 €')).toBeInTheDocument();
  });

  it('adds a labour position and refreshes the totals', async () => {
    const newItem = {
      id: 3,
      kind: 'labour',
      description: 'Ölwechsel',
      hours: 2.5,
      unit_price_cents: 8900,
      line_total_cents: 22250,
    };
    const refreshed = baseOrder({
      items: [...baseOrder().items, newItem],
      subtotal_cents: 52350,
    });

    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });
    (apiPost as Mock).mockResolvedValueOnce({ item: newItem });
    (apiGet as Mock).mockResolvedValueOnce({ order: refreshed });

    const user = userEvent.setup();
    await loadOrderInPage();

    await user.click(screen.getByRole('button', { name: 'Position hinzufügen' }));
    await user.type(screen.getByLabelText(/Beschreibung/), 'Ölwechsel');
    await user.type(screen.getByLabelText(/Arbeitszeit/), '2.5');
    await user.click(screen.getByRole('button', { name: 'Speichern' }));

    await waitFor(() =>
      expect(apiPost).toHaveBeenCalledWith(`/api/workshop/orders/${ORDER_NUMBER}/items`, {
        kind: 'labour',
        description: 'Ölwechsel',
        hours: 2.5,
      }),
    );
    expect(await screen.findByText('523,50 €')).toBeInTheDocument();
    expect(await screen.findByText('622,97 €')).toBeInTheDocument();
  });

  it('adds a part position with quantity and unit price sent as cents', async () => {
    const newItem = {
      id: 3,
      kind: 'part',
      description: 'Bremsscheiben vorne',
      quantity: 2,
      unit_price_cents: 6400,
      line_total_cents: 12800,
    };
    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });
    (apiPost as Mock).mockResolvedValueOnce({ item: newItem });
    (apiGet as Mock).mockResolvedValueOnce({
      order: baseOrder({ items: [...baseOrder().items, newItem], subtotal_cents: 42900 }),
    });

    const user = userEvent.setup();
    await loadOrderInPage();

    await user.click(screen.getByRole('button', { name: 'Position hinzufügen' }));
    await user.selectOptions(screen.getByLabelText(/Typ/), 'part');
    await user.type(screen.getByLabelText(/Bezeichnung/), 'Bremsscheiben vorne');
    await user.type(screen.getByLabelText(/Menge/), '2');
    await user.type(screen.getByLabelText(/Einzelpreis/), '64,00');
    await user.click(screen.getByRole('button', { name: 'Speichern' }));

    await waitFor(() =>
      expect(apiPost).toHaveBeenCalledWith(`/api/workshop/orders/${ORDER_NUMBER}/items`, {
        kind: 'part',
        description: 'Bremsscheiben vorne',
        quantity: 2,
        unit_price_cents: 6400,
      }),
    );
  });

  it('deletes a position and refreshes the totals', async () => {
    const refreshed = baseOrder({
      items: [baseOrder().items[1]],
      subtotal_cents: 7850,
    });
    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });
    (apiDelete as Mock).mockResolvedValueOnce(undefined);
    (apiGet as Mock).mockResolvedValueOnce({ order: refreshed });

    const user = userEvent.setup();
    await loadOrderInPage();

    const row = screen.getByTestId('position-1');
    await user.click(within(row).getByRole('button', { name: 'Löschen' }));

    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: 'Löschen' }));

    await waitFor(() =>
      expect(apiDelete).toHaveBeenCalledWith(`/api/workshop/orders/${ORDER_NUMBER}/items/1`),
    );
    expect(await screen.findByText('93,42 €')).toBeInTheDocument();
  });

  it('edits a position via PATCH', async () => {
    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });
    (apiPatch as Mock).mockResolvedValueOnce({ item: baseOrder().items[0] });
    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });

    const user = userEvent.setup();
    await loadOrderInPage();

    const row = screen.getByTestId('position-1');
    await user.click(within(row).getByRole('button', { name: 'Bearbeiten' }));

    const dialog = await screen.findByRole('dialog');
    const hoursInput = within(dialog).getByLabelText(/Arbeitszeit/);
    await user.clear(hoursInput);
    await user.type(hoursInput, '3');
    await user.click(within(dialog).getByRole('button', { name: 'Speichern' }));

    await waitFor(() =>
      expect(apiPatch).toHaveBeenCalledWith(`/api/workshop/orders/${ORDER_NUMBER}/items/1`, {
        kind: 'labour',
        description: 'Bremsbeläge vorne erneuern',
        hours: 3,
      }),
    );
  });

  it('sets the next status through the confirm dialog and shows the new status', async () => {
    const updatedHistory = [
      { status: 'angefragt', changed_at: '2025-03-10T09:12:00Z' },
      { status: 'bestaetigt', changed_at: '2025-03-11T10:00:00Z' },
    ];
    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });
    (apiPost as Mock).mockResolvedValueOnce({
      order_number: ORDER_NUMBER,
      status: 'bestaetigt',
      status_history: updatedHistory,
    });

    const user = userEvent.setup();
    await loadOrderInPage();

    await user.click(screen.getByRole('button', { name: 'Bestätigen' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('Auftrag bestätigen?')).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Bestätigen' }));

    await waitFor(() =>
      expect(apiPost).toHaveBeenCalledWith(`/api/workshop/orders/${ORDER_NUMBER}/status`, {
        status: 'bestaetigt',
      }),
    );
    expect(within(screen.getByTestId('order-status')).getByText('bestätigt')).toBeInTheDocument();
    expect(screen.getByText('Auftrag bestätigt.')).toBeInTheDocument();
  });

  it('shows visible feedback when a transition is refused with 409', async () => {
    (apiGet as Mock).mockResolvedValueOnce({ order: baseOrder() });
    (apiPost as Mock).mockRejectedValueOnce(
      new ApiError(409, 'conflict', 'Der Übergang angefragt → abgeholt ist nicht erlaubt.'),
    );

    const user = userEvent.setup();
    await loadOrderInPage();

    await user.click(screen.getByRole('button', { name: 'Abgeholt' }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: 'Abgeholt' }));

    expect(
      await screen.findByText(/Dieser Übergang ist nicht erlaubt\./),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Der Übergang angefragt → abgeholt ist nicht erlaubt\./),
    ).toBeInTheDocument();
  });

  it('locks the editing controls on an order that was picked up', async () => {
    (apiGet as Mock).mockResolvedValueOnce({
      order: baseOrder({ status: 'abgeholt' }),
    });

    await loadOrderInPage();

    expect(screen.getByRole('button', { name: 'Position hinzufügen' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Bearbeiten' })).not.toBeInTheDocument();
  });
});
