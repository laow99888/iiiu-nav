import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationCategory, NavigationLink } from '../navigation/types';
import { LinkLogo } from '../navigation/link-logo';
import { RefreshCw } from '../ui/icons/interface-icons';
import { Button, Drawer, FileInput, TextField } from '../ui/primitives';
import {
  createLink,
  recognizeLink,
  updateLink,
  uploadLogo,
  type LinkInput,
} from './link-api';
import { LinkDeleteDialog } from './link-delete-dialog';

type Props = {
  categories: readonly NavigationCategory[];
  categoryID: string;
  link?: NavigationLink | null;
  onClose: () => void;
  onSaved: () => void;
  open: boolean;
};

export function LinkEditor({
  categories,
  categoryID,
  link = null,
  onClose,
  onSaved,
  open,
}: Props) {
  const [selectedCategoryID, setSelectedCategoryID] = useState(categoryID);
  const [url, setURL] = useState(link?.url ?? '');
  const [name, setName] = useState(link?.name ?? '');
  const [description, setDescription] = useState(link?.description ?? '');
  const [logoText, setLogoText] = useState(link?.logoText ?? '');
  const [iconSource, setIconSource] = useState<LinkInput['iconSource']>(
    link?.logoSource ?? 'generated',
  );
  const [iconValue, setIconValue] = useState(link?.logoValue ?? '');
  const [saving, setSaving] = useState(false);
  const [recognizing, setRecognizing] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<'save' | 'recognize' | 'upload' | null>(
    null,
  );
  const [confirmDelete, setConfirmDelete] = useState(false);
  const validURL = isSupportedURL(url);
  const valid =
    selectedCategoryID !== '' &&
    validURL &&
    name.trim() !== '' &&
    Array.from(logoText.trim()).length <= 3;
  const busy = saving || recognizing || uploading;
  const recognize = async () => {
    if (!validURL || busy) return;
    setRecognizing(true);
    setError(null);
    try {
      const result = await recognizeLink(url.trim());
      setName(result.name);
      setDescription(result.description);
      setIconSource(result.iconSource);
      setIconValue(result.iconValue);
      if (result.iconSource === 'generated') setLogoText(result.iconValue);
    } catch {
      setError('recognize');
    } finally {
      setRecognizing(false);
    }
  };
  const upload = async (files: File[]) => {
    const file = files[0];
    if (!file || busy) return;
    if (file.size > 2 * 1024 * 1024) {
      setError('upload');
      return;
    }
    setUploading(true);
    setError(null);
    try {
      const value = await uploadLogo(file);
      setIconSource('upload');
      setIconValue(value);
    } catch {
      setError('upload');
    } finally {
      setUploading(false);
    }
  };
  const save = async () => {
    if (!valid || saving) return;
    setSaving(true);
    setError(null);
    const input = {
      categoryId: selectedCategoryID,
      url: url.trim(),
      name: name.trim(),
      description: description.trim(),
      logoText: logoText.trim(),
      iconSource,
      iconValue: iconSource === 'generated' ? logoText.trim() : iconValue,
    };
    try {
      if (link) await updateLink(link.id, input);
      else await createLink(input);
      onSaved();
    } catch {
      setError('save');
    } finally {
      setSaving(false);
    }
  };
  return (
    <>
      <Drawer
        open={open}
        title={link ? messages.links.edit : messages.links.create}
        onClose={busy ? () => undefined : onClose}
      >
        <form
          class="link-form"
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <label class="link-form__field">
            <span>{messages.links.category}</span>
            <select
              value={selectedCategoryID}
              disabled={busy}
              onChange={(event) =>
                setSelectedCategoryID(event.currentTarget.value)
              }
            >
              {categories.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.name}
                  {category.visibility === 'private'
                    ? ` · ${messages.navigation.privateCategory}`
                    : ''}
                </option>
              ))}
            </select>
          </label>
          <TextField
            id="link-url"
            autoFocus
            required
            type="url"
            label={messages.links.url}
            value={url}
            disabled={busy}
            error={url && !validURL ? messages.links.invalidURL : undefined}
            onInput={(event) => setURL(event.currentTarget.value)}
          />
          <Button
            class="link-form__recognize"
            icon={RefreshCw}
            loading={recognizing}
            disabled={!validURL || saving || uploading}
            onClick={() => void recognize()}
          >
            {link ? messages.links.refreshMetadata : messages.links.recognize}
          </Button>
          <TextField
            id="link-name"
            required
            maxLength={120}
            label={messages.links.name}
            value={name}
            disabled={busy}
            onInput={(event) => setName(event.currentTarget.value)}
          />
          <TextField
            id="link-description"
            maxLength={300}
            label={messages.links.description}
            value={description}
            disabled={busy}
            onInput={(event) => setDescription(event.currentTarget.value)}
          />
          <TextField
            id="link-logo-text"
            maxLength={3}
            label={messages.links.logoText}
            description={messages.links.logoTextDescription}
            value={logoText}
            disabled={busy}
            onInput={(event) => {
              const value = event.currentTarget.value;
              setLogoText(value);
              setIconSource('generated');
              setIconValue(value);
            }}
          />
          <div class="link-form__logo">
            <LinkLogo
              label={name || messages.links.preview}
              text={logoText.trim() || Array.from(name.trim())[0] || '?'}
              tone={link?.logoTone ?? 'ink'}
              url={iconSource === 'generated' ? null : iconValue}
            />
            <div>
              <FileInput
                id="link-logo-file"
                accept=".png,.jpg,.jpeg,.webp,.ico,image/png,image/jpeg,image/webp,image/x-icon"
                label={messages.links.uploadLogo}
                loading={uploading}
                disabled={saving || recognizing}
                error={
                  error === 'upload' ? messages.links.uploadFailed : undefined
                }
                onFilesChange={(files) => void upload(files)}
              />
              {iconSource !== 'generated' ? (
                <Button
                  variant="ghost"
                  size="small"
                  disabled={busy}
                  onClick={() => {
                    setIconSource('generated');
                    setIconValue(logoText);
                  }}
                >
                  {messages.links.useTextLogo}
                </Button>
              ) : null}
            </div>
          </div>
          {error === 'recognize' ? (
            <p class="link-form__error" role="alert">
              {messages.links.recognitionFailed}
            </p>
          ) : null}
          {error === 'save' ? (
            <p class="link-form__error" role="alert">
              {messages.links.saveFailed}
            </p>
          ) : null}
          <div class="link-form__actions">
            {link ? (
              <Button
                variant="danger"
                disabled={busy}
                onClick={() => setConfirmDelete(true)}
              >
                {messages.links.delete}
              </Button>
            ) : (
              <span />
            )}
            <span>
              <Button disabled={busy} onClick={onClose}>
                {messages.links.cancel}
              </Button>
              <Button
                variant="primary"
                loading={saving}
                disabled={
                  !valid ||
                  recognizing ||
                  uploading ||
                  (iconSource !== 'generated' && !iconValue)
                }
                onClick={() => void save()}
              >
                {messages.links.save}
              </Button>
            </span>
          </div>
        </form>
      </Drawer>
      <LinkDeleteDialog
        link={confirmDelete ? link : null}
        onClose={() => setConfirmDelete(false)}
        onDeleted={onSaved}
      />
    </>
  );
}

function isSupportedURL(value: string) {
  try {
    const parsed = new URL(value.trim());
    return (
      (parsed.protocol === 'http:' || parsed.protocol === 'https:') &&
      Boolean(parsed.hostname) &&
      !parsed.username &&
      !parsed.password
    );
  } catch {
    return false;
  }
}
