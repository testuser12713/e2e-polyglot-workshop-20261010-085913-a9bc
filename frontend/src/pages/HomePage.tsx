import { Link } from 'react-router-dom';
import './HomePage.css';

export function HomePage() {
  return (
    <div className="home">
      <section className="home__hero" data-od-id="hero">
        <p className="home__kicker">Ihr Kundenportal</p>
        <h1 className="home__title" data-od-id="hero-title">
          Werkstatttermine, Status und Rechnungen an einem Ort.
        </h1>
        <p className="home__lead">
          Fordern Sie einen Termin an, verfolgen Sie den Fortschritt Ihres Werkstattauftrags
          und sehen Sie Ihre Rechnung, sobald das Fahrzeug fertig ist.
        </p>
        <div className="home__actions">
          <Link className="btn btn--primary" to="/termin">
            Termin anfragen
          </Link>
          <Link className="btn btn--secondary" to="/status">
            Status abfragen
          </Link>
        </div>
      </section>

      <section
        aria-label="Kundenbereich"
        className="home__grid"
        data-od-id="customer-features"
      >
        <Link className="home__card" to="/termin" data-od-id="cta-termin">
          <svg
            className="home__icon"
            width="28"
            height="28"
            viewBox="0 0 24 24"
            fill="none"
            aria-hidden="true"
          >
            <rect x="3" y="5" width="18" height="16" rx="3" stroke="currentColor" strokeWidth="1.6" />
            <path
              d="M3 9h18M8 3v4M16 3v4"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
            />
          </svg>
          <h2 className="home__h3">Termin anfragen</h2>
          <p className="home__muted">
            Kunde, Fahrzeug und Wunschtermin erfassen — Sie erhalten sofort eine Auftragsnummer.
          </p>
          <span className="home__link" data-od-id="feature-termin">
            Termin anfragen
          </span>
        </Link>

        <Link className="home__card" to="/status" data-od-id="cta-status">
          <svg
            className="home__icon"
            width="28"
            height="28"
            viewBox="0 0 24 24"
            fill="none"
            aria-hidden="true"
          >
            <circle cx="11" cy="11" r="7" stroke="currentColor" strokeWidth="1.6" />
            <path d="M16 16l4 4" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
          </svg>
          <h2 className="home__h3">Status abfragen</h2>
          <p className="home__muted">
            Auftragsnummer und Kennzeichen eingeben und den vollständigen Statusverlauf sehen.
          </p>
          <span className="home__link" data-od-id="feature-status">
            Status abfragen
          </span>
        </Link>

        <Link className="home__card" to="/rechnung" data-od-id="cta-rechnung">
          <svg
            className="home__icon"
            width="28"
            height="28"
            viewBox="0 0 24 24"
            fill="none"
            aria-hidden="true"
          >
            <path
              d="M6 3h9l3 3v15H6z"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinejoin="round"
            />
            <path
              d="M9 8h6M9 12h6M9 16h4"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
            />
          </svg>
          <h2 className="home__h3">Rechnung ansehen</h2>
          <p className="home__muted">
            Nach dem Status „fertig“ stehen Positionen, Zwischensumme, MwSt. und Gesamtbetrag
            bereit.
          </p>
          <span className="home__link" data-od-id="feature-rechnung">
            Rechnung ansehen
          </span>
        </Link>
      </section>

      <section className="home__card home__workshop" data-od-id="workshop-entry">
        <div className="home__row-between">
          <div>
            <h2 className="home__h3">Werkstattbereich</h2>
            <p className="home__muted">
              Für Mitarbeitende: Auftragsliste, Positionen, Statusübergänge und Dashboard.
              Anmeldung mit E-Mail und Passwort.
            </p>
          </div>
          <Link className="btn btn--secondary" to="/werkstatt/login" data-od-id="cta-login">
            Anmelden
          </Link>
        </div>
      </section>
    </div>
  );
}
