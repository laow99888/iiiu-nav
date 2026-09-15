import { render } from '@testing-library/preact';
import { beforeEach } from 'vitest';
import { monitorForElements } from '@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter';
import { describe, expect, it, vi } from 'vitest';

import { SortableList } from './sortable-list';

// jsdom 没有 PointerEvent，全局测试桩替换了 pdnd 的注册函数。这里通过被调用
// 的 monitor 参数直接驱动生产环境的 onDrop 逻辑，覆盖指针拖拽路径的重排粘合。
type MonitorArgs = Parameters<typeof monitorForElements>[0];

function monitorArgs(): MonitorArgs {
  const calls = vi.mocked(monitorForElements).mock.calls;
  const latest = calls[calls.length - 1];
  if (!latest) throw new Error('monitorForElements was not registered');
  return latest[0];
}

function renderList(onReorder: (items: string[]) => void) {
  render(
    <SortableList
      items={['a', 'b', 'c']}
      getId={(item) => item}
      getLabel={(item) => item}
      onReorder={onReorder}
      renderItem={(item) => <span>{item}</span>}
    />,
  );
}

function monitorCallbacks() {
  const { canMonitor, onDrop } = monitorArgs();
  if (!canMonitor || !onDrop) {
    throw new Error('monitor callbacks were not registered');
  }
  return { canMonitor, onDrop };
}

describe('可排序列表的拖拽粘合', () => {
  beforeEach(() => {
    vi.mocked(monitorForElements).mockClear();
  });

  it('拖放落点触发重排', () => {
    const onReorder = vi.fn();
    renderList(onReorder);

    const { canMonitor, onDrop } = monitorCallbacks();
    expect(
      canMonitor({
        source: { data: { type: 'iiiu-nav-sortable' } },
      } as unknown as Parameters<typeof canMonitor>[0]),
    ).toBe(true);

    onDrop({
      source: { data: { type: 'iiiu-nav-sortable', index: 0 } },
      location: { current: { dropTargets: [{ data: { index: 2 } }] } },
    } as unknown as Parameters<typeof onDrop>[0]);
    expect(onReorder).toHaveBeenCalledWith(['b', 'c', 'a']);
  });

  it('忽略无关来源与无效落点', () => {
    const onReorder = vi.fn();
    renderList(onReorder);

    const { canMonitor, onDrop } = monitorCallbacks();
    expect(
      canMonitor({
        source: { data: { type: 'something-else' } },
      } as unknown as Parameters<typeof canMonitor>[0]),
    ).toBe(false);

    onDrop({
      source: { data: { type: 'iiiu-nav-sortable', index: 0 } },
      location: { current: { dropTargets: [] } },
    } as unknown as Parameters<typeof onDrop>[0]);
    onDrop({
      source: { data: { type: 'iiiu-nav-sortable' } },
      location: { current: { dropTargets: [{ data: { index: 1 } }] } },
    } as unknown as Parameters<typeof onDrop>[0]);
    expect(onReorder).not.toHaveBeenCalled();
  });
});
