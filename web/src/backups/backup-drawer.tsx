import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { Archive, Download, Plus, Trash2 } from '../ui/icons/interface-icons';
import { Button, Dialog, Drawer, EmptyState, useToast } from '../ui/primitives';
import {
  createBackup,
  deleteBackup,
  downloadBackup,
  listBackups,
  type BackupInfo,
} from './backup-api';
import { RestorePanel } from './restore-panel';

type Props = {
  onClose: () => void;
  onRestored: () => void;
  open: boolean;
};

export function BackupDrawer({ onClose, onRestored, open }: Props) {
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<BackupInfo | null>(null);
  const [error, setError] = useState<'load' | 'create' | null>(null);
  const toast = useToast();
  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      setBackups(await listBackups());
    } catch {
      setError('load');
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    if (open) void load();
  }, [open]);
  const create = async () => {
    if (creating) return;
    setCreating(true);
    setError(null);
    try {
      const created = await createBackup();
      setBackups((current) => [created, ...current]);
      toast.notify({
        tone: 'success',
        title: messages.backups.created,
        message: messages.backups.createdDescription,
      });
    } catch {
      setError('create');
    } finally {
      setCreating(false);
    }
  };
  const busy = creating;
  return (
    <>
      <Drawer
        open={open}
        title={messages.backups.title}
        onClose={busy ? () => undefined : onClose}
      >
        <div class="backup-manager">
          <p>{messages.backups.description}</p>
          <Button
            icon={Plus}
            variant="primary"
            loading={creating}
            disabled={loading}
            onClick={() => void create()}
          >
            {messages.backups.create}
          </Button>
          {error ? (
            <p class="backup-manager__error" role="alert">
              {error === 'load'
                ? messages.backups.loadFailed
                : messages.backups.createFailed}
            </p>
          ) : null}
          {loading ? (
            <EmptyState
              state="loading"
              title={messages.backups.loading}
              description=""
            />
          ) : backups.length === 0 ? (
            <EmptyState
              icon={Archive}
              title={messages.backups.empty}
              description=""
            />
          ) : (
            <ul class="backup-list">
              {backups.map((item) => (
                <li key={item.name}>
                  <div>
                    <strong>{formatBackupDate(item.createdAt)}</strong>
                    <small>{messages.backups.size(item.size)}</small>
                  </div>
                  <span>
                    <Button
                      icon={Download}
                      size="small"
                      aria-label={messages.backups.download}
                      disabled={busy}
                      onClick={() => downloadBackup(item.name)}
                    />
                    <Button
                      icon={Trash2}
                      size="small"
                      variant="ghost"
                      aria-label={messages.backups.delete}
                      disabled={busy}
                      onClick={() => setDeleting(item)}
                    />
                  </span>
                </li>
              ))}
            </ul>
          )}
          <RestorePanel onRestored={onRestored} />
          <div class="bookmark-import__actions">
            <Button disabled={busy} onClick={onClose}>
              {messages.backups.close}
            </Button>
          </div>
        </div>
      </Drawer>
      <BackupDeleteDialog
        backup={deleting}
        onClose={() => setDeleting(null)}
        onDeleted={(name) => {
          setBackups((current) => current.filter((item) => item.name !== name));
          setDeleting(null);
        }}
      />
    </>
  );
}

function BackupDeleteDialog({
  backup,
  onClose,
  onDeleted,
}: {
  backup: BackupInfo | null;
  onClose: () => void;
  onDeleted: (name: string) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);
  useEffect(() => {
    if (backup) {
      setBusy(false);
      setError(false);
    }
  }, [backup]);
  if (!backup) return null;
  const confirm = async () => {
    setBusy(true);
    setError(false);
    try {
      await deleteBackup(backup.name);
      onDeleted(backup.name);
    } catch {
      setError(true);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog
      open
      title={messages.backups.deleteTitle}
      description={messages.backups.deleteDescription}
      onClose={busy ? () => undefined : onClose}
      footer={
        <>
          <Button disabled={busy} onClick={onClose}>
            {messages.backups.close}
          </Button>
          <Button
            variant="danger"
            loading={busy}
            onClick={() => void confirm()}
          >
            {messages.backups.confirmDelete}
          </Button>
        </>
      }
    >
      {error ? (
        <p class="backup-manager__error" role="alert">
          {messages.backups.deleteFailed}
        </p>
      ) : (
        <code class="backup-delete__name">{backup.name}</code>
      )}
    </Dialog>
  );
}

function formatBackupDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN');
}
