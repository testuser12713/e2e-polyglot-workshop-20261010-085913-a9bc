import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiGet: vi.fn(),
  };
});

import { ApiError, apiGet } from '../api/client';
import type { Mock } from 'vitest';
import { OrdersPage } from './OrdersPage';

interface OrderRow {
  order_number: string;
  status: string;
  plate: string;
  preferred_date: string;
  customer_name: string;
  total_cents: number;
}

const ORDERS: OrderRow[] = [
  {
    order_number: 'AUF-2025-0187',
    status: 'angefragt',
    plate: 'B-AB 1234',
    preferred_date: '2025-03-14T00:00:00Z',
    customer_name: 'Anna Schmidt',
    total_cents: 35819,
  },
  {
    order_number: 'AUF-2025-0188',
    status: 'bestaetigt',
    plate: 'B-EF 9012',
    preferred_date: '2025-03-17T00:00:00Z',
    customer_name: 'Miriam Krause',
    total_cents: 12345,
  },
  {
    order_number: 'AUF-2025-0192',
    status: 'fertig',
    plate: 'B-CD 5678',
    preferred_date: '2025-03-15T00:00:00Z',
    customer_name: 'Thomas Becker',
    total_cents: 9900,
  },
];

function serverFilter(params?: Record<string, string>): OrderRow[] {
  const status = params?.status ?? '';
  const plate = (params?.plate ?? '').toLowerCase();
  return ORDERS.filter(
    (order) =>
      (!status || order.status === status) &&
      (!plate || order.plate.toLowerCase().includes(plate)),
  );
}

function mockOrders(): void {
  (apiGet as Mock).mockImplementation(
    (_path: string, params?: Record<string, string>) =>
      Promise.resolve({ orders: serverFilter(params) }),
  );
}

function renderPage() {
  return render(
    <MemoryRouter>
      <OrdersPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.resetAllMocks();
  mockOrders();
});

afterEach(() => {
  cleanup();
});

describe('OrdersPage', () => {
  it('loads the orders through the fixed workshop path and renders the list', async () => {
    renderPage();

    const list = await screen.findByTestId('orders-list');
    expect(apiGet).toHaveBeenCalledWith('/api/workshop/orders', { status: '', plate: '' });

    expect(within(list).getByText('AUF-2025-0187')).toBeInTheDocument();
    expect(within(list).getByText('B-AB 1234')).toBeInTheDocument();
    expect(within(list).getByText('Anna Schmidt')).toBeInTheDocument();
    expect(within(list).getByText('14.03.2025')).toBeInTheDocument();
    expect(within(list).getByText('358,19 €')).toBeInTheDocument();
    expect(within(list).getByText('angefragt')).toBeInTheDocument();
    expect(screen.getByTestId('orders-count')).toHaveTextContent('3');

    const link = screen.getByRole('link', { name: /AUF-2025-0187/ });
    expect(link).toHaveAttribute('href', '/werkstatt/auftraege/AUF-2025-0187');
  });

  it('narrows the hit count when filtering by status', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByTestId('orders-list');
    await user.selectOptions(screen.getByLabelText('Status'), 'bestaetigt');

    await waitFor(() =>
      expect(apiGet).toHaveBeenLastCalledWith('/api/workshop/orders', {
        status: 'bestaetigt',
        plate: '',
      }),
    );

    const list = await screen.findByTestId('orders-list');
    expect(within(list).getByText('AUF-2025-0188')).toBeInTheDocument();
    expect(within(list).queryByText('AUF-2025-0187')).not.toBeInTheDocument();
    expect(screen.getByTestId('orders-count')).toHaveTextContent('1');
  });

  it('searches by plate substring regardless of casing', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByTestId('orders-list');
    await user.type(screen.getByPlaceholderText('Kennzeichen suchen'), 'ef');

    await waitFor(
      () =>
        expect(apiGet).toHaveBeenLastCalledWith('/api/workshop/orders', {
          status: '',
          plate: 'ef',
        }),
      { timeout: 1500 },
    );

    const list = await screen.findByTestId('orders-list');
    expect(within(list).getByText('AUF-2025-0188')).toBeInTheDocument();
    expect(within(list).queryByText('AUF-2025-0187')).not.toBeInTheDocument();
    expect(screen.getByTestId('orders-count')).toHaveTextContent('1');
  });

  it('shows the filter empty state when nothing matches', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByTestId('orders-list');
    await user.type(screen.getByPlaceholderText('Kennzeichen suchen'), 'ZZ-9999');

    await waitFor(
      () => expect(screen.getByTestId('orders-empty')).toBeInTheDocument(),
      { timeout: 1500 },
    );
    expect(screen.getByText('Keine Treffer für diesen Filter')).toBeInTheDocument();
    expect(screen.getByTestId('orders-count')).toHaveTextContent('0');
  });

  it('shows the no-orders empty state when there are no filters and no orders', async () => {
    (apiGet as Mock).mockResolvedValue({ orders: [] });
    renderPage();

    expect(await screen.findByTestId('orders-empty')).toBeInTheDocument();
    expect(screen.getByText('Noch keine Aufträge')).toBeInTheDocument();
  });

  it('shows an error banner when the request fails', async () => {
    (apiGet as Mock).mockRejectedValue(
      new ApiError(500, 'internal_error', 'Serverfehler.'),
    );
    renderPage();

    expect(await screen.findByRole('alert')).toHaveTextContent('Serverfehler.');
  });
});
