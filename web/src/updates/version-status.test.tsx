import { render, screen, waitFor } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { VersionStatus } from './version-status';

describe('版本状态', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('展示可用正式版本、发布链接与手动 Docker 更新命令', async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          automaticUpdate: false,
          checkedAt: '2026-08-20T08:00:00Z',
          currentVersion: 'v1.0.0',
          latestVersion: 'v1.1.0',
          manifestUrl:
            'https://github.com/laow99888/iiiu-nav/releases/download/v1.1.0/release-manifest.json',
          publishedAt: '2026-08-20T07:00:00Z',
          releaseUrl:
            'https://github.com/laow99888/iiiu-nav/releases/tag/v1.1.0',
          state: 'update_available',
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal('fetch', fetchMock);

    render(<VersionStatus />);
    expect(await screen.findByText('发现新版本 v1.1.0')).toBeInTheDocument();
    expect(
      screen.getByText(
        'docker compose pull app && docker compose up -d --wait app',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /查看发布说明/ })).toHaveAttribute(
      'href',
      'https://github.com/laow99888/iiiu-nav/releases/tag/v1.1.0',
    );

    await user.click(screen.getByRole('button', { name: '检查更新' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/updates/check');
  });

  it('开发构建显示本地状态且不提供更新命令', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            automaticUpdate: false,
            currentVersion: 'local',
            state: 'development',
          }),
          { status: 200 },
        ),
      ),
    );

    render(<VersionStatus />);
    expect(await screen.findByText('开发构建（local）')).toBeInTheDocument();
    expect(screen.queryByText('服务器手动更新')).not.toBeInTheDocument();
  });

  it('GitHub 限流状态不影响手动重试', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            automaticUpdate: false,
            checkedAt: '2026-08-20T08:00:00Z',
            currentVersion: 'v1.0.0',
            state: 'rate_limited',
          }),
          { status: 200 },
        ),
      ),
    );

    render(<VersionStatus />);
    expect(
      await screen.findByText('GitHub 检查频率受限，请稍后重试'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '检查更新' })).toBeEnabled();
  });
});
