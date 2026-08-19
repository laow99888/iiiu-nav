import type { ComponentChildren } from 'preact';
import { useCallback, useMemo, useState } from 'preact/hooks';

import { messages } from '../../i18n/messages';
import { Check, CircleAlert, Info, X } from '../icons/interface-icons';
import { Button } from './button';
import { ToastContext, type ToastInput } from './toast-context';

type ToastItem = ToastInput & { id: number };

let nextToastId = 0;

export function ToastProvider({ children }: { children: ComponentChildren }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const notify = useCallback(
    (input: ToastInput) => {
      const id = ++nextToastId;
      setToasts((current) => [...current, { ...input, id }]);
      if (input.duration !== 0) {
        window.setTimeout(() => dismiss(id), input.duration ?? 4500);
      }
      return id;
    },
    [dismiss],
  );

  const value = useMemo(() => ({ dismiss, notify }), [dismiss, notify]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div
        class="ui-toast-viewport"
        aria-label={messages.ui.notifications}
        aria-live="polite"
      >
        {toasts.map((toast) => {
          const Icon =
            toast.tone === 'success'
              ? Check
              : toast.tone === 'error'
                ? CircleAlert
                : Info;
          return (
            <div
              key={toast.id}
              class={`ui-toast ui-toast--${toast.tone ?? 'info'}`}
              role={toast.tone === 'error' ? 'alert' : 'status'}
            >
              <Icon class="ui-toast__icon" aria-hidden="true" />
              <div class="ui-toast__content">
                <strong>{toast.title}</strong>
                <p>{toast.message}</p>
              </div>
              <Button
                variant="ghost"
                size="small"
                icon={X}
                aria-label={messages.ui.closeToast}
                onClick={() => dismiss(toast.id)}
              />
            </div>
          );
        })}
      </div>
    </ToastContext.Provider>
  );
}
