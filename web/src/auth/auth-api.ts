type AuthErrorCode =
  | 'authentication_failed'
  | 'current_password_invalid'
  | 'invalid_credentials'
  | 'invalid_password'
  | 'login_rate_limited'
  | 'logout_failed'
  | 'password_change_failed'
  | 'unknown';

export class AuthAPIError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: AuthErrorCode,
    public readonly retryAfter: number | null = null,
  ) {
    super(`Authentication request failed: ${status} (${code})`);
  }
}

export function login(password: string) {
  return authRequest('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ password }),
  });
}

export function logout() {
  return authRequest('/api/auth/logout', { method: 'POST' });
}

export function changePassword(currentPassword: string, newPassword: string) {
  return authRequest('/api/auth/password', {
    method: 'POST',
    body: JSON.stringify({ currentPassword, newPassword }),
  });
}

async function authRequest(path: string, init: RequestInit) {
  const response = await fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers: init.body
      ? { Accept: 'application/json', 'Content-Type': 'application/json' }
      : { Accept: 'application/json' },
  });
  if (response.ok) return;

  let code: AuthErrorCode = 'unknown';
  try {
    const value: unknown = await response.json();
    if (
      typeof value === 'object' &&
      value !== null &&
      typeof (value as { error?: unknown }).error === 'string'
    ) {
      code = (value as { error: AuthErrorCode }).error;
    }
  } catch {
    // Preserve the HTTP status when an upstream error page is not JSON.
  }
  const retryAfter = Number.parseInt(
    response.headers.get('Retry-After') ?? '',
    10,
  );
  throw new AuthAPIError(
    response.status,
    code,
    Number.isFinite(retryAfter) ? retryAfter : null,
  );
}
