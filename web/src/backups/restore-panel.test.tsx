import { render, screen, waitFor } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ToastProvider } from '../ui/primitives';
import { RestorePanel } from './restore-panel';

function renderRestore(onRestored = vi.fn()) {
  render(
    <ToastProvider>
      <RestorePanel onRestored={onRestored} />
    </ToastProvider>,
  );
  return onRestored;
}

describe('完整恢复', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('要求文件、密码和确认词并提交完整表单', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    const onRestored = renderRestore();
    const action = screen.getByRole('button', { name: '验证并恢复' });
    expect(action).toBeDisabled();

    const file = new File(['backup'], 'backup.zip', {
      type: 'application/zip',
    });
    await user.upload(screen.getByLabelText('选择备份 ZIP'), file);
    await user.type(
      screen.getByLabelText('当前管理员密码'),
      'current password',
    );
    await user.type(screen.getByLabelText(/输入 RESTORE 确认/), 'RESTORE');
    expect(action).toBeEnabled();
    await user.click(action);

    await waitFor(() => expect(onRestored).toHaveBeenCalledOnce());
    const body = fetchMock.mock.calls[0]?.[1]?.body as FormData;
    expect(body.get('backup')).toBe(file);
    expect(body.get('password')).toBe('current password');
    expect(body.get('confirmation')).toBe('RESTORE');
    expect(
      screen.getByText('当前会话已失效，请使用备份中的管理员密码登录。'),
    ).toBeInTheDocument();
  });

  it('映射密码、兼容性、大小、空间和回滚错误', async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: 'restore_archive_incompatible' }), {
        status: 422,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);
    renderRestore();
    await user.upload(
      screen.getByLabelText('选择备份 ZIP'),
      new File(['backup'], 'backup.zip', { type: 'application/zip' }),
    );
    await user.type(
      screen.getByLabelText('当前管理员密码'),
      'current password',
    );
    await user.type(screen.getByLabelText(/输入 RESTORE 确认/), 'RESTORE');
    await user.click(screen.getByRole('button', { name: '验证并恢复' }));
    expect(await screen.findByRole('alert')).toHaveTextContent(
      '结构与当前程序不兼容',
    );
  });
});
