import '@testing-library/jest-dom/vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it } from 'vitest';
import { App } from '../App';
import { DatenschutzPage } from './DatenschutzPage';
import { ImpressumPage } from './ImpressumPage';

afterEach(() => {
  cleanup();
});

const IMPRESSUM_SECTIONS = [
  'Angaben gemäß § 5 TMG',
  'Kontakt',
  'Berufsbezeichnung und Aufsicht',
  'Umsatzsteuer-Identifikationsnummer',
  'Verantwortlich für den Inhalt nach § 18 Abs. 2 MStV',
  'Verbraucherstreitbeilegung',
  'Haftung für Inhalte',
];

const DATENSCHUTZ_SECTIONS = [
  'Verantwortlicher',
  'Zweck der Verarbeitung',
  'Welche Daten wir verarbeiten',
  'Speicherdauer',
  'Kontaktweg für Fragen und zur Wahrnehmung Ihrer Rechte',
  'Keine Drittanbieter-Ressourcen vor Einwilligung',
  'Protokolle und Fehlermeldungen',
];

describe('ImpressumPage', () => {
  it('renders the legal notice with all required sections', () => {
    render(<ImpressumPage />);
    expect(screen.getByRole('heading', { level: 1, name: 'Impressum' })).toBeInTheDocument();
    for (const section of IMPRESSUM_SECTIONS) {
      expect(screen.getByRole('heading', { name: section })).toBeInTheDocument();
    }
  });
});

describe('DatenschutzPage', () => {
  it('renders the privacy statement with all required sections', () => {
    render(<DatenschutzPage />);
    expect(
      screen.getByRole('heading', { level: 1, name: 'Datenschutzerklärung' }),
    ).toBeInTheDocument();
    for (const section of DATENSCHUTZ_SECTIONS) {
      expect(screen.getByRole('heading', { name: section })).toBeInTheDocument();
    }
  });

  it('names purpose, storage duration and contact channel for the collected data', () => {
    render(<DatenschutzPage />);

    expect(screen.getByRole('heading', { name: 'Zweck der Verarbeitung' })).toBeInTheDocument();
    expect(screen.getByText(/Ihre Terminanfrage zu bearbeiten/)).toBeInTheDocument();

    expect(screen.getByRole('heading', { name: 'Speicherdauer' })).toBeInTheDocument();
    expect(screen.getByText(/handels- und steuerrechtlichen Aufbewahrungsfristen/)).toBeInTheDocument();

    expect(
      screen.getByRole('heading', { name: 'Kontaktweg für Fragen und zur Wahrnehmung Ihrer Rechte' }),
    ).toBeInTheDocument();
    expect(screen.getAllByText(/datenschutz@werkstatt-vogel\.de/).length).toBeGreaterThan(0);

    for (const data of ['Name', 'E-Mail', 'Telefon', 'Kennzeichen']) {
      expect(screen.getByText(data)).toBeInTheDocument();
    }
  });

  it('declares that no third-party resources are loaded', () => {
    render(<DatenschutzPage />);
    expect(screen.getByText(/keine Ressourcen von Drittanbietern geladen/)).toBeInTheDocument();
  });
});

describe('legal pages reachability', () => {
  it('is reachable from another page through the footer links', async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole('link', { name: 'Impressum' }));
    expect(screen.getByRole('heading', { level: 1, name: 'Impressum' })).toBeInTheDocument();

    await user.click(screen.getByRole('link', { name: 'Datenschutz' }));
    expect(
      screen.getByRole('heading', { level: 1, name: 'Datenschutzerklärung' }),
    ).toBeInTheDocument();
  });
});
