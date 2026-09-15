export type BackupInfo = {
  name: string;
  size: number;
  createdAt: string;
};

function parseBackupInfo(value: unknown): BackupInfo {
  if (typeof value !== 'object' || value === null) {
    throw new Error('invalid backup response');
  }
  const { name, size, createdAt } = value as Record<string, unknown>;
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
  const response = await fetch('/api/backups', { credentials: 'same-origin' });
  if (!response.ok) throw new Error('backup list failed');
  const body: unknown = await response.json();
  const backups = (body as { backups?: unknown }).backups;
  if (!Array.isArray(backups)) throw new Error('invalid backup response');
  return backups.map(parseBackupInfo);
}

export async function createBackup(): Promise<BackupInfo> {
  const response = await fetch('/api/backups', {
    method: 'POST',
    credentials: 'same-origin',
  });
  if (!response.ok) throw new Error('backup creation failed');
  return parseBackupInfo(await response.json());
}

export async function deleteBackup(name: string) {
  const response = await fetch(`/api/backups/${encodeURIComponent(name)}`, {
    method: 'DELETE',
    credentials: 'same-origin',
  });
  if (!response.ok) throw new Error('backup deletion failed');
}

export function downloadBackup(name: string) {
  const anchor = document.createElement('a');
  anchor.href = `/api/backups/${encodeURIComponent(name)}`;
  anchor.download = name;
  anchor.click();
}
