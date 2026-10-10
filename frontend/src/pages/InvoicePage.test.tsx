import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { UserEvent } from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { apiGet, ApiError } from '../api/client';
import { InvoicePage } from './InvoicePage';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, apiGet: vi.fn() };
});

const mockedApiGet = vi.mocked(apiGet);

const STORED_INVOICE = {
  invoice: {
    items: [
      {
        kind: 'labour',
        description: 'Ölwechsel inkl. Ölfilter',
        hours: 1,
        quantity: null,
        unit_price_cents: 8900,
        line_total_cents: 8900,
      },
      {
        kind: 'part',
        description: 'Motoröl 5W-30, 5 l',
        hours: null,
        quantity: 1,
        unit_price_cents: 6200,
        line_total_cents: 6200,
      },
      {
        kind: 'part',
        description: 'Ölfilter',
        hours: null,
        quantity: 1,
        unit_price_cents: 1450,
        line_total_cents: 1450,
      },
    ],
    subtotal_cents: 16550,
    tax_cents: 3145,
    total_cents: 19695,
    created_at: '2025-03-28T10:00:00Z',
  },
};

async function submit(
  user: UserEvent,
  order = 'AUF-2025-0192',
  plate = 'B-CD 5678',
) {
  await user.type(screen.getByLabelText(/Auftragsnummer/), order);
  await user.type(screen.getByLabelText(/Kennzeichen/), plate);
  await user.click(screen.getByRole('button', { name: 'Rechnung ansehen' }));
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('InvoicePage', () => {
  it('renders positions, subtotal, VAT and total for a stored invoice', async () => {
    mockedApiGet.mockResolvedValueOnce(STORED_INVOICE);
    const user = userEvent.setup();
    render(<InvoicePage />);

    await submit(user);

    expect(await screen.findByText('Ölwechsel inkl. Ölfilter')).toBeInTheDocument();
    expect(screen.getByText('Motoröl 5W-30, 5 l')).toBeInTheDocument();
    expect(screen.getByText('Ölfilter')).toBeInTheDocument();
    expect(screen.getByText('Zwischensumme')).toBeInTheDocument();
    expect(screen.getByText('19 % MwSt.')).toBeInTheDocument();
    expect(screen.getByText('Gesamtbetrag')).toBeInTheDocument();
    expect(screen.getByText('165,50 \u20ac')).toBeInTheDocument();
    expect(screen.getByText('31,45 \u20ac')).toBeInTheDocument();
    expect(screen.getByText('196,95 \u20ac')).toBeInTheDocument();
    expect(screen.getByText('28.03.2025')).toBeInTheDocument();
  });

  it('renders the neutral message when the order or plate does not match', async () => {
    mockedApiGet.mockRejectedValue(new ApiError(404, 'not_found', 'Nicht gefunden.'));
    const user = userEvent.setup();
    render(<InvoicePage />);

    await submit(user, 'AUF-0000-0000', 'B-XX 0000');

    expect(await screen.findByText('Rechnung nicht gefunden')).toBeInTheDocument();
    expect(screen.queryByText('Zwischensumme')).not.toBeInTheDocument();
  });

  it('explains that the invoice is available once the order is finished', async () => {
    mockedApiGet
      .mockRejectedValueOnce(new ApiError(404, 'not_found', 'Nicht gefunden.'))
      .mockResolvedValueOnce({ order_number: 'AUF-2025-0187', status: 'in_arbeit' });
    const user = userEvent.setup();
    render(<InvoicePage />);

    await submit(user, 'AUF-2025-0187', 'B-AB 1234');

    expect(
      await screen.findByText('Die Rechnung ist ab dem Status fertig verfügbar.'),
    ).toBeInTheDocument();
    expect(screen.queryByText('Zwischensumme')).not.toBeInTheDocument();
  });
});
