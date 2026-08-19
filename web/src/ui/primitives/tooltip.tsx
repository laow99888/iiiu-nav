import { createPortal } from 'preact/compat';
import type { ComponentChildren } from 'preact';
import { useId, useState } from 'preact/hooks';

import { useFloatingPosition } from '../hooks/use-floating-position';

type TooltipProps = {
  children: ComponentChildren;
  content: string;
};

export function Tooltip({ children, content }: TooltipProps) {
  const [open, setOpen] = useState(false);
  const tooltipId = useId();
  const { referenceRef, floatingRef } = useFloatingPosition(open, 'top');

  return (
    <>
      <span
        ref={referenceRef}
        class="ui-tooltip__anchor"
        aria-describedby={open ? tooltipId : undefined}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
      >
        {children}
      </span>
      {open
        ? createPortal(
            <span
              ref={floatingRef}
              id={tooltipId}
              class="ui-tooltip"
              role="tooltip"
            >
              {content}
            </span>,
            document.body,
          )
        : null}
    </>
  );
}
