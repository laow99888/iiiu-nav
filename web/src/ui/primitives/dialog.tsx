import type { ComponentChildren } from 'preact';
import { useId } from 'preact/hooks';

import { messages } from '../../i18n/messages';
import { X } from '../icons/interface-icons';
import { Button } from './button';
import { ModalSurface } from './modal-surface';

type DialogProps = {
  children: ComponentChildren;
  description?: string;
  footer?: ComponentChildren;
  onClose: () => void;
  open: boolean;
  title: string;
};

export function Dialog({
  children,
  description,
  footer,
  onClose,
  open,
  title,
}: DialogProps) {
  const titleId = useId();

  if (!open) {
    return null;
  }

  return (
    <ModalSurface class="ui-dialog" labelledBy={titleId} onClose={onClose}>
      <header class="ui-modal__header">
        <div>
          <h2 id={titleId}>{title}</h2>
          {description ? <p>{description}</p> : null}
        </div>
        <Button
          variant="ghost"
          size="small"
          icon={X}
          aria-label={messages.ui.closeDialog}
          onClick={onClose}
        />
      </header>
      <div class="ui-modal__body">{children}</div>
      {footer ? <footer class="ui-modal__footer">{footer}</footer> : null}
    </ModalSurface>
  );
}
