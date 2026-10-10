import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <section className="page">
      <h1 className="page__title" data-testid="not-found">
        Seite nicht gefunden
      </h1>
      <p className="page__text">
        Die aufgerufene Seite existiert nicht. Zurück zur{' '}
        <Link to="/">Terminanfrage</Link>.
      </p>
    </section>
  );
}
