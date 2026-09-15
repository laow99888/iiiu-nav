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
  if (typeof value !== 'object' || value === null) {
    throw new Error('invalid import preview');
  }
  const { format, visibility, summary, categories } = value as Record<
    string,
    unknown
  >;
  if (
    (format !== 'html' && format !== 'json') ||
    (visibility !== 'public' && visibility !== 'private') ||
    typeof summary !== 'object' ||
    summary === null ||
    !Array.isArray(categories)
  ) {
    throw new Error('invalid import preview');
  }
  const counts = summary as Record<string, unknown>;
  for (const key of [
    'new',
    'duplicates',
    'invalid',
    'newCategories',
    'existingCategories',
  ]) {
    if (typeof counts[key] !== 'number')
      throw new Error('invalid import preview');
  }
  return value as ImportPreview;
}

function parseImportResult(value: unknown): ImportResult {
  if (typeof value !== 'object' || value === null) {
    throw new Error('invalid import result');
  }
  const counts = value as Record<string, unknown>;
  for (const key of [
    'createdCategories',
    'createdLinks',
    'updatedLinks',
    'skippedLinks',
    'invalidLinks',
  ]) {
    if (typeof counts[key] !== 'number')
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
  const response = await fetch(endpoint, {
    method: 'POST',
    credentials: 'same-origin',
    body,
  });
  if (!response.ok) throw new Error('bookmark import request failed');
  return (await response.json()) as Result;
}
