import { render, screen, waitFor } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useNavigation } from './use-navigation';

const snapshot = {
  administrator: true,
  categories: [],
  searchEngines: [
    { id: 'google', enabled: true },
    { id: 'baidu', enabled: true },
    { id: 'bing', enabled: true },
    { id: 'duckduckgo', enabled: true },
  ],
  site: {
    accentColor: '#1769aa',
    backgroundOverlay: 78,
    backgroundUrl: '',
    faviconUrl: '',
    indexingEnabled: false,
    logoUrl: '',
    name: '测试导航',
  },
};

function NavigationHarness() {
  const navigation = useNavigation('all');
  return (
    <div>
      <span>{navigation.status}</span>
      <span>{navigation.snapshot?.site.name}</span>
      <button onClick={navigation.retry}>刷新</button>
    </div>
  );
}

describe('导航数据刷新', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('刷新时保留已经加载的快照和界面', async () => {
    const user = userEvent.setup();
    let resolveRefresh: ((response: Response) => void) | undefined;
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(snapshot), { status: 200 }),
      )
      .mockReturnValueOnce(
        new Promise<Response>((resolve) => {
          resolveRefresh = resolve;
        }),
      );
    vi.stubGlobal('fetch', fetchMock);
    render(<NavigationHarness />);

    expect(await screen.findByText('测试导航')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '刷新' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(screen.getByText('ready')).toBeInTheDocument();
    expect(screen.getByText('测试导航')).toBeInTheDocument();

    resolveRefresh?.(
      new Response(
        JSON.stringify({
          ...snapshot,
          site: { ...snapshot.site, name: '已刷新导航' },
        }),
        { status: 200 },
      ),
    );
    expect(await screen.findByText('已刷新导航')).toBeInTheDocument();
  });
});
