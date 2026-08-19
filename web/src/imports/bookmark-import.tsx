import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { Button, Drawer, FileInput, useToast } from '../ui/primitives';
import {
  importBookmarks,
  previewBookmarks,
  type DuplicateStrategy,
  type ImportPreview,
  type ImportVisibility,
} from './bookmark-import-api';

type Props = {
  onClose: () => void;
  onImported: () => void;
  open: boolean;
};

export function BookmarkImportDrawer({ onClose, onImported, open }: Props) {
  const [file, setFile] = useState<File | null>(null);
  const [visibility, setVisibility] = useState<ImportVisibility>('private');
  const [duplicates, setDuplicates] = useState<DuplicateStrategy>('skip');
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [busy, setBusy] = useState<'preview' | 'commit' | null>(null);
  const [error, setError] = useState<'preview' | 'commit' | null>(null);
  const toast = useToast();
  useEffect(() => {
    if (open) {
      setFile(null);
      setVisibility('private');
      setDuplicates('skip');
      setPreview(null);
      setError(null);
    }
  }, [open]);
  const createPreview = async () => {
    if (!file || busy) return;
    setBusy('preview');
    setError(null);
    try {
      setPreview(await previewBookmarks(file, visibility));
    } catch {
      setPreview(null);
      setError('preview');
    } finally {
      setBusy(null);
    }
  };
  const commit = async () => {
    if (!file || !preview || busy) return;
    setBusy('commit');
    setError(null);
    try {
      const result = await importBookmarks(file, visibility, duplicates);
      toast.notify({
        tone: 'success',
        title: messages.imports.imported,
        message: messages.imports.importedSummary(
          result.createdLinks,
          result.updatedLinks,
          result.skippedLinks,
        ),
      });
      onImported();
    } catch {
      setError('commit');
    } finally {
      setBusy(null);
    }
  };
  return (
    <Drawer
      open={open}
      title={messages.imports.title}
      onClose={busy ? () => undefined : onClose}
    >
      <div class="bookmark-import">
        <p class="bookmark-import__intro">{messages.imports.description}</p>
        <div class="bookmark-import__field">
          <FileInput
            id="bookmark-import-file"
            accept=".html,.htm,.json,text/html,application/json"
            label={messages.imports.file}
            disabled={busy !== null}
            error={
              error === 'preview' ? messages.imports.previewFailed : undefined
            }
            onFilesChange={(files) => {
              const selected = files[0] ?? null;
              setFile(selected);
              setPreview(null);
              setError(null);
            }}
          />
          <small>{messages.imports.fileDescription}</small>
        </div>
        <label class="bookmark-import__field">
          <span>{messages.imports.visibility}</span>
          <select
            value={visibility}
            disabled={busy !== null}
            onChange={(event) => {
              setVisibility(event.currentTarget.value as ImportVisibility);
              setPreview(null);
            }}
          >
            <option value="private">{messages.imports.private}</option>
            <option value="public">{messages.imports.public}</option>
          </select>
        </label>
        <Button
          variant="primary"
          loading={busy === 'preview'}
          disabled={!file || busy !== null}
          onClick={() => void createPreview()}
        >
          {messages.imports.preview}
        </Button>
        {preview ? <ImportPreviewDetails preview={preview} /> : null}
        {preview ? (
          <label class="bookmark-import__field">
            <span>{messages.imports.duplicates}</span>
            <select
              value={duplicates}
              disabled={busy !== null}
              onChange={(event) =>
                setDuplicates(event.currentTarget.value as DuplicateStrategy)
              }
            >
              <option value="skip">{messages.imports.skip}</option>
              <option value="update">{messages.imports.update}</option>
              <option value="create">{messages.imports.create}</option>
            </select>
          </label>
        ) : null}
        {error === 'commit' ? (
          <p class="bookmark-import__error" role="alert">
            {messages.imports.importFailed}
          </p>
        ) : null}
        <div class="bookmark-import__actions">
          <Button disabled={busy !== null} onClick={onClose}>
            {messages.imports.cancel}
          </Button>
          <Button
            variant="primary"
            loading={busy === 'commit'}
            disabled={!preview || busy !== null}
            onClick={() => void commit()}
          >
            {messages.imports.commit}
          </Button>
        </div>
      </div>
    </Drawer>
  );
}

function ImportPreviewDetails({ preview }: { preview: ImportPreview }) {
  return (
    <section
      class="bookmark-preview"
      aria-label={messages.imports.previewTitle}
    >
      <div class="bookmark-preview__heading">
        <h3>{messages.imports.previewTitle}</h3>
        <span>
          {preview.format === 'html'
            ? messages.imports.formatHTML
            : messages.imports.formatJSON}
        </span>
      </div>
      <dl class="bookmark-preview__summary">
        <div>
          <dt>{messages.imports.newLinks}</dt>
          <dd>{preview.summary.new}</dd>
        </div>
        <div>
          <dt>{messages.imports.duplicateLinks}</dt>
          <dd>{preview.summary.duplicates}</dd>
        </div>
        <div>
          <dt>{messages.imports.invalidLinks}</dt>
          <dd>{preview.summary.invalid}</dd>
        </div>
      </dl>
      <div class="bookmark-preview__categories">
        {preview.categories.map((category) => (
          <section key={`${category.name}-${category.existingId ?? 'new'}`}>
            <header>
              <strong>{category.name}</strong>
              <span>
                {category.mapping === 'existing'
                  ? messages.imports.existingCategory
                  : messages.imports.newCategory}
              </span>
            </header>
            <ul>
              {category.links.map((link, index) => (
                <li key={`${link.url}-${index}`} data-status={link.status}>
                  <span>
                    <strong>{link.name || link.url}</strong>
                    <small>{link.url}</small>
                  </span>
                  <small>
                    {link.status === 'duplicate'
                      ? messages.imports.duplicateOf(link.duplicateOf ?? '')
                      : link.status === 'invalid'
                        ? messages.imports.invalidReason
                        : messages.imports.newLinks}
                  </small>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </section>
  );
}
