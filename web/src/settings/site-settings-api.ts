import type { SiteSettings } from '../navigation/types';

export async function saveSiteSettings(settings: SiteSettings) {
  const response = await fetch('/api/settings/site', {
    method: 'PUT',
    credentials: 'same-origin',
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  });
  if (!response.ok)
    throw new Error(`Site settings update failed: ${response.status}`);
}

export function uploadSiteLogo(file: File) {
  return uploadImage('/api/settings/site/logo', file);
}

export function uploadFavicon(file: File) {
  return uploadImage('/api/settings/site/favicon', file);
}

export function uploadBackground(file: File) {
  return uploadImage('/api/settings/site/background', file);
}

async function uploadImage(path: string, file: File) {
  const body = new FormData();
  body.append('image', file);
  const response = await fetch(path, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
    body,
  });
  if (!response.ok) throw new Error(`Image upload failed: ${response.status}`);
  const value: unknown = await response.json();
  if (
    !isRecord(value) ||
    typeof value.url !== 'string' ||
    !value.url.startsWith('/uploads/')
  ) {
    throw new Error('Image upload response is invalid');
  }
  return value.url;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}
