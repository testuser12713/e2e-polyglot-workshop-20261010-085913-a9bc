export interface ErrorBannerProps {
  message: string;
  onClose?: () => void;
}

export function ErrorBanner({ message, onClose }: ErrorBannerProps) {
  return (
    <div className="inline-alert inline-alert--error" role="alert" aria-live="assertive">
      <span className="inline-alert__message">{message}</span>
      {onClose ? (
        <button
          type="button"
          className="inline-alert__close"
          aria-label="Meldung schließen"
          onClick={onClose}
        >
          ×
        </button>
      ) : null}
    </div>
  );
}
