import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ToastProvider } from '../ui/primitives';
import { BookmarkImportDrawer } from './bookmark-import';

function renderImport(onImported = vi.fn()) {
  render(
    <ToastProvider>
      <BookmarkImportDrawer
        open
        onClose={() => undefined}
        onImported={onImported}
      />
    </ToastProvider>,
  );
  return onImported;
}

describe('书签导入', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('先显示分类与重复预览，再按选择的策略提交原文件', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({
          format: 'html',
          visibility: 'private',
          summary: {
            new: 1,
            duplicates: 1,
            invalid: 1,
            newCategories: 0,
            existingCategories: 1,
          },
          categories: [
            {
              name: '工具',
              mapping: 'existing',
              existingId: '7',
              links: [
                {
                  name: '新站点',
                  url: 'https://new.example',
                  status: 'new',
                },
                {
                  name: '重复站点',
                  url: 'https://example.com',
                  status: 'duplicate',
                  duplicateOf: '原站点',
                },
                {
                  name: '坏链接',
                  url: 'javascript:void(0)',
                  status: 'invalid',
                  problem: 'url_invalid',
                },
              ],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          createdCategories: 0,
          createdLinks: 1,
          updatedLinks: 1,
          skippedLinks: 0,
          invalidLinks: 1,
        }),
      );
    vi.stubGlobal('fetch', fetchMock);
    const onImported = renderImport();
    const drawer = screen.getByRole('dialog', { name: '导入书签' });
    const file = new File(['bookmarks'], 'bookmarks.html', {
      type: 'text/html',
    });
    await user.upload(within(drawer).getByLabelText('选择书签文件'), file);
    await user.click(within(drawer).getByRole('button', { name: '预览导入' }));

    const preview = await screen.findByRole('region', { name: '导入预览' });
    expect(within(preview).getByText('已有分类')).toBeInTheDocument();
    expect(within(preview).getByText('与“原站点”重复')).toBeInTheDocument();
    expect(within(preview).getByText('内容或网址无效')).toBeInTheDocument();
    await user.selectOptions(
      within(drawer).getByLabelText('重复链接处理'),
      'update',
    );
    await user.click(within(drawer).getByRole('button', { name: '确认导入' }));

    await waitFor(() => expect(onImported).toHaveBeenCalledOnce());
    const previewBody = fetchMock.mock.calls[0]?.[1]?.body as FormData;
    const commitBody = fetchMock.mock.calls[1]?.[1]?.body as FormData;
    expect(previewBody.get('visibility')).toBe('private');
    expect(commitBody.get('duplicates')).toBe('update');
    expect(commitBody.get('bookmarks')).toBe(file);
    expect(
      screen.getByText('新增 1 个，更新 1 个，跳过 0 个。'),
    ).toBeInTheDocument();
  });

  it('预览失败时不允许提交并保留重新选择入口', async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('', { status: 422 })),
    );
    renderImport();
    const drawer = screen.getByRole('dialog', { name: '导入书签' });
    await user.upload(
      within(drawer).getByLabelText('选择书签文件'),
      new File(['bad'], 'bad.html', { type: 'text/html' }),
    );
    await user.click(within(drawer).getByRole('button', { name: '预览导入' }));
    expect(await within(drawer).findByRole('alert')).toHaveTextContent(
      '检查格式、编码和 20 MiB',
    );
    expect(
      within(drawer).getByRole('button', { name: '确认导入' }),
    ).toBeDisabled();
  });
});

function jsonResponse(value: unknown) {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}
