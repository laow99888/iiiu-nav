import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { Button, Dialog, TextField } from '../ui/primitives';
import { AuthAPIError, changePassword } from './auth-api';

type PasswordDialogProps = {
  onClose: () => void;
  onSuccess: () => void;
  open: boolean;
};

export function PasswordDialog({
  onClose,
  onSuccess,
  open,
}: PasswordDialogProps) {
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const reset = () => {
    setCurrentPassword('');
    setNewPassword('');
    setConfirmation('');
    setError('');
  };
  const close = () => {
    if (submitting) return;
    reset();
    onClose();
  };

  const mismatch = confirmation.length > 0 && confirmation !== newPassword;
  const valid =
    currentPassword.length > 0 &&
    newPassword.length >= 12 &&
    confirmation === newPassword;
  const submit = async () => {
    if (!valid || submitting) return;
    setSubmitting(true);
    setError('');
    try {
      await changePassword(currentPassword, newPassword);
      reset();
      onSuccess();
      onClose();
    } catch (cause) {
      setError(passwordErrorMessage(cause));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      title={messages.auth.passwordTitle}
      description={messages.auth.passwordDescription}
      onClose={close}
      footer={
        <>
          <Button disabled={submitting} onClick={close}>
            {messages.auth.cancel}
          </Button>
          <Button
            variant="primary"
            loading={submitting}
            disabled={!valid}
            onClick={() => void submit()}
          >
            {messages.auth.changePassword}
          </Button>
        </>
      }
    >
      <form
        class="auth-form"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <TextField
          autoFocus
          required
          type="password"
          autoComplete="current-password"
          label={messages.auth.currentPassword}
          value={currentPassword}
          disabled={submitting}
          error={error || undefined}
          onInput={(event) => setCurrentPassword(event.currentTarget.value)}
        />
        <TextField
          required
          type="password"
          minLength={12}
          autoComplete="new-password"
          label={messages.auth.newPassword}
          description={messages.auth.passwordRule}
          value={newPassword}
          disabled={submitting}
          onInput={(event) => setNewPassword(event.currentTarget.value)}
        />
        <TextField
          required
          type="password"
          autoComplete="new-password"
          label={messages.auth.confirmPassword}
          value={confirmation}
          disabled={submitting}
          error={mismatch ? messages.auth.passwordMismatch : undefined}
          onInput={(event) => setConfirmation(event.currentTarget.value)}
        />
        <button class="auth-form__submit" type="submit" tabIndex={-1} />
      </form>
    </Dialog>
  );
}

function passwordErrorMessage(cause: unknown) {
  if (cause instanceof AuthAPIError) {
    if (cause.code === 'current_password_invalid') {
      return messages.auth.currentPasswordInvalid;
    }
    if (cause.code === 'invalid_password') {
      return messages.auth.invalidPassword;
    }
  }
  return messages.auth.passwordFailed;
}
