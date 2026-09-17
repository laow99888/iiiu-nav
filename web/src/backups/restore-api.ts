import { ApiError, request } from '../api/client';

export type RestoreErrorCode =
  | 'restore_password_invalid'
  | 'restore_archive_invalid'
  | 'restore_archive_incompatible'
  | 'restore_archive_too_large'
  | 'restore_space_insufficient'
  | 'restore_failed';

export class RestoreError extends ApiError {
  declare readonly code: RestoreErrorCode;

  constructor(code: RestoreErrorCode, status: number) {
    super(code, { code, status });
  }
}

export async function restoreBackup(file: File, password: string) {
  const body = new FormData();
  body.set('backup', file);
  body.set('password', password);
  body.set('confirmation', 'RESTORE');
  const response = await request('/api/restore', { method: 'POST', body });
  if (!response.ok) {
    const result = (await response.json().catch(() => null)) as {
      error?: RestoreErrorCode;
    } | null;
    throw new RestoreError(result?.error ?? 'restore_failed', response.status);
  }
}
