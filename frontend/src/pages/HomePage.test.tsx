import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { afterEach, describe, expect, it } from 'vitest';
import { App } from '../App';
import { HomePage } from './HomePage';

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}</div>;
}

afterEach(() => {
  cleanup();
});

describe('HomePage', () => {
  it('renders the hero headline', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <HomePage />
      </MemoryRouter>,
    );

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'Werkstatttermine, Status und Rechnungen an einem Ort.',
    );
  });

  it('renders the three entry cards in the customer feature block', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <HomePage />
      </MemoryRouter>,
    );

    const features = screen.getByRole('region', { name: 'Kundenbereich' });
    for (const label of ['Termin anfragen', 'Status abfragen', 'Rechnung ansehen']) {
      expect(within(features).getByRole('link', { name: new RegExp(label) })).toBeInTheDocument();
    }
  });

  it('links the workshop entry to the workshop login', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <HomePage />
      </MemoryRouter>,
    );

    expect(screen.getByRole('link', { name: 'Anmelden' })).toHaveAttribute(
      'href',
      '/werkstatt/login',
    );
  });

  it('navigates each entry card to its flow', async () => {
    const user = userEvent.setup();
    const flows: Array<[string, string]> = [
      ['Termin anfragen', '/termin'],
      ['Status abfragen', '/status'],
      ['Rechnung ansehen', '/rechnung'],
    ];

    for (const [label, path] of flows) {
      const { unmount } = render(
        <MemoryRouter initialEntries={['/']}>
          <App />
          <LocationProbe />
        </MemoryRouter>,
      );

      const features = screen.getByRole('region', { name: 'Kundenbereich' });
      await user.click(within(features).getByRole('link', { name: new RegExp(label) }));
      expect(screen.getByTestId('location')).toHaveTextContent(path);

      unmount();
    }
  });
});
