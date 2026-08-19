import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { Button, FileInput, TextField, useToast } from '../ui/primitives';
import {
  restoreBackup,
  RestoreError,
  type RestoreErrorCode,
} from './restore-api';

type Props = {
  onRestored: () => void;
};

export function RestorePanel({ onRestored }: Props) {
  const [file, setFile] = useState<File | null>(null);
  const [password, setPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [restoring, setRestoring] = useState(false);
  const [error, setError] = useState<RestoreErrorCode | null>(null);
  const toast = useToast();
  const restore = async () => {
    if (!file || !password || confirmation !== 'RESTORE' || restoring) return;
    setRestoring(true);
    setError(null);
    try {
      await restoreBackup(file, password);
      toast.notify({
        tone: 'success',
        title: messages.restore.succeeded,
        message: messages.restore.succeededDescription,
      });
      onRestored();
    } catch (cause) {
      setError(cause instanceof RestoreError ? cause.code : 'restore_failed');
    } finally {
      setRestoring(false);
    }
  };
  return (
    <section class="restore-panel">
      <div>
        <h3>{messages.restore.title}</h3>
        <p>{messages.restore.description}</p>
      </div>
      <div class="bookmark-import__field">
        <FileInput
          id="restore-backup-file"
          accept=".zip,application/zip"
          label={messages.restore.file}
          disabled={restoring}
          onFilesChange={(files) => {
            setFile(files[0] ?? null);
            setError(null);
          }}
        />
        <small>{messages.restore.fileDescription}</small>
      </div>
      <TextField
        id="restore-password"
        type="password"
        label={messages.restore.password}
        value={password}
        disabled={restoring}
        onInput={(event) => setPassword(event.currentTarget.value)}
      />
      <TextField
        id="restore-confirmation"
        label={messages.restore.confirmation}
        description={messages.restore.confirmationDescription}
        value={confirmation}
        disabled={restoring}
        onInput={(event) => setConfirmation(event.currentTarget.value)}
      />
      {error ? (
        <p class="backup-manager__error" role="alert">
          {restoreErrorMessage(error)}
        </p>
      ) : null}
      <Button
        variant="danger"
        loading={restoring}
        disabled={!file || !password || confirmation !== 'RESTORE'}
        onClick={() => void restore()}
      >
        {messages.restore.action}
      </Button>
    </section>
  );
}

function restoreErrorMessage(code: RestoreErrorCode) {
  switch (code) {
    case 'restore_password_invalid':
      return messages.restore.passwordInvalid;
    case 'restore_archive_invalid':
      return messages.restore.invalid;
    case 'restore_archive_incompatible':
      return messages.restore.incompatible;
    case 'restore_archive_too_large':
      return messages.restore.tooLarge;
    case 'restore_space_insufficient':
      return messages.restore.noSpace;
    default:
      return messages.restore.failed;
  }
}
