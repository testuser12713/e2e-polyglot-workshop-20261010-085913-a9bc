import { useState } from 'react';
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom';
import { clearToken, getToken } from '../auth/session';

interface NavItem {
  to: string;
  label: string;
  end?: boolean;
}

const PUBLIC_LINKS: NavItem[] = [
  { to: '/', label: 'Termin', end: true },
  { to: '/status', label: 'Status' },
  { to: '/rechnung', label: 'Rechnung' },
];

const WORKSHOP_LINKS: NavItem[] = [
  { to: '/werkstatt/dashboard', label: 'Dashboard' },
  { to: '/werkstatt/auftraege', label: 'Aufträge' },
];

export function NavBar() {
  const location = useLocation();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const isWorkshop = location.pathname.startsWith('/werkstatt');
  const links = isWorkshop ? WORKSHOP_LINKS : PUBLIC_LINKS;
  const signedIn = getToken() !== null;

  function handleLogout() {
    clearToken();
    setMenuOpen(false);
    navigate('/werkstatt/login');
  }

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `appbar__link${isActive ? ' appbar__link--active' : ''}`;
  const sheetLinkClass = ({ isActive }: { isActive: boolean }) =>
    `appbar__sheet-link${isActive ? ' appbar__sheet-link--active' : ''}`;

  return (
    <header className="appbar">
      <div className="container appbar__inner">
        <Link to="/" className="appbar__brand" onClick={() => setMenuOpen(false)}>
          Kfz-Werkstatt
        </Link>
        <nav className="appbar__links" aria-label="Hauptnavigation">
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={linkClass}
              onClick={() => setMenuOpen(false)}
            >
              {link.label}
            </NavLink>
          ))}
          {isWorkshop && signedIn ? (
            <button type="button" className="appbar__logout" onClick={handleLogout}>
              Abmelden
            </button>
          ) : null}
        </nav>
        <button
          type="button"
          className="appbar__menu-btn"
          aria-label="Menü umschalten"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((open) => !open)}
        >
          {menuOpen ? 'Schließen' : 'Menü'}
        </button>
      </div>
      {menuOpen ? (
        <nav className="appbar__sheet" aria-label="Hauptnavigation mobil">
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={sheetLinkClass}
              onClick={() => setMenuOpen(false)}
            >
              {link.label}
            </NavLink>
          ))}
          {isWorkshop && signedIn ? (
            <button type="button" className="appbar__sheet-link" onClick={handleLogout}>
              Abmelden
            </button>
          ) : null}
        </nav>
      ) : null}
    </header>
  );
}
