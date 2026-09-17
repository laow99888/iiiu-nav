import { useCallback, useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { useAsyncResource } from '../ui/hooks/use-async-resource';
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
  // 列表的加载与重试生命周期由 hook 负责；抽屉关闭时不请求，
  // 重新打开会回到加载态重新拉取。
  const resource = useAsyncResource(
    useCallback(() => listBackups(), []),
    {
      enabled: open,
    },
  );
  // 就绪结果同步进本地列表，让创建/删除保留原有的本地乐观更新。
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  useEffect(() => {
    if (resource.state.kind === 'ready') setBackups(resource.state.value);
  }, [resource.state]);
  const loading = resource.state.kind === 'loading';
  const [creating, setCreating] = useState(false);
  const [createFailed, setCreateFailed] = useState(false);
  const [deleting, setDeleting] = useState<BackupInfo | null>(null);
  const toast = useToast();
  const create = async () => {
    if (creating) return;
    setCreating(true);
    setCreateFailed(false);
    try {
      const created = await createBackup();
      setBackups((current) => [created, ...current]);
      toast.notify({
        tone: 'success',
        title: messages.backups.created,
        message: messages.backups.createdDescription,
      });
    } catch {
      setCreateFailed(true);
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
          {createFailed || resource.state.kind === 'error' ? (
            <p class="backup-manager__error" role="alert">
              {createFailed
                ? messages.backups.createFailed
                : messages.backups.loadFailed}
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
