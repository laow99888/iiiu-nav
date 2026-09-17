import { ApiError, isRecord, request, requestJSON } from '../api/client';
import type { SiteSettings } from '../navigation/types';

export async function saveSiteSettings(settings: SiteSettings) {
  const response = await request('/api/settings/site', {
    method: 'PUT',
    body: JSON.stringify(settings),
  });
  if (!response.ok) {
    throw new ApiError(`Site settings update failed: ${response.status}`, {
      status: response.status,
    });
  }
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
  const value: unknown = await requestJSON(
    path,
    { method: 'POST', body },
    'Image upload failed',
  );
  if (
    !isRecord(value) ||
    typeof value.url !== 'string' ||
    !value.url.startsWith('/uploads/')
  ) {
    throw new Error('Image upload response is invalid');
  }
  return value.url;
}
