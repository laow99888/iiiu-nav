import { afterEach, describe, expect, it, vi } from 'vitest';

import { changePassword, login, logout } from './auth-api';

describe('认证 API', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('使用同源 Cookie 发送登录、退出和改密请求', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);

    await login('current secret');
    await changePassword('current secret', 'replacement secret');
    await logout();

    expect(fetchMock.mock.calls).toEqual([
      [
        '/api/auth/login',
        expect.objectContaining({
          method: 'POST',
          credentials: 'same-origin',
          body: JSON.stringify({ password: 'current secret' }),
        }),
      ],
      [
        '/api/auth/password',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            currentPassword: 'current secret',
            newPassword: 'replacement secret',
          }),
        }),
      ],
      ['/api/auth/logout', expect.objectContaining({ method: 'POST' })],
    ]);
  });

  it('保留服务端错误代码和限流等待时间', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'login_rate_limited' }), {
          status: 429,
          headers: {
            'Content-Type': 'application/json',
            'Retry-After': '42',
          },
        }),
      ),
    );

    await expect(login('wrong password')).rejects.toEqual(
      expect.objectContaining({
        status: 429,
        code: 'login_rate_limited',
        retryAfter: 42,
      }),
    );
  });
});
