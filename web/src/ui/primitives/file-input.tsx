import type { JSX } from 'preact';
import { useId, useState } from 'preact/hooks';

import { messages } from '../../i18n/messages';
import { LoaderCircle, Upload } from '../icons/interface-icons';

type FileInputProps = Omit<
  JSX.InputHTMLAttributes<HTMLInputElement>,
  'type' | 'onChange'
> & {
  error?: string;
  label: string;
  loading?: boolean;
  onFilesChange: (files: File[]) => void;
};

export function FileInput({
  class: className,
  disabled,
  error,
  id,
  label,
  loading = false,
  multiple,
  onFilesChange,
  ...props
}: FileInputProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const errorId = error ? `${inputId}-error` : undefined;
  const [files, setFiles] = useState<File[]>([]);
  const isDisabled = disabled || loading;

  return (
    <div
      class={`ui-file${error ? ' ui-file--error' : ''}${className ? ` ${className}` : ''}`}
    >
      <input
        {...props}
        class="sr-only ui-file__input"
        id={inputId}
        type="file"
        multiple={multiple}
        disabled={isDisabled}
        aria-invalid={error ? 'true' : undefined}
        aria-describedby={errorId}
        onChange={(event) => {
          const selected = Array.from(event.currentTarget.files ?? []);
          setFiles(selected);
          onFilesChange(selected);
        }}
      />
      <label
        class="ui-file__label"
        for={inputId}
        aria-disabled={isDisabled || undefined}
      >
        {loading ? (
          <LoaderCircle class="ui-file__spinner" aria-hidden="true" />
        ) : (
          <Upload aria-hidden="true" />
        )}
        <span>{label}</span>
      </label>
      {files.length > 0 ? (
        <span class="ui-file__selection">
          {files.length === 1
            ? files[0]?.name
            : messages.ui.selectedFiles(files.length)}
        </span>
      ) : null}
      {error ? (
        <span class="ui-field__error" id={errorId} role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}
