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

type ButtonLinkProps = Omit<
  JSX.AnchorHTMLAttributes<HTMLAnchorElement>,
  'size'
> & {
  children?: ComponentChildren;
  icon?: LucideIcon;
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
        class={buttonClass(variant, size, className)}
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

export const ButtonLink = forwardRef<HTMLAnchorElement, ButtonLinkProps>(
  function ButtonLink(
    {
      children,
      class: className,
      icon: Icon,
      size = 'medium',
      variant = 'secondary',
      ...props
    },
    ref,
  ) {
    return (
      <a {...props} ref={ref} class={buttonClass(variant, size, className)}>
        {Icon ? <Icon aria-hidden="true" /> : null}
        {children ? <span class="ui-button__content">{children}</span> : null}
      </a>
    );
  },
);

function buttonClass(
  variant: NonNullable<ButtonProps['variant']>,
  size: NonNullable<ButtonProps['size']>,
  className: ButtonProps['class'],
) {
  return `ui-button ui-button--${variant} ui-button--${size}${className ? ` ${className}` : ''}`;
}
