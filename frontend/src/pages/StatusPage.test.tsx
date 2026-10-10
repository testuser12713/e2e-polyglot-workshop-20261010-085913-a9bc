import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { UserEvent } from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError, apiGet } from '../api/client';
import { StatusPage } from './StatusPage';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, apiGet: vi.fn() };
});

const mockedApiGet = vi.mocked(apiGet);

const NEUTRAL = 'Status nicht gefunden — bitte Auftragsnummer und Kennzeichen prüfen';

const FOUND = {
  order_number: 'AUF-2025-0187',
  status: 'in_arbeit',
  status_history: [
    { status: 'angefragt', changed_at: '2025-03-10T09:12:00Z' },
    { status: 'bestaetigt', changed_at: '2025-03-11T14:05:00Z' },
    { status: 'in_arbeit', changed_at: '2025-03-14T08:30:00Z' },
  ],
};

async function submitForm(user: UserEvent, order = 'AUF-2025-0187', plate = 'B-AB 1234') {
  await user.type(screen.getByLabelText(/Auftragsnummer/), order);
  await user.type(screen.getByLabelText(/Kennzeichen/), plate);
  await user.click(screen.getByRole('button', { name: 'Status abfragen' }));
}

function emptyState(): Element | null {
  return document.querySelector('[data-od-id="status-empty"]');
}

beforeEach(() => {
  mockedApiGet.mockReset();
});

afterEach(() => {
  cleanup();
});

describe('StatusPage', () => {
  it('shows the status and the history for a matching order number and plate', async () => {
    const user = userEvent.setup();
    mockedApiGet.mockResolvedValue(FOUND);
    render(<StatusPage />);

    await submitForm(user);

    expect(await screen.findByText('Statusverlauf')).toBeInTheDocument();
    expect(screen.getByText('angefragt')).toBeInTheDocument();
    expect(screen.getByText('bestätigt')).toBeInTheDocument();
    expect(screen.getAllByText('in Arbeit').length).toBeGreaterThan(0);
    expect(screen.getByText('AUF-2025-0187')).toBeInTheDocument();
    expect(mockedApiGet).toHaveBeenCalledWith('/api/public/orders/AUF-2025-0187', {
      plate: 'B-AB 1234',
    });
  });

  it('shows the identical neutral message for a wrong plate', async () => {
    const user = userEvent.setup();
    mockedApiGet.mockRejectedValue(new ApiError(404, 'not_found', 'Nicht gefunden.'));
    render(<StatusPage />);

    await submitForm(user, 'AUF-2025-0187', 'B-ZZ 9999');

    expect(await screen.findByText(NEUTRAL)).toBeInTheDocument();
    expect(emptyState()).not.toBeNull();
    expect(screen.queryByText('Statusverlauf')).not.toBeInTheDocument();
  });

  it('shows the identical neutral message for an unknown order number', async () => {
    const user = userEvent.setup();
    mockedApiGet.mockRejectedValue(new ApiError(404, 'not_found', 'Nicht gefunden.'));
    render(<StatusPage />);

    await submitForm(user, 'AUF-9999-0000', 'B-AB 1234');

    expect(await screen.findByText(NEUTRAL)).toBeInTheDocument();
    expect(emptyState()).not.toBeNull();
    expect(screen.queryByText('Statusverlauf')).not.toBeInTheDocument();
  });

  it('renders the same neutral message whether the order is unknown or the plate mismatches', async () => {
    const user = userEvent.setup();
    mockedApiGet.mockRejectedValue(new ApiError(404, 'not_found', 'Nicht gefunden.'));

    const first = render(<StatusPage />);
    await submitForm(user, 'AUF-2025-0187', 'B-ZZ 9999');
    const wrongPlateText = emptyState()?.textContent;
    first.unmount();

    mockedApiGet.mockReset();
    mockedApiGet.mockRejectedValue(new ApiError(404, 'not_found', 'Nicht gefunden.'));
    render(<StatusPage />);
    await submitForm(user, 'AUF-9999-0000', 'B-AB 1234');
    const unknownNumberText = emptyState()?.textContent;

    expect(wrongPlateText).toBe(NEUTRAL);
    expect(unknownNumberText).toBe(wrongPlateText);
  });

  it('does not show validation feedback before the form is touched', () => {
    render(<StatusPage />);
    expect(
      screen.queryByText(/Bitte geben Sie die Auftragsnummer an\./),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/Bitte geben Sie das Kennzeichen an\./)).not.toBeInTheDocument();
  });

  it('shows validation feedback only after submitting an empty form', async () => {
    const user = userEvent.setup();
    render(<StatusPage />);

    await user.click(screen.getByRole('button', { name: 'Status abfragen' }));

    expect(screen.getByText(/Bitte geben Sie die Auftragsnummer an\./)).toBeInTheDocument();
    expect(screen.getByText(/Bitte geben Sie das Kennzeichen an\./)).toBeInTheDocument();
    expect(mockedApiGet).not.toHaveBeenCalled();
  });
});
