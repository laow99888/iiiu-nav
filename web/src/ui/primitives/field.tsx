import { useId } from 'preact/hooks';
import type { JSX } from 'preact';

type TextFieldProps = Omit<
  JSX.InputHTMLAttributes<HTMLInputElement>,
  'label'
> & {
  description?: string;
  error?: string;
  label: string;
};

export function TextField({
  class: className,
  description,
  disabled,
  error,
  id,
  label,
  required,
  ...props
}: TextFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const descriptionId = description ? `${inputId}-description` : undefined;
  const errorId = error ? `${inputId}-error` : undefined;
  const describedBy =
    [descriptionId, errorId].filter(Boolean).join(' ') || undefined;

  return (
    <div
      class={`ui-field${error ? ' ui-field--error' : ''}${className ? ` ${className}` : ''}`}
    >
      <label class="ui-field__label" for={inputId}>
        {label}
        {required ? <span aria-hidden="true"> *</span> : null}
      </label>
      {description ? (
        <span class="ui-field__description" id={descriptionId}>
          {description}
        </span>
      ) : null}
      <input
        {...props}
        class="ui-input"
        id={inputId}
        disabled={disabled}
        required={required}
        aria-invalid={error ? 'true' : undefined}
        aria-describedby={describedBy}
      />
      {error ? (
        <span class="ui-field__error" id={errorId} role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}
