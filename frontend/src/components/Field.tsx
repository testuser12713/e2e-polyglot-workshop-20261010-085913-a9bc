import { useId, useState } from 'react';
import type {
  InputHTMLAttributes,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from 'react';

interface FieldBaseProps {
  label: string;
  required?: boolean;
  error?: string | null;
  helper?: string;
  id?: string;
}

export interface FieldProps
  extends FieldBaseProps,
    Omit<InputHTMLAttributes<HTMLInputElement>, 'id'> {}

export interface TextAreaFieldProps
  extends FieldBaseProps,
    Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'id'> {}

export interface SelectFieldProps
  extends FieldBaseProps,
    Omit<SelectHTMLAttributes<HTMLSelectElement>, 'id'> {}

function helpers(fieldId: string, error: string | null | undefined, helper?: string) {
  const errorId = `${fieldId}-error`;
  const helperId = `${fieldId}-helper`;
  return {
    describedBy: error ? errorId : helper ? helperId : undefined,
    errorId,
    helperId,
  };
}

function FieldFeedback({
  error,
  helper,
  errorId,
  helperId,
}: {
  error: string | null | undefined;
  helper?: string;
  errorId: string;
  helperId: string;
}) {
  if (error) {
    return (
      <p className="field__error" id={errorId}>
        ! {error}
      </p>
    );
  }
  if (helper) {
    return (
      <p className="field__helper" id={helperId}>
        {helper}
      </p>
    );
  }
  return null;
}

export function Field({
  label,
  required,
  error,
  helper,
  id,
  type = 'text',
  className,
  ...rest
}: FieldProps) {
  const generatedId = useId();
  const fieldId = id ?? generatedId;
  const [showPassword, setShowPassword] = useState(false);
  const isPassword = type === 'password';
  const inputType = isPassword && showPassword ? 'text' : type;
  const { describedBy, errorId, helperId } = helpers(fieldId, error, helper);

  return (
    <div className={`field${error ? ' field--error' : ''}`}>
      <label className="field__label" htmlFor={fieldId}>
        {label}
        {required ? (
          <span className="field__required" aria-hidden="true">
            {' '}
            *
          </span>
        ) : null}
      </label>
      <div className="field__input-wrap">
        <input
          {...rest}
          id={fieldId}
          className={['field__control', className].filter(Boolean).join(' ')}
          type={inputType}
          required={required}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
        />
        {isPassword ? (
          <button
            type="button"
            className="field__password-toggle"
            onClick={() => setShowPassword((value) => !value)}
            aria-label={showPassword ? 'Passwort verbergen' : 'Passwort anzeigen'}
          >
            {showPassword ? 'Verbergen' : 'Anzeigen'}
          </button>
        ) : null}
      </div>
      <FieldFeedback error={error} helper={helper} errorId={errorId} helperId={helperId} />
    </div>
  );
}

export function TextAreaField({
  label,
  required,
  error,
  helper,
  id,
  className,
  ...rest
}: TextAreaFieldProps) {
  const generatedId = useId();
  const fieldId = id ?? generatedId;
  const { describedBy, errorId, helperId } = helpers(fieldId, error, helper);

  return (
    <div className={`field${error ? ' field--error' : ''}`}>
      <label className="field__label" htmlFor={fieldId}>
        {label}
        {required ? (
          <span className="field__required" aria-hidden="true">
            {' '}
            *
          </span>
        ) : null}
      </label>
      <textarea
        {...rest}
        id={fieldId}
        className={['field__control', 'field__control--textarea', className]
          .filter(Boolean)
          .join(' ')}
        required={required}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
      />
      <FieldFeedback error={error} helper={helper} errorId={errorId} helperId={helperId} />
    </div>
  );
}

export function SelectField({
  label,
  required,
  error,
  helper,
  id,
  className,
  children,
  ...rest
}: SelectFieldProps) {
  const generatedId = useId();
  const fieldId = id ?? generatedId;
  const { describedBy, errorId, helperId } = helpers(fieldId, error, helper);

  return (
    <div className={`field${error ? ' field--error' : ''}`}>
      <label className="field__label" htmlFor={fieldId}>
        {label}
        {required ? (
          <span className="field__required" aria-hidden="true">
            {' '}
            *
          </span>
        ) : null}
      </label>
      <div className="field__input-wrap">
        <select
          {...rest}
          id={fieldId}
          className={['field__control', 'field__control--select', className]
            .filter(Boolean)
            .join(' ')}
          required={required}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
        >
          {children}
        </select>
      </div>
      <FieldFeedback error={error} helper={helper} errorId={errorId} helperId={helperId} />
    </div>
  );
}
