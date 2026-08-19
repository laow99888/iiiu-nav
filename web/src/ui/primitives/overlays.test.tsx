import { render, screen, waitFor } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { Button } from './button';
import { Dialog } from './dialog';
import { Drawer } from './drawer';
import { Menu } from './menu';
import { Tooltip } from './tooltip';

describe('浮层控件', () => {
  it('对话框管理初始焦点、Escape 关闭和关闭后的焦点恢复', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <div>
        <button type="button">打开</button>
        <Dialog open title="删除链接" onClose={onClose}>
          <Button>确认</Button>
        </Dialog>
      </div>,
    );

    expect(
      screen.getByRole('dialog', { name: '删除链接' }),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '关闭对话框' })).toHaveFocus(),
    );
    await user.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('抽屉呈现标题并可用关闭按钮退出', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <Drawer open title="站点设置" side="left" onClose={onClose}>
        设置内容
      </Drawer>,
    );

    expect(screen.getByRole('dialog', { name: '站点设置' })).toHaveClass(
      'ui-drawer--left',
    );
    await user.click(screen.getByRole('button', { name: '关闭抽屉' }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('菜单支持禁用项、键盘导航和选择后关闭', async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <Menu
        label="更多"
        items={[
          { id: 'edit', label: '编辑', onSelect },
          {
            id: 'disabled',
            label: '不可用',
            disabled: true,
            onSelect: vi.fn(),
          },
        ]}
      />,
    );

    await user.click(screen.getByRole('button', { name: /更多/ }));
    const edit = screen.getByRole('menuitem', { name: '编辑' });
    await waitFor(() => expect(edit).toHaveFocus());
    expect(screen.getByRole('menuitem', { name: '不可用' })).toBeDisabled();
    await user.keyboard('{Enter}');
    expect(onSelect).toHaveBeenCalledOnce();
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('禁用菜单不会打开', async () => {
    const user = userEvent.setup();
    render(<Menu disabled label="更多" items={[]} />);
    const trigger = screen.getByRole('button', { name: /更多/ });
    expect(trigger).toBeDisabled();
    await user.click(trigger);
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('提示可由悬停和焦点触发', async () => {
    const user = userEvent.setup();
    render(
      <Tooltip content="打开设置">
        <button type="button">设置</button>
      </Tooltip>,
    );
    const trigger = screen.getByRole('button', { name: '设置' });

    await user.hover(trigger);
    expect(screen.getByRole('tooltip')).toHaveTextContent('打开设置');
    await user.unhover(trigger);
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
    await user.tab();
    expect(screen.getByRole('tooltip')).toBeInTheDocument();
  });
});
