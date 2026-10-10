import { useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { ApiError, apiPost } from '../api/client';
import { Button } from '../components/Button';
import { ErrorBanner } from '../components/ErrorBanner';
import { Field, TextAreaField } from '../components/Field';
import { StatusBadge } from '../components/StatusBadge';

// Endpoint fixed by the sprint's shared interface.
const APPOINTMENT_PATH = '/api/public/appointments';

type FieldName =
  | 'name'
  | 'email'
  | 'phone'
  | 'plate'
  | 'brand'
  | 'model'
  | 'mileage'
  | 'preferred_date'
  | 'description';

type FormValues = Record<FieldName, string>;

interface AppointmentResult {
  orderNumber: string;
  status: string;
  name: string;
}

interface AppointmentResponse {
  order_number: string;
  status: string;
}

const EMPTY_VALUES: FormValues = {
  name: '',
  email: '',
  phone: '',
  plate: '',
  brand: '',
  model: '',
  mileage: '',
  preferred_date: '',
  description: '',
};

// Order used to focus the first invalid field after a submit.
const FIELD_ORDER: FieldName[] = [
  'name',
  'email',
  'phone',
  'plate',
  'mileage',
  'preferred_date',
  'description',
];

const EMAIL_PATTERN = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;
const MILEAGE_PATTERN = /^\d+$/;

// German field-level messages, reused for both client validation and the mapped
// 400 response (the API names the field, never the value — AC-35).
const FIELD_MESSAGES: Partial<Record<FieldName, string>> = {
  name: 'Bitte geben Sie Ihren Namen an.',
  email: 'Bitte geben Sie eine gültige E-Mail-Adresse an.',
  phone: 'Bitte geben Sie Ihre Telefonnummer an.',
  plate: 'Bitte geben Sie das Kennzeichen an.',
  mileage: 'Bitte geben Sie den Kilometerstand als ganze Zahl an.',
  preferred_date: 'Bitte wählen Sie einen Wunschtermin an.',
  description: 'Bitte beschreiben Sie Ihr Anliegen kurz.',
};

// The API's validation message starts with the dotted field path, e.g.
// "customer.email is required". Map it back to the form field.
const SERVER_FIELD_PREFIXES: Array<[string, FieldName]> = [
  ['customer.name', 'name'],
  ['customer.email', 'email'],
  ['customer.phone', 'phone'],
  ['vehicle.plate', 'plate'],
  ['vehicle.brand', 'brand'],
  ['vehicle.model', 'model'],
  ['vehicle.mileage', 'mileage'],
  ['order.preferred_date', 'preferred_date'],
  ['order.description', 'description'],
];

const PLATE_ALREADY_REGISTERED =
  'Ein Fahrzeug mit diesem Kennzeichen ist bereits registriert. Bitte kontaktieren Sie uns telefonisch.';

function validate(values: FormValues): Partial<Record<FieldName, string>> {
  const errors: Partial<Record<FieldName, string>> = {};
  if (values.name.trim().length < 2) {
    errors.name = FIELD_MESSAGES.name;
  }
  if (!EMAIL_PATTERN.test(values.email.trim())) {
    errors.email = FIELD_MESSAGES.email;
  }
  if (values.phone.replace(/\D/g, '').length < 6) {
    errors.phone = FIELD_MESSAGES.phone;
  }
  if (values.plate.trim().length < 3) {
    errors.plate = FIELD_MESSAGES.plate;
  }
  if (values.mileage.trim() !== '' && !MILEAGE_PATTERN.test(values.mileage.trim())) {
    errors.mileage = FIELD_MESSAGES.mileage;
  }
  if (values.description.trim().length < 5) {
    errors.description = FIELD_MESSAGES.description;
  }
  return errors;
}

function fieldForServerMessage(message: string): FieldName | null {
  for (const [prefix, field] of SERVER_FIELD_PREFIXES) {
    if (message.startsWith(prefix)) return field;
  }
  return null;
}

function parseMileage(value: string): number {
  const trimmed = value.trim();
  if (trimmed === '') return 0;
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  return 'Es ist ein Fehler aufgetreten.';
}

export function AppointmentPage() {
  const [values, setValues] = useState<FormValues>(EMPTY_VALUES);
  const [touched, setTouched] = useState<Partial<Record<FieldName, boolean>>>({});
  const [submitted, setSubmitted] = useState(false);
  const [serverErrors, setServerErrors] = useState<Partial<Record<FieldName, string>>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<AppointmentResult | null>(null);

  const validationErrors = validate(values);

  function setField(field: FieldName, value: string) {
    setValues((previous) => ({ ...previous, [field]: value }));
    setServerErrors((previous) => {
      if (!previous[field]) return previous;
      const next = { ...previous };
      delete next[field];
      return next;
    });
  }

  function handleChange(field: FieldName) {
    return (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      const raw = event.target.value;
      setField(field, field === 'plate' ? raw.toUpperCase() : raw);
    };
  }

  function handleBlur(field: FieldName) {
    return () => {
      // AC-25: an empty field stays neutral until the form is submitted.
      if (values[field].trim() !== '') {
        setTouched((previous) => ({ ...previous, [field]: true }));
      }
    };
  }

  // AC-25: feedback only after a field was typed into and left, or after submit.
  function errorFor(field: FieldName): string | undefined {
    if (!touched[field] && !submitted) return undefined;
    return serverErrors[field] ?? validationErrors[field];
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);
    setFormError(null);

    const errors = validate(values);
    if (Object.keys(errors).length > 0) {
      const first = FIELD_ORDER.find((field) => errors[field]);
      if (first) document.getElementById(first)?.focus();
      return;
    }

    setSubmitting(true);
    try {
      const response = await apiPost<AppointmentResponse>(APPOINTMENT_PATH, {
        customer: {
          name: values.name.trim(),
          email: values.email.trim(),
          phone: values.phone.trim(),
        },
        vehicle: {
          plate: values.plate.trim().toUpperCase(),
          brand: values.brand.trim(),
          model: values.model.trim(),
          mileage: parseMileage(values.mileage),
        },
        order: {
          preferred_date: values.preferred_date.trim(),
          description: values.description.trim(),
        },
      });
      setResult({
        orderNumber: response.order_number,
        status: response.status,
        name: values.name.trim(),
      });
    } catch (error) {
      if (error instanceof ApiError && error.status === 400) {
        const field = fieldForServerMessage(error.message);
        if (field) {
          const message = FIELD_MESSAGES[field] ?? 'Bitte prüfen Sie Ihre Eingabe.';
          setServerErrors((previous) => ({ ...previous, [field]: message }));
        } else {
          setFormError('Bitte prüfen Sie Ihre Eingaben.');
        }
      } else if (error instanceof ApiError && error.status === 409) {
        setFormError(PLATE_ALREADY_REGISTERED);
      } else {
        setFormError(errorMessage(error));
      }
    } finally {
      setSubmitting(false);
    }
  }

  function startNewRequest() {
    setValues(EMPTY_VALUES);
    setTouched({});
    setSubmitted(false);
    setServerErrors({});
    setFormError(null);
    setResult(null);
  }

  return (
    <section className="page" data-testid="appointment-page">
      <header className="row-between" data-od-id="page-head">
        <div>
          <h1 className="page__title">Terminanfrage</h1>
          <p className="page__text">
            Erfassen Sie Ihre Kontaktdaten und Ihr Fahrzeug. Wir melden uns mit einem passenden
            Termin.
          </p>
        </div>
        <Link className="btn btn--ghost" to="/">
          Zurück zur Startseite
        </Link>
      </header>

      {result ? (
        <section
          className="card"
          id="termin-success"
          data-od-id="termin-success"
          data-testid="termin-success"
          aria-live="polite"
        >
          <div className="card__header">
            <h2 className="h3">Terminanfrage eingegangen</h2>
            <span data-od-id="success-status" data-testid="success-status">
              <StatusBadge status={result.status} />
            </span>
          </div>
          <p className="muted">
            Vielen Dank, {result.name}. Ihre Terminanfrage wurde mit folgender Auftragsnummer
            angelegt:
          </p>
          <p
            className="h2 mono tnum"
            id="termin-success-number"
            data-testid="success-order-number"
          >
            {result.orderNumber}
          </p>
          <div className="form-actions">
            <Link className="btn btn--primary" to="/status">
              Status abfragen
            </Link>
            <Button variant="secondary" onClick={startNewRequest}>
              Neue Anfrage
            </Button>
          </div>
        </section>
      ) : (
        <form
          className="card form-col"
          id="termin-form"
          data-od-id="termin-form"
          data-testid="termin-form"
          noValidate
          onSubmit={handleSubmit}
        >
          {formError ? (
            <ErrorBanner message={formError} onClose={() => setFormError(null)} />
          ) : null}

          <fieldset className="form-fieldset">
            <legend className="h3">Ihre Kontaktdaten</legend>

            <Field
              id="name"
              label="Name"
              required
              autoComplete="name"
              value={values.name}
              onChange={handleChange('name')}
              onBlur={handleBlur('name')}
              error={errorFor('name')}
              helper="Vor- und Nachname."
            />

            <div className="field-2">
              <Field
                id="email"
                label="E-Mail"
                type="email"
                required
                autoComplete="email"
                value={values.email}
                onChange={handleChange('email')}
                onBlur={handleBlur('email')}
                error={errorFor('email')}
                helper="z. B. name@beispiel.de"
              />
              <Field
                id="phone"
                label="Telefon"
                type="tel"
                required
                autoComplete="tel"
                value={values.phone}
                onChange={handleChange('phone')}
                onBlur={handleBlur('phone')}
                error={errorFor('phone')}
                helper="z. B. 0151 23456789"
              />
            </div>
          </fieldset>

          <fieldset className="form-fieldset">
            <legend className="h3">Ihr Fahrzeug</legend>

            <Field
              id="plate"
              label="Kennzeichen"
              required
              autoCapitalize="characters"
              value={values.plate}
              onChange={handleChange('plate')}
              onBlur={handleBlur('plate')}
              error={errorFor('plate')}
              helper="z. B. B-AB 1234"
            />

            <div className="field-2">
              <Field
                id="brand"
                label="Marke"
                value={values.brand}
                onChange={handleChange('brand')}
              />
              <Field
                id="model"
                label="Modell"
                value={values.model}
                onChange={handleChange('model')}
              />
            </div>

            <Field
              id="mileage"
              label="Kilometerstand"
              type="number"
              inputMode="numeric"
              min="0"
              className="tnum"
              value={values.mileage}
              onChange={handleChange('mileage')}
              onBlur={handleBlur('mileage')}
              error={errorFor('mileage')}
              helper="Ganze Kilometer, z. B. 128450."
            />
          </fieldset>

          <fieldset className="form-fieldset">
            <legend className="h3">Ihr Anliegen</legend>

            <Field
              id="preferred_date"
              label="Wunschtermin"
              type="date"
              className="tnum"
              value={values.preferred_date}
              onChange={handleChange('preferred_date')}
              onBlur={handleBlur('preferred_date')}
              error={errorFor('preferred_date')}
              helper="Format TT.MM.JJJJ — wir bestätigen den Termin per E-Mail."
            />

            <TextAreaField
              id="description"
              label="Problembeschreibung"
              required
              value={values.description}
              onChange={handleChange('description')}
              onBlur={handleBlur('description')}
              error={errorFor('description')}
              helper="Beschreiben Sie das Problem oder den gewünschten Service."
            />
          </fieldset>

          <p className="caption muted">
            Mit dem Absenden werden Name, E-Mail, Telefon und Kennzeichen zur Bearbeitung Ihrer
            Anfrage gespeichert. Details in der <Link to="/datenschutz">Datenschutzerklärung</Link>.
          </p>

          <div className="form-actions">
            <Button
              type="submit"
              variant="primary"
              loading={submitting}
              data-od-id="submit-termin"
            >
              Termin anfragen
            </Button>
            <Link className="btn btn--secondary" to="/">
              Abbrechen
            </Link>
          </div>
        </form>
      )}
    </section>
  );
}
