import { ApiError, downloadFile, request } from '../api/client';

export type ExportScope = 'all' | 'public' | 'private';
export type ExportFormat = 'html' | 'json';

export async function downloadBookmarkExport(
  format: ExportFormat,
  scope: ExportScope,
) {
  const query = new URLSearchParams({ format, scope });
  const response = await request(`/api/bookmarks/export?${query}`);
  if (!response.ok) {
    throw new ApiError('bookmark export failed', { status: response.status });
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  downloadFile(url, `iiiu-nav-bookmarks.${format}`);
  URL.revokeObjectURL(url);
}
