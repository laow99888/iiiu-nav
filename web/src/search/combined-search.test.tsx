import { render, screen } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import { CombinedSearch } from './combined-search';
import { defaultSearchEngines } from './search-engines';

describe('组合搜索', () => {
  it('指针点击本地结果直接打开链接', async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    render(<CombinedSearch categories={navigationFixtures} onOpen={onOpen} />);

    const input = screen.getByRole('combobox', { name: '搜索导航或网页' });
    await user.type(input, 'GitHub');
    await user.click(screen.getByRole('option', { name: /GitHub/ }));
    expect(onOpen).toHaveBeenCalledWith('https://github.com/');
  });

  it('Enter 未选择本地结果时使用当前网页引擎', async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    render(<CombinedSearch categories={navigationFixtures} onOpen={onOpen} />);

    const input = screen.getByRole('combobox', { name: '搜索导航或网页' });
    await user.type(input, 'GitHub{Enter}');
    expect(onOpen).toHaveBeenCalledWith(
      'https://www.google.com/search?q=GitHub',
    );
  });

  it('方向键选中本地结果后 Enter 打开本地链接', async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    render(<CombinedSearch categories={navigationFixtures} onOpen={onOpen} />);

    const input = screen.getByRole('combobox', { name: '搜索导航或网页' });
    await user.type(input, 'GitHub');
    await user.keyboard('{ArrowDown}');
    expect(input).toHaveAttribute('aria-activedescendant');
    expect(screen.getByRole('option', { name: /GitHub/ })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    await user.keyboard('{Enter}');
    expect(onOpen).toHaveBeenCalledWith('https://github.com/');
  });

  it('可切换引擎并遵守启停与排序配置', async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    const engines = [
      { ...defaultSearchEngines[1]!, enabled: true },
      { ...defaultSearchEngines[0]!, enabled: false },
      { ...defaultSearchEngines[2]!, enabled: true },
    ];
    render(
      <CombinedSearch
        categories={navigationFixtures}
        engines={engines}
        onOpen={onOpen}
      />,
    );

    expect(screen.getByRole('button', { name: /百度/ })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /百度/ }));
    const menuItems = screen.getAllByRole('menuitem');
    expect(menuItems.map((item) => item.textContent)).toEqual(['百度', 'Bing']);
    await user.click(screen.getByRole('menuitem', { name: 'Bing' }));

    const input = screen.getByRole('combobox', { name: '搜索导航或网页' });
    await user.type(input, '没有本地结果{Enter}');
    expect(onOpen).toHaveBeenCalledWith(
      'https://www.bing.com/search?q=%E6%B2%A1%E6%9C%89%E6%9C%AC%E5%9C%B0%E7%BB%93%E6%9E%9C',
    );
  });

  it('Escape 关闭结果面板', async () => {
    const user = userEvent.setup();
    render(<CombinedSearch categories={navigationFixtures} onOpen={vi.fn()} />);
    const input = screen.getByRole('combobox', { name: '搜索导航或网页' });
    await user.type(input, 'GitHub');
    expect(screen.getByRole('listbox')).toBeInTheDocument();
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
  });
});
