import { ApiError, isRecord, request } from '../api/client';

type AuthErrorCode =
  | 'authentication_failed'
  | 'current_password_invalid'
  | 'invalid_credentials'
  | 'invalid_password'
  | 'login_rate_limited'
  | 'logout_failed'
  | 'password_change_failed'
  | 'unknown';

export class AuthAPIError extends ApiError {
  declare readonly code: AuthErrorCode;

  constructor(
    status: number,
    code: AuthErrorCode,
    retryAfter: number | null = null,
  ) {
    super(`Authentication request failed: ${status} (${code})`, {
      code,
      retryAfter,
      status,
    });
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
  const response = await request(path, init);
  if (response.ok) return;

  let code: AuthErrorCode = 'unknown';
  try {
    const value: unknown = await response.json();
    if (isRecord(value) && typeof value.error === 'string') {
      code = value.error as AuthErrorCode;
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
