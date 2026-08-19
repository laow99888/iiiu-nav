import { afterEach, describe, expect, it, vi } from 'vitest';

import { fetchUpdateStatus } from './update-api';

describe('版本检测 API', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('读取缓存状态并通过 POST 强制重新检查', async () => {
    const status = {
      automaticUpdate: false,
      checkedAt: '2026-08-20T08:00:00Z',
      currentVersion: 'v1.0.0',
      latestVersion: 'v1.1.0',
      manifestUrl:
        'https://github.com/laow99888/iiiu-nav/releases/download/v1.1.0/release-manifest.json',
      publishedAt: '2026-08-20T07:00:00Z',
      releaseUrl: 'https://github.com/laow99888/iiiu-nav/releases/tag/v1.1.0',
      state: 'update_available',
    } as const;
    const fetchMock = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(new Response(JSON.stringify(status), { status: 200 })),
      );
    vi.stubGlobal('fetch', fetchMock);

    await expect(fetchUpdateStatus()).resolves.toEqual(status);
    await expect(fetchUpdateStatus(true)).resolves.toEqual(status);
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      '/api/updates',
      expect.objectContaining({ method: 'GET', credentials: 'same-origin' }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      '/api/updates/check',
      expect.objectContaining({ method: 'POST', credentials: 'same-origin' }),
    );
  });

  it('拒绝非官方链接和不稳定版本', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            automaticUpdate: false,
            currentVersion: 'v1.0.0',
            latestVersion: 'v1.1.0-beta',
            manifestUrl: 'https://attacker.example/manifest.json',
            publishedAt: '2026-08-20T07:00:00Z',
            releaseUrl: 'https://attacker.example/release',
            state: 'update_available',
          }),
          { status: 200 },
        ),
      ),
    );

    await expect(fetchUpdateStatus()).rejects.toThrow(
      'Update status response is invalid',
    );
  });
});
