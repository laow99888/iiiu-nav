import { createPortal } from 'preact/compat';
import type { ComponentChildren, JSX } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

const focusableSelector = [
  'button:not([disabled])',
  '[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',');

// Stacked surfaces (a delete dialog over an editing drawer) each listen on the
// document, so only the most recently opened one may react to Escape.
const openSurfaceIDs: number[] = [];
let nextSurfaceID = 1;

type ModalSurfaceProps = {
  children: ComponentChildren;
  class?: string;
  labelledBy: string;
  onClose: () => void;
  role?: JSX.AriaRole;
};

export function ModalSurface({
  children,
  class: className,
  labelledBy,
  onClose,
  role = 'dialog',
}: ModalSurfaceProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const onCloseRef = useRef(onClose);

  useLayoutEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useLayoutEffect(() => {
    const surfaceID = nextSurfaceID++;
    openSurfaceIDs.push(surfaceID);
    const previousFocus = document.activeElement as HTMLElement | null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    // 浏览器返回键（尤其 Android）应关闭最上层浮层，而不是直接离开页面。
    window.history.pushState({ uiModal: surfaceID }, '');
    let sentinelActive = true;
    let closedByHistory = false;
    const handlePopState = () => {
      const state = window.history.state as { uiModal?: number } | null;
      if (sentinelActive && state?.uiModal !== surfaceID) {
        // 历史离开了本浮层的哨兵条目：说明用户按了返回键。
        sentinelActive = false;
        closedByHistory = true;
        onCloseRef.current();
        return;
      }
      if (state?.uiModal === surfaceID) {
        sentinelActive = true;
      }
    };
    window.addEventListener('popstate', handlePopState);

    const panel = panelRef.current;
    const firstFocusable = panel?.querySelector<HTMLElement>(focusableSelector);
    (firstFocusable ?? panel)?.focus();

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (openSurfaceIDs[openSurfaceIDs.length - 1] === surfaceID) {
          event.preventDefault();
          onCloseRef.current();
        }
        return;
      }

      if (event.key !== 'Tab' || !panel) {
        return;
      }

      const focusable = Array.from(
        panel.querySelectorAll<HTMLElement>(focusableSelector),
      );
      if (focusable.length === 0) {
        event.preventDefault();
        panel.focus();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      const index = openSurfaceIDs.indexOf(surfaceID);
      if (index >= 0) {
        openSurfaceIDs.splice(index, 1);
      }
      document.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('popstate', handlePopState);
      const state = window.history.state as { uiModal?: number } | null;
      if (!closedByHistory && state && state.uiModal === surfaceID) {
        window.history.back();
      }
      document.body.style.overflow = previousOverflow;
      previousFocus?.focus();
    };
  }, []);

  return createPortal(
    <div
      class="ui-modal-backdrop"
      onMouseDown={(event) => {
        if (event.currentTarget === event.target) {
          onClose();
        }
      }}
    >
      <div
        ref={panelRef}
        class={className}
        role={role}
        aria-modal="true"
        aria-labelledby={labelledBy}
        tabIndex={-1}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}
