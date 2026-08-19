export type ExportScope = 'all' | 'public' | 'private';
export type ExportFormat = 'html' | 'json';

export async function downloadBookmarkExport(
  format: ExportFormat,
  scope: ExportScope,
) {
  const query = new URLSearchParams({ format, scope });
  const response = await fetch(`/api/bookmarks/export?${query}`, {
    credentials: 'same-origin',
  });
  if (!response.ok) throw new Error('bookmark export failed');
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `iiiu-nav-bookmarks.${format}`;
  anchor.click();
  URL.revokeObjectURL(url);
}
