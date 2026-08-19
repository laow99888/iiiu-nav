import type { ComponentChildren } from 'preact';
import { useId } from 'preact/hooks';

import { messages } from '../../i18n/messages';
import { X } from '../icons/interface-icons';
import { Button } from './button';
import { ModalSurface } from './modal-surface';

type DrawerProps = {
  children: ComponentChildren;
  footer?: ComponentChildren;
  onClose: () => void;
  open: boolean;
  side?: 'left' | 'right';
  title: string;
};

export function Drawer({
  children,
  footer,
  onClose,
  open,
  side = 'right',
  title,
}: DrawerProps) {
  const titleId = useId();

  if (!open) {
    return null;
  }

  return (
    <ModalSurface
      class={`ui-drawer ui-drawer--${side}`}
      labelledBy={titleId}
      onClose={onClose}
    >
      <header class="ui-modal__header">
        <h2 id={titleId}>{title}</h2>
        <Button
          variant="ghost"
          size="small"
          icon={X}
          aria-label={messages.ui.closeDrawer}
          onClick={onClose}
        />
      </header>
      <div class="ui-modal__body">{children}</div>
      {footer ? <footer class="ui-modal__footer">{footer}</footer> : null}
    </ModalSurface>
  );
}
