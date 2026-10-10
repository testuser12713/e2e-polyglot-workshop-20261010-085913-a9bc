import { getToken } from '../auth/session';

export type QueryParams = Record<string, string | number | boolean | undefined | null>;

interface ApiErrorBody {
  error?: {
    code?: string;
    message?: string;
  };
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

const rawBaseUrl = import.meta.env.VITE_API_BASE_URL;
export const API_BASE_URL = (typeof rawBaseUrl === 'string' && rawBaseUrl.length > 0
  ? rawBaseUrl
  : 'http://localhost:8000'
).replace(/\/+$/, '');

function fallbackCode(status: number): string {
  switch (status) {
    case 400:
      return 'validation_failed';
    case 401:
      return 'unauthorized';
    case 404:
      return 'not_found';
    case 409:
      return 'conflict';
    case 429:
      return 'rate_limited';
    case 501:
      return 'not_implemented';
    default:
      return 'internal_error';
  }
}

const DEFAULT_MESSAGES: Record<string, string> = {
  validation_failed: 'Die Eingaben sind ungültig.',
  unauthorized: 'Bitte melden Sie sich an.',
  not_found: 'Nicht gefunden.',
  conflict: 'Dieser Vorgang ist nicht möglich.',
  rate_limited: 'Zu viele Versuche. Bitte kurz warten.',
  not_implemented: 'Diese Funktion ist noch nicht verfügbar.',
  internal_error: 'Es ist ein Fehler aufgetreten.',
  network_error: 'Die Verbindung zum Server ist fehlgeschlagen.',
};

function buildUrl(path: string, params?: QueryParams): string {
  const url = `${API_BASE_URL}${path}`;
  if (!params) return url;
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue;
    search.append(key, String(value));
  }
  const query = search.toString();
  return query ? `${url}?${query}` : url;
}

async function request<T>(
  method: string,
  path: string,
  opts: { params?: QueryParams; body?: unknown } = {},
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  let body: string | undefined;
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(opts.body);
  }

  let response: Response;
  try {
    response = await fetch(buildUrl(path, opts.params), { method, headers, body });
  } catch {
    throw new ApiError(0, 'network_error', DEFAULT_MESSAGES.network_error);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  let data: unknown;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = undefined;
    }
  }

  if (!response.ok) {
    const error = (data as ApiErrorBody | undefined)?.error;
    const code = error?.code ?? fallbackCode(response.status);
    const message = error?.message ?? DEFAULT_MESSAGES[code] ?? DEFAULT_MESSAGES.internal_error;
    throw new ApiError(response.status, code, message);
  }

  return data as T;
}

export function apiGet<T>(path: string, params?: QueryParams): Promise<T> {
  return request<T>('GET', path, { params });
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('POST', path, { body });
}

export function apiPatch<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('PATCH', path, { body });
}

export function apiDelete<T = void>(path: string, body?: unknown): Promise<T> {
  return request<T>('DELETE', path, { body });
}
