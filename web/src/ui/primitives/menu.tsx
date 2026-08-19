import { createPortal } from 'preact/compat';
import type { JSX } from 'preact';
import { useCallback, useEffect, useId, useRef, useState } from 'preact/hooks';

import { useFloatingPosition } from '../hooks/use-floating-position';
import { ChevronDown, type LucideIcon } from '../icons/interface-icons';
import { Button } from './button';

export type MenuItem = {
  danger?: boolean;
  disabled?: boolean;
  icon?: LucideIcon;
  id: string;
  label: string;
  onSelect: () => void;
};

type MenuProps = {
  disabled?: boolean;
  items: readonly MenuItem[];
  label: string;
  triggerIcon?: LucideIcon;
};

export function Menu({ disabled, items, label, triggerIcon }: MenuProps) {
  const [open, setOpen] = useState(false);
  const menuId = useId();
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const { referenceRef, floatingRef } = useFloatingPosition(open);
  const menuElementRef = useCallback(
    (element: HTMLDivElement | null) => {
      menuRef.current = element;
      floatingRef(element);
    },
    [floatingRef],
  );

  useEffect(() => {
    if (!open) {
      return;
    }

    const closeOnOutsidePress = (event: PointerEvent) => {
      const target = event.target as Node;
      if (
        !menuRef.current?.contains(target) &&
        !triggerRef.current?.contains(target)
      ) {
        setOpen(false);
      }
    };

    document.addEventListener('pointerdown', closeOnOutsidePress);
    return () =>
      document.removeEventListener('pointerdown', closeOnOutsidePress);
  }, [open]);

  const focusItem = (index: number) => {
    const enabledItems = menuRef.current?.querySelectorAll<HTMLElement>(
      '[role="menuitem"]:not([aria-disabled="true"])',
    );
    enabledItems?.[index]?.focus();
  };

  const handleMenuKeyDown = (
    event: JSX.TargetedKeyboardEvent<HTMLDivElement>,
  ) => {
    const enabledItems = Array.from(
      menuRef.current?.querySelectorAll<HTMLElement>(
        '[role="menuitem"]:not([aria-disabled="true"])',
      ) ?? [],
    );
    const currentIndex = enabledItems.indexOf(
      document.activeElement as HTMLElement,
    );

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      enabledItems[(currentIndex + 1) % enabledItems.length]?.focus();
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      enabledItems[
        (currentIndex - 1 + enabledItems.length) % enabledItems.length
      ]?.focus();
    } else if (event.key === 'Home') {
      event.preventDefault();
      enabledItems[0]?.focus();
    } else if (event.key === 'End') {
      event.preventDefault();
      enabledItems.at(-1)?.focus();
    } else if (event.key === 'Escape') {
      event.preventDefault();
      setOpen(false);
      triggerRef.current?.focus();
    }
  };

  return (
    <>
      <span ref={referenceRef} class="ui-menu__anchor">
        <Button
          ref={triggerRef}
          icon={triggerIcon}
          disabled={disabled}
          aria-expanded={open}
          aria-controls={open ? menuId : undefined}
          aria-haspopup="menu"
          onClick={() => {
            setOpen((value) => !value);
            requestAnimationFrame(() => focusItem(0));
          }}
        >
          {label}
          <ChevronDown aria-hidden="true" class="ui-menu__chevron" />
        </Button>
      </span>
      {open
        ? createPortal(
            <div
              ref={menuElementRef}
              id={menuId}
              class="ui-menu"
              role="menu"
              onKeyDown={handleMenuKeyDown}
            >
              {items.map((item) => {
                const Icon = item.icon;
                return (
                  <button
                    key={item.id}
                    type="button"
                    class={`ui-menu__item${item.danger ? ' ui-menu__item--danger' : ''}`}
                    role="menuitem"
                    tabIndex={-1}
                    disabled={item.disabled}
                    aria-disabled={item.disabled || undefined}
                    onClick={() => {
                      if (item.disabled) return;
                      item.onSelect();
                      setOpen(false);
                      triggerRef.current?.focus();
                    }}
                  >
                    {Icon ? <Icon aria-hidden="true" /> : null}
                    <span>{item.label}</span>
                  </button>
                );
              })}
            </div>,
            document.body,
          )
        : null}
    </>
  );
}
