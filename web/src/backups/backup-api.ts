import { ApiError, downloadFile, isRecord, request } from '../api/client';

export type BackupInfo = {
  name: string;
  size: number;
  createdAt: string;
};

function parseBackupInfo(value: unknown): BackupInfo {
  if (!isRecord(value)) {
    throw new Error('invalid backup response');
  }
  const { name, size, createdAt } = value;
  if (
    typeof name !== 'string' ||
    name === '' ||
    typeof size !== 'number' ||
    !Number.isFinite(size) ||
    typeof createdAt !== 'string' ||
    createdAt === ''
  ) {
    throw new Error('invalid backup response');
  }
  return { name, size, createdAt };
}

export async function listBackups(): Promise<BackupInfo[]> {
  const response = await request('/api/backups');
  if (!response.ok) {
    throw new ApiError('backup list failed', { status: response.status });
  }
  const body: unknown = await response.json();
  const backups = isRecord(body) ? body.backups : undefined;
  if (!Array.isArray(backups)) throw new Error('invalid backup response');
  return backups.map(parseBackupInfo);
}

export async function createBackup(): Promise<BackupInfo> {
  const response = await request('/api/backups', { method: 'POST' });
  if (!response.ok) {
    throw new ApiError('backup creation failed', { status: response.status });
  }
  return parseBackupInfo(await response.json());
}

export async function deleteBackup(name: string) {
  const response = await request(`/api/backups/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new ApiError('backup deletion failed', { status: response.status });
  }
}

export function downloadBackup(name: string) {
  downloadFile(`/api/backups/${encodeURIComponent(name)}`, name);
}
