const TOKEN_KEY = 'werkstatt.token';

function storage(): Storage | null {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

export function getToken(): string | null {
  const store = storage();
  if (!store) return null;
  return store.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  const store = storage();
  if (!store) return;
  store.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  const store = storage();
  if (!store) return;
  store.removeItem(TOKEN_KEY);
}

export function isAuthenticated(): boolean {
  return getToken() !== null;
}
