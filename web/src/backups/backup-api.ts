export type BackupInfo = {
  name: string;
  size: number;
  createdAt: string;
};

export async function listBackups(): Promise<BackupInfo[]> {
  const response = await fetch('/api/backups', { credentials: 'same-origin' });
  if (!response.ok) throw new Error('backup list failed');
  const body = (await response.json()) as { backups: BackupInfo[] };
  return body.backups;
}

export async function createBackup(): Promise<BackupInfo> {
  const response = await fetch('/api/backups', {
    method: 'POST',
    credentials: 'same-origin',
  });
  if (!response.ok) throw new Error('backup creation failed');
  return (await response.json()) as BackupInfo;
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
