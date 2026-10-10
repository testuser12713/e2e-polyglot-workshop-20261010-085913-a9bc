import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { ApiError, apiPost } from '../api/client';
import { setToken } from '../auth/session';
import { Button } from '../components/Button';
import { ErrorBanner } from '../components/ErrorBanner';
import { Field } from '../components/Field';

const LOGIN_PATH = '/api/workshop/login';
const EMAIL_PATTERN = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;
const INVALID_CREDENTIALS = 'E-Mail oder Passwort ist falsch.';
const RATE_LIMITED = 'Zu viele Versuche. Bitte kurz warten.';
const GENERIC_ERROR = 'Es ist ein Fehler aufgetreten.';

interface LoginResponse {
  token: string;
  expires_at?: string;
}

interface TouchedFields {
  email: boolean;
  password: boolean;
}

export function LoginPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [touched, setTouched] = useState<TouchedFields>({ email: false, password: false });
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const emailError =
    (touched.email || submitted) && !EMAIL_PATTERN.test(email.trim())
      ? 'Bitte geben Sie eine gültige E-Mail-Adresse an.'
      : null;
  const passwordError =
    (touched.password || submitted) && password.length === 0
      ? 'Bitte geben Sie Ihr Passwort an.'
      : null;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);
    setFormError(null);

    if (!EMAIL_PATTERN.test(email.trim()) || password.length === 0) {
      return;
    }

    setSubmitting(true);
    try {
      const result = await apiPost<LoginResponse>(LOGIN_PATH, {
        email: email.trim(),
        password,
      });
      setToken(result.token);
      navigate('/werkstatt/auftraege', { replace: true });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setFormError(INVALID_CREDENTIALS);
      } else if (error instanceof ApiError && error.status === 429) {
        setFormError(RATE_LIMITED);
      } else if (error instanceof ApiError) {
        setFormError(error.message);
      } else {
        setFormError(GENERIC_ERROR);
      }
      setSubmitting(false);
    }
  }

  return (
    <section className="page">
      <div className="login-wrap">
        <header className="login-head">
          <h1 className="page__title">Anmeldung</h1>
          <p className="login-head__sub">
            Zugang zum Werkstattbereich mit Ihrer geschäftlichen E-Mail.
          </p>
        </header>

        <section className="card" data-testid="login-card">
          {formError ? <ErrorBanner message={formError} /> : null}

          <form className="login-form" onSubmit={handleSubmit} noValidate>
            <Field
              label="E-Mail"
              required
              type="email"
              id="login-email"
              name="email"
              autoComplete="username"
              value={email}
              error={emailError}
              helper="z. B. werkstatt@werkstatt-vogel.de"
              onChange={(event) => setEmail(event.target.value)}
              onBlur={() => setTouched((current) => ({ ...current, email: true }))}
            />

            <Field
              label="Passwort"
              required
              type="password"
              id="login-password"
              name="password"
              autoComplete="current-password"
              value={password}
              error={passwordError}
              onChange={(event) => setPassword(event.target.value)}
              onBlur={() => setTouched((current) => ({ ...current, password: true }))}
            />

            <Button
              type="submit"
              variant="primary"
              block
              loading={submitting}
              className="login-form__submit"
              data-testid="submit-login"
            >
              Anmelden
            </Button>
          </form>
        </section>

        <p className="login-note">
          Die Anmelderoute begrenzt Fehlversuche pro Client und antwortet nicht unterschiedlich für
          unbekannte E-Mail und falsches Passwort.
        </p>
      </div>
    </section>
  );
}
