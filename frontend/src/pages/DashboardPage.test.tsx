import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
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

import { apiGet } from '../api/client';
import type { Mock } from 'vitest';
import { DashboardPage } from './DashboardPage';

function row(overrides: Record<string, unknown> = {}) {
  return {
    order_number: 'AUF-2025-0187',
    status: 'angefragt',
    plate: 'B-AB 1234',
    preferred_date: '2025-03-14T00:00:00Z',
    customer_name: 'Anna Schmidt',
    total_cents: 35819,
    ...overrides,
  };
}

function renderPage() {
  return render(
    <MemoryRouter>
      <DashboardPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.resetAllMocks();
});

afterEach(() => {
  cleanup();
});

describe('DashboardPage', () => {
  it('loads the fixed dashboard path and renders the three blocks', async () => {
    (apiGet as Mock).mockResolvedValueOnce({
      open: [
        row(),
        row({ order_number: 'AUF-2025-0188', status: 'bestaetigt' }),
        row({ order_number: 'AUF-2025-0195', status: 'in_arbeit' }),
      ],
      finished_today: [row({ order_number: 'AUF-2025-0192', status: 'fertig' })],
      revenue_month_cents: 123456,
    });

    renderPage();

    await screen.findByTestId('dashboard');

    expect(apiGet).toHaveBeenCalledWith('/api/workshop/dashboard');
    expect(screen.getByTestId('stat-open')).toBeInTheDocument();
    expect(screen.getByTestId('stat-done-today')).toBeInTheDocument();
    expect(screen.getByTestId('stat-revenue')).toBeInTheDocument();
    expect(screen.getByTestId('dashboard-open-orders')).toBeInTheDocument();
    expect(screen.getByTestId('dashboard-done-today')).toBeInTheDocument();
  });

  it('counts the open orders and the ones finished today', async () => {
    (apiGet as Mock).mockResolvedValueOnce({
      open: [
        row(),
        row({ order_number: 'AUF-2025-0188', status: 'bestaetigt' }),
        row({ order_number: 'AUF-2025-0195', status: 'in_arbeit' }),
      ],
      finished_today: [
        row({ order_number: 'AUF-2025-0192', status: 'fertig' }),
        row({ order_number: 'AUF-2025-0193', status: 'fertig' }),
      ],
      revenue_month_cents: 0,
    });

    renderPage();

    expect(await screen.findByTestId('stat-open-value')).toHaveTextContent('3');
    expect(screen.getByTestId('stat-done-value')).toHaveTextContent('2');
  });

  it('shows the monthly revenue from integer cents in euro format', async () => {
    (apiGet as Mock).mockResolvedValueOnce({
      open: [],
      finished_today: [],
      revenue_month_cents: 123456,
    });

    renderPage();

    expect(await screen.findByTestId('stat-revenue-value')).toHaveTextContent('1.234,56 €');
  });

  it('links every open order to its detail page', async () => {
    (apiGet as Mock).mockResolvedValueOnce({
      open: [row({ order_number: 'AUF-2025-0187' })],
      finished_today: [],
      revenue_month_cents: 0,
    });

    renderPage();

    const link = await screen.findByRole('link', { name: /AUF-2025-0187/ });
    expect(link).toHaveAttribute('href', '/werkstatt/auftraege/AUF-2025-0187');
  });

  it('reloads the numbers when the refresh control is used', async () => {
    (apiGet as Mock)
      .mockResolvedValueOnce({
        open: [row()],
        finished_today: [],
        revenue_month_cents: 100000,
      })
      .mockResolvedValueOnce({
        open: [row()],
        finished_today: [row({ order_number: 'AUF-2025-0192', status: 'fertig' })],
        revenue_month_cents: 222250,
      });

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByTestId('stat-revenue-value')).toHaveTextContent('1.000,00 €');
    expect(screen.getByTestId('stat-done-value')).toHaveTextContent('0');

    await user.click(screen.getByRole('button', { name: 'Aktualisieren' }));

    await waitFor(() =>
      expect(screen.getByTestId('stat-revenue-value')).toHaveTextContent('2.222,50 €'),
    );
    expect(screen.getByTestId('stat-done-value')).toHaveTextContent('1');
    expect(apiGet).toHaveBeenCalledTimes(2);
  });

  it('renders a readable error state instead of throwing when the dashboard request fails', async () => {
    (apiGet as Mock).mockRejectedValueOnce(new Error('boom'));

    renderPage();

    expect(await screen.findByRole('alert')).toBeInTheDocument();
  });
});
