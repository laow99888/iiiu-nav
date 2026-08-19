import type { ComponentChildren, JSX } from 'preact';
import { forwardRef } from 'preact/compat';

import { LoaderCircle, type LucideIcon } from '../icons/interface-icons';

type ButtonProps = Omit<
  JSX.ButtonHTMLAttributes<HTMLButtonElement>,
  'icon' | 'size'
> & {
  children?: ComponentChildren;
  icon?: LucideIcon;
  loading?: boolean;
  size?: 'small' | 'medium' | 'large';
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button(
    {
      children,
      class: className,
      disabled,
      icon: Icon,
      loading = false,
      size = 'medium',
      type = 'button',
      variant = 'secondary',
      ...props
    },
    ref,
  ) {
    const isDisabled = Boolean(disabled || loading);

    return (
      <button
        {...props}
        ref={ref}
        type={type}
        class={`ui-button ui-button--${variant} ui-button--${size}${className ? ` ${className}` : ''}`}
        disabled={isDisabled}
        aria-busy={loading || undefined}
      >
        {loading ? (
          <LoaderCircle class="ui-button__spinner" aria-hidden="true" />
        ) : Icon ? (
          <Icon aria-hidden="true" />
        ) : null}
        {children ? <span class="ui-button__content">{children}</span> : null}
      </button>
    );
  },
);
