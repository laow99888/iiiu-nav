import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import { ToastProvider } from '../ui/primitives';
import { CategoryManager } from './category-manager';

function renderManager(onSaved = vi.fn()) {
  render(
    <ToastProvider>
      <CategoryManager
        open
        categories={navigationFixtures}
        onClose={vi.fn()}
        onSaved={onSaved}
      />
    </ToastProvider>,
  );
  return onSaved;
}

describe('分类管理', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('创建私有分类并通过关键词选择稳定图标键', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);
    const onSaved = renderManager();

    await user.click(screen.getByRole('button', { name: '添加分类' }));
    const drawer = screen.getByRole('dialog', { name: '新建分类' });
    await user.type(
      within(drawer).getByLabelText('分类名称 *', { exact: true }),
      '内部工具',
    );
    await user.click(
      within(drawer).getByRole('checkbox', { name: /设为私有分类/ }),
    );
    await user.type(
      within(drawer).getByRole('searchbox', { name: '搜索图标' }),
      '编程',
    );
    await user.click(within(drawer).getByRole('button', { name: '开发' }));
    await user.click(within(drawer).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({
      name: '内部工具',
      iconName: 'code',
      visibility: 'private',
    });
  });

  it('提供无图标选项并在保存失败时保留可编辑表单', async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('', { status: 500 })),
    );
    renderManager();

    await user.click(screen.getByRole('button', { name: '添加分类' }));
    const drawer = screen.getByRole('dialog', { name: '新建分类' });
    await user.type(
      within(drawer).getByLabelText('分类名称 *', { exact: true }),
      '稍后处理',
    );
    await user.click(within(drawer).getByRole('button', { name: '无图标' }));
    await user.click(within(drawer).getByRole('button', { name: '保存' }));

    expect(await within(drawer).findByRole('alert')).toHaveTextContent(
      '保存失败',
    );
    expect(
      within(drawer).getByLabelText('分类名称 *', { exact: true }),
    ).toHaveValue('稍后处理');
  });

  it('编辑已有分类的名称、可见性和图标', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    renderManager();

    await user.click(screen.getByRole('button', { name: '编辑“工作”' }));
    const drawer = screen.getByRole('dialog', { name: '编辑分类' });
    const name = within(drawer).getByLabelText('分类名称 *', { exact: true });
    await user.clear(name);
    await user.type(name, '私有工作');
    await user.click(
      within(drawer).getByRole('checkbox', { name: /设为私有分类/ }),
    );
    await user.click(within(drawer).getByRole('button', { name: '收藏' }));
    await user.click(within(drawer).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/categories/work');
    expect(fetchMock.mock.calls[0]?.[1]).toEqual(
      expect.objectContaining({ method: 'PUT' }),
    );
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({
      name: '私有工作',
      iconName: 'star',
      visibility: 'private',
    });
  });

  it('使用键盘替代按钮调整顺序并保存全部分类 ID', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    renderManager();

    await user.click(screen.getByRole('button', { name: '下移“工作”' }));
    await user.click(screen.getByRole('button', { name: '保存排序' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    const body = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(body.ids.slice(0, 2)).toEqual(['development', 'work']);
    expect(body.ids).toHaveLength(navigationFixtures.length);
  });

  it('删除含链接分类时明确选择移动目标或永久删除', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    renderManager();

    await user.click(screen.getByRole('button', { name: '删除“工作”' }));
    const dialog = screen.getByRole('dialog', { name: '删除“工作”' });
    expect(within(dialog).getByText(/包含 4 个链接/)).toBeInTheDocument();
    expect(
      within(dialog).getByRole('radio', { name: '移动到其他分类' }),
    ).toBeChecked();
    await user.selectOptions(
      within(dialog).getByRole('combobox', { name: '目标分类' }),
      'development',
    );
    await user.click(within(dialog).getByRole('button', { name: '确认删除' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      '/api/categories/work?links=move&target=development',
    );
    expect(
      screen.queryByRole('dialog', { name: '删除“工作”' }),
    ).not.toBeInTheDocument();
  });
});
