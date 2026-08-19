import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { Button, Dialog, TextField } from '../ui/primitives';
import { AuthAPIError, login } from './auth-api';

type LoginDialogProps = {
  onClose: () => void;
  onSuccess: () => void;
  open: boolean;
};

export function LoginDialog({ onClose, onSuccess, open }: LoginDialogProps) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const close = () => {
    if (submitting) return;
    setPassword('');
    setError('');
    onClose();
  };

  const submit = async () => {
    if (!password || submitting) return;
    setSubmitting(true);
    setError('');
    try {
      await login(password);
      setPassword('');
      onSuccess();
      onClose();
    } catch (cause) {
      setError(loginErrorMessage(cause));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      title={messages.auth.loginTitle}
      description={messages.auth.loginDescription}
      onClose={close}
      footer={
        <>
          <Button disabled={submitting} onClick={close}>
            {messages.auth.cancel}
          </Button>
          <Button
            variant="primary"
            loading={submitting}
            disabled={!password}
            onClick={() => void submit()}
          >
            {messages.auth.login}
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
          label={messages.auth.password}
          value={password}
          disabled={submitting}
          error={error || undefined}
          onInput={(event) => setPassword(event.currentTarget.value)}
        />
        <button class="auth-form__submit" type="submit" tabIndex={-1} />
      </form>
    </Dialog>
  );
}

function loginErrorMessage(cause: unknown) {
  if (cause instanceof AuthAPIError) {
    if (cause.code === 'invalid_credentials') {
      return messages.auth.invalidCredentials;
    }
    if (cause.code === 'login_rate_limited') {
      return cause.retryAfter
        ? messages.auth.rateLimitedSeconds(cause.retryAfter)
        : messages.auth.rateLimited;
    }
  }
  return messages.auth.loginFailed;
}
