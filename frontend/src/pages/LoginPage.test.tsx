import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { apiPost, ApiError } from '../api/client';
import { clearToken, getToken } from '../auth/session';
import { LoginPage } from './LoginPage';

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client');
  return { ...actual, apiPost: vi.fn() };
});

const apiPostMock = vi.mocked(apiPost);

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}</div>;
}

function renderLoginPage() {
  return render(
    <MemoryRouter initialEntries={['/werkstatt/login']}>
      <Routes>
        <Route path="/werkstatt/login" element={<LoginPage />} />
        <Route path="/werkstatt/auftraege" element={<div data-testid="orders-page">Aufträge</div>} />
      </Routes>
      <LocationProbe />
    </MemoryRouter>,
  );
}

async function fillAndSubmit(email: string, password: string) {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText(/E-Mail/, { selector: 'input' }), email);
  await user.type(screen.getByLabelText(/Passwort/, { selector: 'input' }), password);
  await user.click(screen.getByRole('button', { name: 'Anmelden' }));
}

beforeEach(() => {
  clearToken();
  apiPostMock.mockReset();
});

afterEach(() => {
  cleanup();
  clearToken();
});

describe('LoginPage', () => {
  it('stores the returned token and redirects to the workshop orders page on success', async () => {
    apiPostMock.mockResolvedValueOnce({ token: 'jwt-token-123', expires_at: '2026-10-10T12:00:00Z' });

    renderLoginPage();
    await fillAndSubmit('werkstatt@werkstatt-vogel.de', 'Fahrzeug#2025');

    await waitFor(() => {
      expect(screen.getByTestId('location')).toHaveTextContent('/werkstatt/auftraege');
    });
    expect(getToken()).toBe('jwt-token-123');
    expect(apiPostMock).toHaveBeenCalledWith('/api/workshop/login', {
      email: 'werkstatt@werkstatt-vogel.de',
      password: 'Fahrzeug#2025',
    });
  });

  it('shows exactly one neutral message on 401', async () => {
    apiPostMock.mockRejectedValueOnce(new ApiError(401, 'unauthorized', 'Bitte melden Sie sich an.'));

    renderLoginPage();
    await fillAndSubmit('unbekannt@example.com', 'falsch');

    expect(await screen.findByText('E-Mail oder Passwort ist falsch.')).toBeInTheDocument();
    expect(getToken()).toBeNull();
  });

  it('shows the same message for an unknown e-mail and a wrong password', async () => {
    apiPostMock.mockRejectedValueOnce(new ApiError(401, 'unauthorized', 'Bitte melden Sie sich an.'));
    const first = renderLoginPage();
    await fillAndSubmit('unbekannt@example.com', 'irgendwas');
    const firstMessage = await screen.findByText('E-Mail oder Passwort ist falsch.');
    const firstText = firstMessage.textContent;
    first.unmount();
    clearToken();

    apiPostMock.mockRejectedValueOnce(new ApiError(401, 'unauthorized', 'Bitte melden Sie sich an.'));
    renderLoginPage();
    await fillAndSubmit('werkstatt@werkstatt-vogel.de', 'falsch');
    const secondMessage = await screen.findByText('E-Mail oder Passwort ist falsch.');
    expect(secondMessage.textContent).toBe(firstText);
  });

  it('shows the rate-limit message on 429', async () => {
    apiPostMock.mockRejectedValueOnce(new ApiError(429, 'rate_limited', 'Zu viele Versuche.'));

    renderLoginPage();
    await fillAndSubmit('werkstatt@werkstatt-vogel.de', 'Fahrzeug#2025');

    expect(await screen.findByText('Zu viele Versuche. Bitte kurz warten.')).toBeInTheDocument();
  });

  it('shows no validation error on an untouched form', () => {
    renderLoginPage();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.queryByText(/Bitte geben Sie/)).not.toBeInTheDocument();
  });

  it('validates the fields only after a submit attempt', async () => {
    const user = userEvent.setup();
    renderLoginPage();
    await user.click(screen.getByRole('button', { name: 'Anmelden' }));

    expect(
      await screen.findByText(/Bitte geben Sie eine gültige E-Mail-Adresse an\./),
    ).toBeInTheDocument();
    expect(screen.getByText(/Bitte geben Sie Ihr Passwort an\./)).toBeInTheDocument();
    expect(apiPostMock).not.toHaveBeenCalled();
  });

  it('never echoes the typed credentials in an error message', async () => {
    apiPostMock.mockRejectedValueOnce(new ApiError(401, 'unauthorized', 'Bitte melden Sie sich an.'));

    renderLoginPage();
    await fillAndSubmit('geheim@example.com', 'SuperGeheim1');

    const alert = await screen.findByRole('alert');
    expect(alert.textContent).not.toContain('geheim@example.com');
    expect(alert.textContent).not.toContain('SuperGeheim1');
  });
});
