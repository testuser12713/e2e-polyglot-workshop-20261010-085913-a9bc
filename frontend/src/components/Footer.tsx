import { Link } from 'react-router-dom';

export function Footer() {
  return (
    <footer className="footer">
      <div className="container footer__inner">
        <span className="footer__name">Kfz-Werkstatt Kundenportal</span>
        <nav className="footer__links" aria-label="Rechtliches">
          <Link to="/impressum" className="footer__link">
            Impressum
          </Link>
          <Link to="/datenschutz" className="footer__link">
            Datenschutz
          </Link>
        </nav>
      </div>
    </footer>
  );
}
