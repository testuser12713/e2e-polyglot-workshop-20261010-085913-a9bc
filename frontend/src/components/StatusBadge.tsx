import { formatStatusLabel } from '../lib/format';

export interface StatusBadgeProps {
  status: string;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span className={`status-badge status-badge--${status}`}>
      <span className="status-badge__dot" aria-hidden="true" />
      {formatStatusLabel(status)}
    </span>
  );
}
