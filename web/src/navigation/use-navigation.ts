import { useCallback, useEffect, useState } from 'preact/hooks';

import { fetchNavigation, type NavigationSnapshot } from './navigation-api';

type NavigationState =
  | { status: 'loading'; snapshot: null }
  | { status: 'ready'; snapshot: NavigationSnapshot }
  | { status: 'error'; snapshot: null };

export function useNavigation() {
  const [revision, setRevision] = useState(0);
  const [state, setState] = useState<NavigationState>({
    status: 'loading',
    snapshot: null,
  });

  useEffect(() => {
    const controller = new AbortController();
    setState({ status: 'loading', snapshot: null });
    fetchNavigation(controller.signal)
      .then((snapshot) => setState({ status: 'ready', snapshot }))
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === 'AbortError') {
          return;
        }
        setState({ status: 'error', snapshot: null });
      });
    return () => controller.abort();
  }, [revision]);

  const retry = useCallback(() => setRevision((value) => value + 1), []);
  return { ...state, retry };
}
