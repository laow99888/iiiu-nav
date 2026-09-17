import { cleanup, render, screen, waitFor } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { useState } from 'preact/hooks';
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

  it('堆叠浮层时 Escape 只关闭最上层', async () => {
    const user = userEvent.setup();
    const events: string[] = [];

    function Harness() {
      const [stage, setStage] = useState<'both' | 'drawer' | 'none'>('both');
      const close = (label: string) => {
        events.push(label);
        setStage((current) => (current === 'both' ? 'drawer' : 'none'));
      };
      return (
        <>
          {stage !== 'none' && (
            <Drawer open title="编辑链接" onClose={() => close('drawer')}>
              表单内容
            </Drawer>
          )}
          {stage === 'both' && (
            <Dialog open title="删除链接" onClose={() => close('dialog')}>
              <Button>确认</Button>
            </Dialog>
          )}
        </>
      );
    }

    render(<Harness />);
    await user.keyboard('{Escape}');
    expect(events).toEqual(['dialog']);

    await user.keyboard('{Escape}');
    expect(events).toEqual(['dialog', 'drawer']);
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

describe('浮层与浏览器历史', () => {
  it('浏览器返回键关闭最上层浮层', async () => {
    const onClose = vi.fn();
    window.history.pushState(null, '', '/');
    render(
      <Dialog open title="返回键浮层" onClose={onClose}>
        <Button>确认</Button>
      </Dialog>,
    );

    await waitFor(() =>
      expect(
        (window.history.state as { uiModal?: number } | null)?.uiModal,
      ).toBeDefined(),
    );
    window.history.back();
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
    cleanup();
  });

  it('浮层被主动关闭时回退哨兵历史记录', async () => {
    const user = userEvent.setup();
    window.history.pushState(null, '', '/');
    function Harness() {
      const [open, setOpen] = useState(true);
      if (!open) {
        return <p>已关闭</p>;
      }
      return (
        <Dialog open title="历史浮层" onClose={() => setOpen(false)}>
          <Button onClick={() => setOpen(false)}>关闭</Button>
        </Dialog>
      );
    }
    render(<Harness />);
    await waitFor(() =>
      expect(
        (window.history.state as { uiModal?: number } | null)?.uiModal,
      ).toBeDefined(),
    );

    await user.click(screen.getByRole('button', { name: '关闭' }));
    expect(screen.getByText('已关闭')).toBeInTheDocument();
    await waitFor(() =>
      expect(
        (window.history.state as { uiModal?: number } | null)?.uiModal,
      ).toBeUndefined(),
    );
  });
});
