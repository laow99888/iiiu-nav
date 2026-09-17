import {
  ApiError,
  isRecord,
  readErrorCode,
  request,
  requestJSON,
} from '../api/client';

export type LinkInput = {
  categoryId: string;
  description: string;
  logoText: string;
  iconSource: 'auto' | 'generated' | 'upload' | 'url';
  iconValue: string;
  name: string;
  url: string;
};

export type RecognitionResult = {
  description: string;
  iconSource: LinkInput['iconSource'];
  iconValue: string;
  name: string;
};

export function normalizeLinkURL(value: string) {
  const trimmed = value.trim();
  if (!trimmed || /^[a-z][a-z0-9+.-]*:\/\//i.test(trimmed)) {
    return trimmed;
  }
  return `https://${trimmed}`;
}

export function createLink(input: LinkInput) {
  return linkRequest('/api/links', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updateLink(id: string, input: LinkInput) {
  return linkRequest(`/api/links/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}

export function deleteLink(id: string) {
  return linkRequest(`/api/links/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export function reorderLinks(categoryID: string, ids: readonly string[]) {
  return linkRequest(
    `/api/categories/${encodeURIComponent(categoryID)}/links/order`,
    { method: 'PUT', body: JSON.stringify({ ids }) },
  );
}

export async function recognizeLink(url: string): Promise<RecognitionResult> {
  const response = await request('/api/metadata/recognize', {
    method: 'POST',
    body: JSON.stringify({ url: normalizeLinkURL(url) }),
  });
  if (!response.ok) {
    const code = await readErrorCode(response);
    throw new ApiError(code ?? `Recognition failed: ${response.status}`, {
      code,
      status: response.status,
    });
  }
  const value: unknown = await response.json();
  if (!isRecognitionResult(value))
    throw new Error('Recognition response is invalid');
  return value;
}

export function isMetadataTargetBlocked(error: unknown) {
  return (
    error instanceof ApiError && error.code === 'metadata_target_not_public'
  );
}

export async function uploadLogo(file: File): Promise<string> {
  const form = new FormData();
  form.append('logo', file);
  const value: unknown = await requestJSON(
    '/api/logos',
    { method: 'POST', body: form },
    'Logo upload failed',
  );
  if (
    !isRecord(value) ||
    typeof value.url !== 'string' ||
    !value.url.startsWith('/uploads/logos/')
  ) {
    throw new Error('Logo upload response is invalid');
  }
  return value.url;
}

export async function refreshAllLinks(): Promise<{
  updated: number;
  failed: number;
}> {
  const value: unknown = await requestJSON(
    '/api/links/refresh',
    { method: 'POST' },
    'Link refresh failed',
  );
  if (
    !isRecord(value) ||
    typeof value.updated !== 'number' ||
    typeof value.failed !== 'number'
  ) {
    throw new Error('Link refresh response is invalid');
  }
  return { updated: value.updated, failed: value.failed };
}

async function linkRequest(path: string, init: RequestInit) {
  const response = await request(path, init);
  if (!response.ok) {
    throw new ApiError(`Link request failed: ${response.status}`, {
      status: response.status,
    });
  }
}

function isRecognitionResult(value: unknown): value is RecognitionResult {
  return (
    isRecord(value) &&
    typeof value.name === 'string' &&
    typeof value.description === 'string' &&
    typeof value.iconValue === 'string' &&
    (value.iconSource === 'auto' ||
      value.iconSource === 'generated' ||
      value.iconSource === 'upload' ||
      value.iconSource === 'url')
  );
}
