import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ToastProvider } from '../ui/primitives';
import { BackupDrawer } from './backup-drawer';

function backup(name: string, size = 2048) {
  return { name, size, createdAt: '2026-08-19T01:02:03Z' };
}

function renderBackups() {
  return render(
    <ToastProvider>
      <BackupDrawer
        open
        onClose={() => undefined}
        onRestored={() => undefined}
      />
    </ToastProvider>,
  );
}

describe('数据备份', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('加载、创建、下载并确认删除备份', async () => {
    const user = userEvent.setup();
    const existing = backup('iiiu-nav-backup-20260819-010203-1234abcd.zip');
    const created = backup('iiiu-nav-backup-20260819-020304-abcd1234.zip');
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ backups: [existing] }, 200))
      .mockResolvedValueOnce(jsonResponse(created, 201))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined);
    renderBackups();
    const drawer = screen.getByRole('dialog', { name: '数据备份' });

    await waitFor(() =>
      expect(
        within(drawer).getByRole('button', { name: '下载备份' }),
      ).toBeInTheDocument(),
    );
    await user.click(within(drawer).getByRole('button', { name: '创建备份' }));
    await waitFor(() =>
      expect(
        within(drawer).getAllByRole('button', { name: '下载备份' }),
      ).toHaveLength(2),
    );
    expect(
      screen.getByText('数据库与上传文件已写入完整 ZIP。'),
    ).toBeInTheDocument();

    await user.click(
      within(drawer).getAllByRole('button', { name: '下载备份' })[0]!,
    );
    expect(click).toHaveBeenCalledOnce();

    await user.click(
      within(drawer).getAllByRole('button', { name: '删除备份' })[0]!,
    );
    const dialog = screen.getByRole('dialog', { name: '删除这个备份？' });
    expect(within(dialog).getByText(created.name)).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: '确认删除' }));
    await waitFor(() => expect(screen.queryByText(created.name)).toBeNull());
    expect(fetchMock.mock.calls[2]?.[0]).toContain(
      encodeURIComponent(created.name),
    );
  });

  it('列表和创建失败时显示明确错误', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('', { status: 500 }))
      .mockResolvedValueOnce(jsonResponse({ backups: [] }, 200))
      .mockResolvedValueOnce(new Response('', { status: 507 }));
    vi.stubGlobal('fetch', fetchMock);
    const firstView = renderBackups();
    const drawer = screen.getByRole('dialog', { name: '数据备份' });
    expect(await within(drawer).findByRole('alert')).toHaveTextContent(
      '无法读取备份列表',
    );

    firstView.unmount();
    renderBackups();
    const second = screen.getByRole('dialog', { name: '数据备份' });
    await waitFor(() =>
      expect(
        within(second).getByRole('button', { name: '创建备份' }),
      ).toBeEnabled(),
    );
    await user.click(within(second).getByRole('button', { name: '创建备份' }));
    expect(await within(second).findByRole('alert')).toHaveTextContent(
      '检查磁盘空间',
    );
  });
});

function jsonResponse(value: unknown, status: number) {
  return new Response(JSON.stringify(value), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
