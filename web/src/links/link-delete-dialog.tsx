import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationLink } from '../navigation/types';
import { Button, Dialog } from '../ui/primitives';
import { deleteLink } from './link-api';

type Props = {
  link: NavigationLink | null;
  onClose: () => void;
  onDeleted: () => void;
};

export function LinkDeleteDialog({ link, onClose, onDeleted }: Props) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState(false);
  if (!link) return null;
  const confirm = async () => {
    setDeleting(true);
    setError(false);
    try {
      await deleteLink(link.id);
      onDeleted();
    } catch {
      setError(true);
    } finally {
      setDeleting(false);
    }
  };
  return (
    <Dialog
      open
      title={messages.links.deleteTitle(link.name)}
      description={messages.links.deleteDescription}
      onClose={deleting ? () => undefined : onClose}
      footer={
        <>
          <Button disabled={deleting} onClick={onClose}>
            {messages.links.cancel}
          </Button>
          <Button
            variant="danger"
            loading={deleting}
            onClick={() => void confirm()}
          >
            {messages.links.confirmDelete}
          </Button>
        </>
      }
    >
      {error ? (
        <p class="link-form__error" role="alert">
          {messages.links.deleteFailed}
        </p>
      ) : (
        <p class="link-delete__url">{link.url}</p>
      )}
    </Dialog>
  );
}
