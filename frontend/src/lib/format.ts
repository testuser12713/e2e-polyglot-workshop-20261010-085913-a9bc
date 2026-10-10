export type OrderStatus =
  | 'angefragt'
  | 'bestaetigt'
  | 'in_arbeit'
  | 'fertig'
  | 'abgeholt';

export const STATUS_LABELS: Record<OrderStatus, string> = {
  angefragt: 'angefragt',
  bestaetigt: 'bestätigt',
  in_arbeit: 'in Arbeit',
  fertig: 'fertig',
  abgeholt: 'abgeholt',
};

export const ORDER_STATUSES: OrderStatus[] = [
  'angefragt',
  'bestaetigt',
  'in_arbeit',
  'fertig',
  'abgeholt',
];

export function isOrderStatus(value: string): value is OrderStatus {
  return (ORDER_STATUSES as string[]).includes(value);
}

export function formatStatusLabel(status: string): string {
  return isOrderStatus(status) ? STATUS_LABELS[status] : status;
}

export function formatMoney(cents: number): string {
  const value = Number.isFinite(cents) ? Math.round(cents) : 0;
  const negative = value < 0;
  const absolute = Math.abs(value);
  const euros = Math.floor(absolute / 100)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, '.');
  const decimals = (absolute % 100).toString().padStart(2, '0');
  return `${negative ? '\u2212' : ''}${euros},${decimals} \u20ac`;
}
