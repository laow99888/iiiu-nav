import { ApiError, isRecord, request } from '../api/client';

export type ImportVisibility = 'private' | 'public';
export type DuplicateStrategy = 'skip' | 'update' | 'create';
export type ImportEntryStatus = 'new' | 'duplicate' | 'invalid';

export type ImportPreview = {
  format: 'html' | 'json';
  visibility: ImportVisibility;
  summary: {
    new: number;
    duplicates: number;
    invalid: number;
    newCategories: number;
    existingCategories: number;
  };
  categories: Array<{
    name: string;
    mapping: 'new' | 'existing';
    existingId?: string;
    links: Array<{
      name: string;
      url: string;
      status: ImportEntryStatus;
      problem?: string;
      duplicateOf?: string;
    }>;
  }>;
};

export type ImportResult = {
  createdCategories: number;
  createdLinks: number;
  updatedLinks: number;
  skippedLinks: number;
  invalidLinks: number;
};

export async function previewBookmarks(
  file: File,
  visibility: ImportVisibility,
) {
  return parseImportPreview(
    await sendImport<unknown>('/api/imports/preview', file, visibility),
  );
}

export async function importBookmarks(
  file: File,
  visibility: ImportVisibility,
  duplicates: DuplicateStrategy,
) {
  return parseImportResult(
    await sendImport<unknown>(
      '/api/imports/commit',
      file,
      visibility,
      duplicates,
    ),
  );
}

function parseImportPreview(value: unknown): ImportPreview {
  if (!isRecord(value)) {
    throw new Error('invalid import preview');
  }
  const { format, visibility, summary, categories } = value;
  if (
    (format !== 'html' && format !== 'json') ||
    (visibility !== 'public' && visibility !== 'private') ||
    !isRecord(summary) ||
    !Array.isArray(categories)
  ) {
    throw new Error('invalid import preview');
  }
  for (const key of [
    'new',
    'duplicates',
    'invalid',
    'newCategories',
    'existingCategories',
  ]) {
    if (typeof summary[key] !== 'number')
      throw new Error('invalid import preview');
  }
  return value as ImportPreview;
}

function parseImportResult(value: unknown): ImportResult {
  if (!isRecord(value)) {
    throw new Error('invalid import result');
  }
  for (const key of [
    'createdCategories',
    'createdLinks',
    'updatedLinks',
    'skippedLinks',
    'invalidLinks',
  ]) {
    if (typeof value[key] !== 'number')
      throw new Error('invalid import result');
  }
  return value as ImportResult;
}

async function sendImport<Result>(
  endpoint: string,
  file: File,
  visibility: ImportVisibility,
  duplicates?: DuplicateStrategy,
): Promise<Result> {
  const body = new FormData();
  body.set('bookmarks', file);
  body.set('visibility', visibility);
  if (duplicates) body.set('duplicates', duplicates);
  const response = await request(endpoint, { method: 'POST', body });
  if (!response.ok) {
    throw new ApiError('bookmark import request failed', {
      status: response.status,
    });
  }
  return (await response.json()) as Result;
}
