import { useCallback, useEffect, useRef, useState } from 'preact/hooks';

/** 异步资源的渲染状态：加载中、失败，或就绪（携带加载到的值）。 */
export type AsyncResourceState<T> =
  { kind: 'loading' } | { kind: 'error' } | { kind: 'ready'; value: T };

export type AsyncResource<T> = {
  state: AsyncResourceState<T>;
  /** 重新执行 loader：已就绪/失败的内容保留展示，避免刷新时闪烁，完成后整体替换为新结果。 */
  reload: () => void;
};

/**
 * 加载一个异步资源的通用 hook：内部持有 AbortController 与请求序号守卫，
 * 卸载、中止或被更新的请求取代时忽略过期结果，旧响应永远不会覆盖新状态。
 *
 * - 首次挂载、`enabled` 由关闭转为打开、或 loader 因依赖变化而更新时，
 *   回到加载态重新拉取。loader 必须用 useCallback 稳定，仅在资源定义
 *   真正变化（如查询参数改变）时才允许改变引用。
 * - `reload()` 用于手动刷新（重试、变更后刷新列表），不清空当前内容。
 * - `options.enabled` 为 false 时不发起请求（用于关闭后不加载的抽屉等场景）。
 */
export function useAsyncResource<T>(
  loader: (signal: AbortSignal) => Promise<T>,
  options?: { enabled?: boolean },
): AsyncResource<T> {
  const enabled = options?.enabled !== false;
  const [state, setState] = useState<AsyncResourceState<T>>({
    kind: 'loading',
  });
  const sequenceRef = useRef(0);
  const controllerRef = useRef<AbortController | null>(null);

  const load = useCallback(
    (reset: boolean) => {
      // 请求序号守卫：只有最新一次请求允许写入状态。
      const sequence = ++sequenceRef.current;
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;
      if (reset) setState({ kind: 'loading' });
      void loader(controller.signal).then(
        (value) => {
          if (sequence !== sequenceRef.current) return;
          setState({ kind: 'ready', value });
        },
        () => {
          // 过期请求或中止（含卸载）不落错误态，避免旧响应覆盖新状态。
          if (sequence !== sequenceRef.current || controller.signal.aborted) {
            return;
          }
          setState({ kind: 'error' });
        },
      );
    },
    [loader],
  );

  useEffect(() => {
    if (!enabled) return;
    load(true);
    return () => controllerRef.current?.abort();
  }, [enabled, load]);

  const reload = useCallback(() => load(false), [load]);

  return { reload, state };
}
