import { act, renderHook, waitFor } from '@testing-library/preact';
import { describe, expect, it, vi } from 'vitest';

import { useAsyncResource } from './use-async-resource';

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, reject, resolve };
}

describe('useAsyncResource', () => {
  it('挂载后加载并在成功时进入就绪态', async () => {
    const loader = vi.fn().mockResolvedValue('内容');
    const { result } = renderHook(() => useAsyncResource(loader));

    expect(result.current.state).toEqual({ kind: 'loading' });
    await waitFor(() =>
      expect(result.current.state).toEqual({ kind: 'ready', value: '内容' }),
    );
    expect(loader).toHaveBeenCalledOnce();
  });

  it('加载失败时进入错误态', async () => {
    const loader = vi.fn().mockRejectedValue(new Error('网络故障'));
    const { result } = renderHook(() => useAsyncResource(loader));

    await waitFor(() =>
      expect(result.current.state).toEqual({ kind: 'error' }),
    );
  });

  it('reload 期间保留已就绪内容，完成后替换为新值', async () => {
    const first = deferred<string>();
    const second = deferred<string>();
    const loader = vi
      .fn<(signal: AbortSignal) => Promise<string>>()
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const { result } = renderHook(() => useAsyncResource(loader));

    await act(async () => {
      first.resolve('旧值');
    });
    expect(result.current.state).toEqual({ kind: 'ready', value: '旧值' });

    act(() => {
      result.current.reload();
    });
    expect(loader).toHaveBeenCalledTimes(2);
    expect(result.current.state).toEqual({ kind: 'ready', value: '旧值' });

    await act(async () => {
      second.resolve('新值');
    });
    expect(result.current.state).toEqual({ kind: 'ready', value: '新值' });
  });

  it('过期响应不会覆盖更新请求的结果', async () => {
    const slow = deferred<string>();
    const fast = deferred<string>();
    const loader = vi
      .fn<(signal: AbortSignal) => Promise<string>>()
      .mockReturnValueOnce(slow.promise)
      .mockReturnValueOnce(fast.promise);
    const { result } = renderHook(() => useAsyncResource(loader));

    act(() => {
      result.current.reload();
    });
    await act(async () => {
      fast.resolve('新结果');
    });
    expect(result.current.state).toEqual({ kind: 'ready', value: '新结果' });

    await act(async () => {
      slow.resolve('过期结果');
    });
    expect(result.current.state).toEqual({ kind: 'ready', value: '新结果' });
  });

  it('卸载时中止在途请求', () => {
    let observed: AbortSignal | undefined;
    const loader = (signal: AbortSignal) => {
      observed = signal;
      return new Promise<string>(() => undefined);
    };
    const { unmount } = renderHook(() => useAsyncResource(loader));

    expect(observed?.aborted).toBe(false);
    unmount();
    expect(observed?.aborted).toBe(true);
  });

  it('enabled 为 false 时不加载，打开后才开始请求', async () => {
    const loader = vi.fn().mockResolvedValue('值');
    const { result, rerender } = renderHook(
      ({ enabled }) => useAsyncResource(loader, { enabled }),
      { initialProps: { enabled: false } },
    );

    expect(loader).not.toHaveBeenCalled();

    rerender({ enabled: true });
    await waitFor(() =>
      expect(result.current.state).toEqual({ kind: 'ready', value: '值' }),
    );
    expect(loader).toHaveBeenCalledOnce();
  });

  it('loader 因依赖变化而更新时回到加载态重新拉取', async () => {
    const loaderA = vi.fn().mockResolvedValue('A');
    const loaderB = vi.fn().mockResolvedValue('B');
    const { result, rerender } = renderHook(
      ({ loader }) => useAsyncResource(loader),
      { initialProps: { loader: loaderA } },
    );

    await waitFor(() =>
      expect(result.current.state).toEqual({ kind: 'ready', value: 'A' }),
    );

    rerender({ loader: loaderB });
    expect(result.current.state).toEqual({ kind: 'loading' });
    await waitFor(() =>
      expect(result.current.state).toEqual({ kind: 'ready', value: 'B' }),
    );
  });
});
