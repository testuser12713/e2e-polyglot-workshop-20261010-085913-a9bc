import '@testing-library/jest-dom/vitest';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiPost: vi.fn(),
  };
});

import type { Mock } from 'vitest';
import { ApiError, apiPost } from '../api/client';
import { AppointmentPage } from './AppointmentPage';

const ORDER_NUMBER = 'AUF-2025-0201';

function renderPage() {
  return render(
    <MemoryRouter>
      <AppointmentPage />
    </MemoryRouter>,
  );
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/Name/), 'Anna Schmidt');
  await user.type(screen.getByLabelText(/E-Mail/), 'anna.schmidt@example.de');
  await user.type(screen.getByLabelText(/Telefon/), '0151 23456789');
  await user.type(screen.getByLabelText(/Kennzeichen/), 'B-AB 1234');
  await user.type(screen.getByLabelText(/Kilometerstand/), '128450');
  fireEvent.change(screen.getByLabelText(/Wunschtermin/), { target: { value: '2025-03-14' } });
  await user.type(screen.getByLabelText(/Problembeschreibung/), 'Bremsen quietschen vorne.');
}

beforeEach(() => {
  vi.resetAllMocks();
});

afterEach(() => {
  cleanup();
});

describe('AppointmentPage', () => {
  it('starts neutral: no validation feedback before any input (AC-25)', () => {
    renderPage();

    expect(screen.queryByText(/Bitte geben Sie Ihren Namen an/)).not.toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(document.querySelectorAll('[aria-invalid="true"]')).toHaveLength(0);
  });

  it('reveals a field error only after that field was typed into (AC-25)', async () => {
    const user = userEvent.setup();
    renderPage();

    const email = screen.getByLabelText(/E-Mail/);
    await user.type(email, 'keine-adresse');
    await user.tab();

    expect(await screen.findByText(/gültige E-Mail-Adresse/)).toBeInTheDocument();
    expect(email).toHaveAttribute('aria-invalid', 'true');
  });

  it('submits the appointment and shows the returned order number and status', async () => {
    (apiPost as Mock).mockResolvedValueOnce({ order_number: ORDER_NUMBER, status: 'angefragt' });
    const user = userEvent.setup();
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: 'Termin anfragen' }));

    await waitFor(() =>
      expect(apiPost).toHaveBeenCalledWith('/api/public/appointments', {
        customer: {
          name: 'Anna Schmidt',
          email: 'anna.schmidt@example.de',
          phone: '0151 23456789',
        },
        vehicle: { plate: 'B-AB 1234', brand: '', model: '', mileage: 128450 },
        order: { preferred_date: '2025-03-14', description: 'Bremsen quietschen vorne.' },
      }),
    );

    const success = await screen.findByTestId('termin-success');
    expect(within(success).getByText(ORDER_NUMBER)).toBeInTheDocument();
    expect(within(screen.getByTestId('success-status')).getByText('angefragt')).toBeInTheDocument();
    expect(screen.queryByTestId('termin-form')).not.toBeInTheDocument();
  });

  it('blocks submit until the required fields are filled (AC-25)', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole('button', { name: 'Termin anfragen' }));

    expect(apiPost).not.toHaveBeenCalled();
    expect(await screen.findByText(/Bitte geben Sie Ihren Namen an/)).toBeInTheDocument();
    expect(screen.getByTestId('termin-form')).toBeInTheDocument();
  });

  it('maps a 409 conflict to a comprehensible message about the plate', async () => {
    (apiPost as Mock).mockRejectedValueOnce(
      new ApiError(409, 'conflict', 'a vehicle with this plate is already registered'),
    );
    const user = userEvent.setup();
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: 'Termin anfragen' }));

    expect(await screen.findByText(/bereits registriert/)).toBeInTheDocument();
    expect(screen.queryByTestId('termin-success')).not.toBeInTheDocument();
  });

  it('maps a 400 response to the matching field-level message', async () => {
    (apiPost as Mock).mockRejectedValueOnce(
      new ApiError(400, 'validation_failed', 'customer.email is not a valid address'),
    );
    const user = userEvent.setup();
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: 'Termin anfragen' }));

    const email = await screen.findByLabelText(/E-Mail/);
    expect(email).toHaveAttribute('aria-invalid', 'true');
    expect(await screen.findByText(/gültige E-Mail-Adresse/)).toBeInTheDocument();
  });

  it('maps a missing preferred date returned by the API to the date field', async () => {
    (apiPost as Mock).mockRejectedValueOnce(
      new ApiError(400, 'validation_failed', 'order.preferred_date is required'),
    );
    const user = userEvent.setup();
    renderPage();

    await fillValidForm(user);
    const date = screen.getByLabelText(/Wunschtermin/);
    fireEvent.change(date, { target: { value: '' } });
    await user.click(screen.getByRole('button', { name: 'Termin anfragen' }));

    expect(await screen.findByText(/Wunschtermin an/)).toBeInTheDocument();
    expect(date).toHaveAttribute('aria-invalid', 'true');
  });
});
