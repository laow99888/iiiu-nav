import { render, screen, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { useState } from 'preact/hooks';
import { describe, expect, it } from 'vitest';

import {
  categoryIconRegistry,
  findCategoryIcons,
} from '../icons/category-icon-registry';
import { SortableList } from '../sortable/sortable-list';
import { reorderItems } from '../sortable/reorder-items';
import { ToastProvider } from './toast';
import { useToast } from './toast-context';

function ToastHarness() {
  const { notify } = useToast();
  return (
    <div>
      <button
        type="button"
        onClick={() =>
          notify({
            title: '保存成功',
            message: '设置已更新',
            tone: 'success',
            duration: 0,
          })
        }
      >
        成功通知
      </button>
      <button
        type="button"
        onClick={() =>
          notify({
            title: '保存失败',
            message: '请稍后重试',
            tone: 'error',
            duration: 0,
          })
        }
      >
        错误通知
      </button>
    </div>
  );
}

function SortableHarness({ disabled = false }: { disabled?: boolean }) {
  const [items, setItems] = useState(['工作', '学习', '工具']);
  return (
    <SortableList
      items={items}
      disabled={disabled}
      getId={(item) => item}
      getLabel={(item) => item}
      renderItem={(item) => <span>{item}</span>}
      onReorder={setItems}
    />
  );
}

describe('反馈、图标和排序', () => {
  it('通知覆盖成功、错误与手动关闭状态', async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>,
    );

    await user.click(screen.getByRole('button', { name: '成功通知' }));
    expect(screen.getByRole('status')).toHaveTextContent('保存成功');
    await user.click(screen.getByRole('button', { name: '错误通知' }));
    expect(screen.getByRole('alert')).toHaveTextContent('保存失败');
    const closeButtons = screen.getAllByRole('button', { name: '关闭通知' });
    await user.click(closeButtons[0]!);
    expect(screen.queryByText('保存成功')).not.toBeInTheDocument();
  });

  it('图标注册表仅返回登记图标并支持中文检索', () => {
    expect(Object.keys(categoryIconRegistry).length).toBeLessThan(30);
    expect(findCategoryIcons('编程').map(([name]) => name)).toEqual(['code']);
    expect(findCategoryIcons('')).toHaveLength(
      Object.keys(categoryIconRegistry).length,
    );
  });

  it('纯排序函数不修改输入数组', () => {
    const source = ['a', 'b', 'c'];
    expect(reorderItems(source, 0, 2)).toEqual(['b', 'c', 'a']);
    expect(source).toEqual(['a', 'b', 'c']);
  });

  it('排序列表提供键盘可用的上下移动替代操作', async () => {
    const user = userEvent.setup();
    render(<SortableHarness />);
    await user.click(screen.getByRole('button', { name: '下移“工作”' }));

    const rows = screen.getAllByRole('listitem');
    expect(within(rows[0]!).getByText('学习')).toBeInTheDocument();
    expect(within(rows[1]!).getByText('工作')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '上移“学习”' })).toBeDisabled();
  });

  it('禁用排序同时禁用拖动和替代操作', () => {
    render(<SortableHarness disabled />);
    expect(
      screen.getByRole('button', { name: '拖动调整“工作”的位置' }),
    ).toBeDisabled();
    expect(screen.getByRole('button', { name: '下移“工作”' })).toBeDisabled();
  });
});
