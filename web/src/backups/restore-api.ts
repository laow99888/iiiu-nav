export type RestoreErrorCode =
  | 'restore_password_invalid'
  | 'restore_archive_invalid'
  | 'restore_archive_incompatible'
  | 'restore_archive_too_large'
  | 'restore_space_insufficient'
  | 'restore_failed';

export class RestoreError extends Error {
  constructor(readonly code: RestoreErrorCode) {
    super(code);
  }
}

export async function restoreBackup(file: File, password: string) {
  const body = new FormData();
  body.set('backup', file);
  body.set('password', password);
  body.set('confirmation', 'RESTORE');
  const response = await fetch('/api/restore', {
    method: 'POST',
    credentials: 'same-origin',
    body,
  });
  if (!response.ok) {
    const result = (await response.json().catch(() => null)) as {
      error?: RestoreErrorCode;
    } | null;
    throw new RestoreError(result?.error ?? 'restore_failed');
  }
}
