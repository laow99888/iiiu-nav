import { createContext } from 'preact';
import { useContext } from 'preact/hooks';

export type ToastTone = 'info' | 'success' | 'error';

export type ToastInput = {
  duration?: number;
  message: string;
  title: string;
  tone?: ToastTone;
};

export type ToastContextValue = {
  dismiss: (id: number) => void;
  notify: (toast: ToastInput) => number;
};

export const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used inside ToastProvider');
  }
  return context;
}
