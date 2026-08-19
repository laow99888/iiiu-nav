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
  const response = await fetch('/api/metadata/recognize', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
    body: JSON.stringify({ url: normalizeLinkURL(url) }),
  });
  if (!response.ok) throw new Error(`Recognition failed: ${response.status}`);
  const value: unknown = await response.json();
  if (!isRecognitionResult(value))
    throw new Error('Recognition response is invalid');
  return value;
}

export async function uploadLogo(file: File): Promise<string> {
  const form = new FormData();
  form.append('logo', file);
  const response = await fetch('/api/logos', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
    body: form,
  });
  if (!response.ok) throw new Error(`Logo upload failed: ${response.status}`);
  const value: unknown = await response.json();
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
  const response = await fetch('/api/links/refresh', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) throw new Error(`Link refresh failed: ${response.status}`);
  const value: unknown = await response.json();
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
  const response = await fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers: init.body
      ? { Accept: 'application/json', 'Content-Type': 'application/json' }
      : { Accept: 'application/json' },
  });
  if (!response.ok) throw new Error(`Link request failed: ${response.status}`);
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

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}
