import type { ComponentChildren } from 'preact';

import {
  CircleAlert,
  LoaderCircle,
  type LucideIcon,
} from '../icons/interface-icons';

type EmptyStateProps = {
  action?: ComponentChildren;
  description: string;
  icon?: LucideIcon;
  state?: 'empty' | 'loading' | 'error';
  title: string;
};

export function EmptyState({
  action,
  description,
  icon: Icon,
  state = 'empty',
  title,
}: EmptyStateProps) {
  const StateIcon =
    state === 'loading' ? LoaderCircle : state === 'error' ? CircleAlert : Icon;

  return (
    <div
      class={`ui-empty ui-empty--${state}`}
      aria-busy={state === 'loading' || undefined}
    >
      {StateIcon ? (
        <span class="ui-empty__icon" aria-hidden="true">
          <StateIcon
            class={state === 'loading' ? 'ui-empty__spinner' : undefined}
          />
        </span>
      ) : null}
      <h2>{title}</h2>
      <p>{description}</p>
      {action ? <div class="ui-empty__action">{action}</div> : null}
    </div>
  );
}
