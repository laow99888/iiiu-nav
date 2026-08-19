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
  return sendImport<ImportPreview>('/api/imports/preview', file, visibility);
}

export async function importBookmarks(
  file: File,
  visibility: ImportVisibility,
  duplicates: DuplicateStrategy,
) {
  return sendImport<ImportResult>(
    '/api/imports/commit',
    file,
    visibility,
    duplicates,
  );
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
