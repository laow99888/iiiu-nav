import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ToastProvider } from '../ui/primitives';
import { AuthenticationControls } from './authentication-controls';

function renderControls(administrator: boolean, onSessionChanged = vi.fn()) {
  render(
    <ToastProvider>
      <AuthenticationControls
        administrator={administrator}
        onSessionChanged={onSessionChanged}
      />
    </ToastProvider>,
  );
  return onSessionChanged;
}

describe('管理员认证控件', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('登录成功后关闭表单并刷新管理员视图', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    const onSessionChanged = renderControls(false);

    await user.click(screen.getByRole('button', { name: '管理员登录' }));
    const dialog = screen.getByRole('dialog', { name: '管理员登录' });
    await user.type(
      within(dialog).getByLabelText(/管理员密码/),
      'correct password',
    );
    await user.click(
      within(dialog).getByRole('button', { name: '管理员登录' }),
    );

    await waitFor(() => expect(onSessionChanged).toHaveBeenCalledOnce());
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.getByText('管理员模式已启用。')).toBeInTheDocument();
  });

  it('展示错误密码与限流错误且保留表单', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'invalid_credentials' }), {
          status: 401,
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'login_rate_limited' }), {
          status: 429,
          headers: { 'Retry-After': '30' },
        }),
      );
    vi.stubGlobal('fetch', fetchMock);
    renderControls(false);

    await user.click(screen.getByRole('button', { name: '管理员登录' }));
    const dialog = screen.getByRole('dialog', { name: '管理员登录' });
    const password = within(dialog).getByLabelText(/管理员密码/);
    await user.type(password, 'wrong password');
    expect(
      within(dialog).getByRole('button', { name: '管理员登录' }),
    ).toBeEnabled();
    await user.keyboard('{Enter}');
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      '密码不正确。',
    );

    await user.clear(password);
    await user.type(password, 'still wrong');
    await user.keyboard('{Enter}');
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      '请在 30 秒后再试',
    );
  });

  it('校验新密码确认并提交改密请求', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    const onSessionChanged = renderControls(true);

    await user.click(screen.getByRole('button', { name: /管理员菜单/ }));
    await user.click(screen.getByRole('menuitem', { name: '修改密码' }));
    const dialog = screen.getByRole('dialog', { name: '修改管理员密码' });
    await user.type(within(dialog).getByLabelText(/当前密码/), 'old password');
    await user.type(
      within(dialog).getByLabelText('新密码 *', { exact: true }),
      '123456789',
    );
    await user.type(
      within(dialog).getByLabelText(/确认新密码/),
      'does not match',
    );
    expect(
      within(dialog).getByRole('button', { name: '修改密码' }),
    ).toBeDisabled();
    expect(within(dialog).getByRole('alert')).toHaveTextContent(
      '两次输入的新密码不一致。',
    );

    await user.clear(within(dialog).getByLabelText(/确认新密码/));
    await user.type(within(dialog).getByLabelText(/确认新密码/), '123456789');
    const submitPassword = within(dialog).getByRole('button', {
      name: '修改密码',
    });
    expect(submitPassword).toBeEnabled();
    await user.click(submitPassword);

    await waitFor(() => expect(onSessionChanged).toHaveBeenCalledOnce());
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({
      currentPassword: 'old password',
      newPassword: '123456789',
    });
    expect(
      screen.getByText('当前会话已退出，请使用新密码重新登录。'),
    ).toBeInTheDocument();
  });

  it('管理员可退出，失败时保持当前视图并提示', async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'logout_failed' }), {
          status: 500,
        }),
      ),
    );
    const onSessionChanged = renderControls(true);

    await user.click(screen.getByRole('button', { name: /管理员菜单/ }));
    await user.click(screen.getByRole('menuitem', { name: '退出登录' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '会话未能正常退出',
    );
    expect(onSessionChanged).not.toHaveBeenCalled();
    expect(
      screen.getByRole('button', { name: /管理员菜单/ }),
    ).toBeInTheDocument();
  });
});
