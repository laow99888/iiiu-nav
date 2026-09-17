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

  it('配置执行器后可一键更新并跟踪进度', async () => {
    const user = userEvent.setup();
    const statusPayload = {
      automaticUpdate: true,
      checkedAt: '2026-08-20T08:00:00Z',
      currentVersion: 'v1.0.0',
      latestVersion: 'v1.1.0',
      manifestUrl:
        'https://github.com/laow99888/iiiu-nav/releases/download/v1.1.0/release-manifest.json',
      publishedAt: '2026-08-20T07:00:00Z',
      releaseUrl: 'https://github.com/laow99888/iiiu-nav/releases/tag/v1.1.0',
      state: 'update_available',
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(statusPayload), { status: 200 }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ started: true }), { status: 202 }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            state: 'running',
            phase: 'pulling',
            targetVersion: 'v1.1.0',
          }),
          { status: 200 },
        ),
      );
    vi.stubGlobal('fetch', fetchMock);

    render(<VersionStatus />);
    expect(
      await screen.findByRole('button', { name: '一键更新到 v1.1.0' }),
    ).toBeInTheDocument();
    expect(screen.getByText('已启用一键更新执行器')).toBeInTheDocument();
    expect(screen.queryByText('服务器手动更新')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '一键更新到 v1.1.0' }));
    expect(
      screen.getByText(
        '更新会先创建完整备份，然后短暂重启应用。期间请保持本页面打开，完成后会自动刷新。',
      ),
    ).toBeInTheDocument();

    await user.type(screen.getByLabelText(/当前管理员密码/), 'secret');
    await user.click(screen.getByRole('button', { name: '确认并开始更新' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(String(fetchMock.mock.calls[1]?.[0])).toBe('/api/updates/install');
    expect(await screen.findByText(/一键更新进行中/)).toBeInTheDocument();
    // 轮询到执行器进度后展示对应阶段文案。
    await waitFor(() =>
      expect(screen.getByText(/正在拉取新版本镜像/)).toBeInTheDocument(),
    );
  });

  it('密码错误时提示且不会进入更新流程', async () => {
    const user = userEvent.setup();
    const statusPayload = {
      automaticUpdate: true,
      checkedAt: '2026-08-20T08:00:00Z',
      currentVersion: 'v1.0.0',
      latestVersion: 'v1.1.0',
      manifestUrl:
        'https://github.com/laow99888/iiiu-nav/releases/download/v1.1.0/release-manifest.json',
      publishedAt: '2026-08-20T07:00:00Z',
      releaseUrl: 'https://github.com/laow99888/iiiu-nav/releases/tag/v1.1.0',
      state: 'update_available',
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(statusPayload), { status: 200 }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'update_password_invalid' }), {
          status: 401,
        }),
      );
    vi.stubGlobal('fetch', fetchMock);

    render(<VersionStatus />);
    await user.click(
      await screen.findByRole('button', { name: '一键更新到 v1.1.0' }),
    );
    await user.type(screen.getByLabelText(/当前管理员密码/), 'wrong');
    await user.click(screen.getByRole('button', { name: '确认并开始更新' }));

    expect(await screen.findByText('密码不正确，请重试。')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(
      screen.getByRole('button', { name: '检查更新' }),
    ).toBeInTheDocument();
  });
});
