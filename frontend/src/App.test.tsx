import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { App } from './App';
import { clearToken, setToken } from './auth/session';

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}</div>;
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <App />
      <LocationProbe />
    </MemoryRouter>,
  );
}

const PUBLIC_ROUTES = ['/', '/status', '/rechnung', '/impressum', '/datenschutz', '/werkstatt/login'];
const PROTECTED_ROUTES = [
  '/werkstatt/auftraege',
  '/werkstatt/auftraege/AUF-2025-0187',
  '/werkstatt/dashboard',
];

beforeEach(() => {
  clearToken();
});

afterEach(() => {
  cleanup();
  clearToken();
});

describe('App shell', () => {
  it('renders the public navigation', () => {
    renderAt('/');
    expect(screen.getByRole('link', { name: 'Termin' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Status' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Rechnung' })).toBeInTheDocument();
  });

  it('carries the Impressum and Datenschutz footer links on every route', () => {
    for (const path of [...PUBLIC_ROUTES, ...PROTECTED_ROUTES]) {
      setToken('test-token');
      const { unmount } = renderAt(path);
      expect(screen.getByRole('link', { name: 'Impressum' })).toHaveAttribute('href', '/impressum');
      expect(screen.getByRole('link', { name: 'Datenschutz' })).toHaveAttribute(
        'href',
        '/datenschutz',
      );
      unmount();
      clearToken();
    }
  });

  it('resolves every public route without falling through to the not-found page', () => {
    for (const path of PUBLIC_ROUTES) {
      const { unmount } = renderAt(path);
      expect(screen.queryByTestId('not-found')).not.toBeInTheDocument();
      unmount();
    }
  });

  it('redirects protected routes to the workshop login when no token is stored', () => {
    for (const path of PROTECTED_ROUTES) {
      const { unmount } = renderAt(path);
      expect(screen.getByTestId('location')).toHaveTextContent('/werkstatt/login');
      expect(screen.queryByTestId('not-found')).not.toBeInTheDocument();
      unmount();
    }
  });

  it('resolves protected routes when a token is stored', () => {
    for (const path of PROTECTED_ROUTES) {
      setToken('test-token');
      const { unmount } = renderAt(path);
      expect(screen.getByTestId('location')).toHaveTextContent(path);
      expect(screen.queryByTestId('not-found')).not.toBeInTheDocument();
      unmount();
      clearToken();
    }
  });

  it('renders the not-found page for an unknown route', () => {
    renderAt('/diese-seite-gibt-es-nicht');
    expect(screen.getByTestId('not-found')).toBeInTheDocument();
  });

  it('navigates between public pages through the navigation', async () => {
    const user = userEvent.setup();
    renderAt('/');
    await user.click(screen.getByRole('link', { name: 'Status' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/status');
    await user.click(screen.getByRole('link', { name: 'Rechnung' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/rechnung');
  });
});
