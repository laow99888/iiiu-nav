import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { SiteSettings } from '../navigation/types';
import {
  Button,
  Drawer,
  FileInput,
  TextField,
  useToast,
} from '../ui/primitives';
import {
  saveSiteSettings,
  uploadBackground,
  uploadFavicon,
  uploadSiteLogo,
} from './site-settings-api';

type Props = {
  initial: SiteSettings;
  onClose: () => void;
  onSaved: () => void;
  open: boolean;
};

type UploadField = 'background' | 'favicon' | 'logo';

export function SiteSettingsDrawer({ initial, onClose, onSaved, open }: Props) {
  const [draft, setDraft] = useState(initial);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState<UploadField | null>(null);
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();
  useEffect(() => {
    if (open) {
      setDraft(initial);
      setError(null);
    }
  }, [initial, open]);
  const busy = saving || uploading !== null;
  const update = <Key extends keyof SiteSettings>(
    key: Key,
    value: SiteSettings[Key],
  ) => setDraft((current) => ({ ...current, [key]: value }));
  const upload = async (field: UploadField, files: File[]) => {
    const file = files[0];
    if (!file || busy) return;
    setUploading(field);
    setError(null);
    try {
      const url =
        field === 'logo'
          ? await uploadSiteLogo(file)
          : field === 'favicon'
            ? await uploadFavicon(file)
            : await uploadBackground(file);
      update(
        field === 'logo'
          ? 'logoUrl'
          : field === 'favicon'
            ? 'faviconUrl'
            : 'backgroundUrl',
        url,
      );
    } catch {
      setError(field);
    } finally {
      setUploading(null);
    }
  };
  const save = async () => {
    if (busy || !draft.name.trim()) return;
    setSaving(true);
    setError(null);
    try {
      await saveSiteSettings({ ...draft, name: draft.name.trim() });
      toast.notify({
        tone: 'success',
        title: messages.settings.saved,
        message: messages.settings.savedDescription,
      });
      onSaved();
    } catch {
      setError('save');
    } finally {
      setSaving(false);
    }
  };
  return (
    <Drawer
      open={open}
      title={messages.settings.title}
      onClose={busy ? () => undefined : onClose}
    >
      <form
        class="site-settings"
        onSubmit={(event) => {
          event.preventDefault();
          void save();
        }}
      >
        <section class="site-settings__section">
          <h3>{messages.settings.identity}</h3>
          <TextField
            id="site-name"
            autoFocus
            required
            maxLength={80}
            label={messages.settings.name}
            value={draft.name}
            disabled={busy}
            onInput={(event) => update('name', event.currentTarget.value)}
          />
          <ImageSetting
            id="site-logo"
            label={messages.settings.logo}
            description={messages.settings.logoDescription}
            url={draft.logoUrl}
            accept=".png,.jpg,.jpeg,.webp,.ico,image/png,image/jpeg,image/webp,image/x-icon"
            loading={uploading === 'logo'}
            disabled={busy && uploading !== 'logo'}
            error={error === 'logo' ? messages.settings.logoInvalid : undefined}
            onClear={() => update('logoUrl', '')}
            onFiles={(files) => void upload('logo', files)}
          />
          <ImageSetting
            id="site-favicon"
            label={messages.settings.favicon}
            description={messages.settings.faviconDescription}
            url={draft.faviconUrl}
            accept=".png,.ico,image/png,image/x-icon"
            loading={uploading === 'favicon'}
            disabled={busy && uploading !== 'favicon'}
            error={
              error === 'favicon' ? messages.settings.faviconInvalid : undefined
            }
            onClear={() => update('faviconUrl', '')}
            onFiles={(files) => void upload('favicon', files)}
          />
        </section>
        <section class="site-settings__section">
          <h3>{messages.settings.appearance}</h3>
          <label class="site-settings__color">
            <span>{messages.settings.accent}</span>
            <span>
              <input
                type="color"
                value={draft.accentColor}
                disabled={busy}
                aria-label={messages.settings.accent}
                onInput={(event) =>
                  update('accentColor', event.currentTarget.value)
                }
              />
              <code>{draft.accentColor}</code>
            </span>
          </label>
          <ImageSetting
            id="site-background"
            label={messages.settings.background}
            description={messages.settings.backgroundDescription}
            url={draft.backgroundUrl}
            accept=".png,.jpg,.jpeg,.webp,image/png,image/jpeg,image/webp"
            loading={uploading === 'background'}
            disabled={busy && uploading !== 'background'}
            error={
              error === 'background'
                ? messages.settings.backgroundInvalid
                : undefined
            }
            wide
            onClear={() => update('backgroundUrl', '')}
            onFiles={(files) => void upload('background', files)}
          />
          <label class="site-settings__range">
            <span>
              <strong>{messages.settings.overlay}</strong>
              <output>{draft.backgroundOverlay}%</output>
            </span>
            <input
              type="range"
              min="60"
              max="95"
              step="1"
              value={draft.backgroundOverlay}
              disabled={busy || !draft.backgroundUrl}
              onInput={(event) =>
                update('backgroundOverlay', Number(event.currentTarget.value))
              }
            />
          </label>
        </section>
        <section class="site-settings__section">
          <h3>{messages.settings.privacy}</h3>
          <label class="site-settings__toggle">
            <input
              type="checkbox"
              checked={draft.indexingEnabled}
              disabled={busy}
              onChange={(event) =>
                update('indexingEnabled', event.currentTarget.checked)
              }
            />
            <span>
              <strong>{messages.settings.indexing}</strong>
              <small>{messages.settings.indexingDescription}</small>
            </span>
          </label>
        </section>
        {error === 'save' ? (
          <p class="site-settings__error" role="alert">
            {messages.settings.saveFailed}
          </p>
        ) : null}
        <div class="site-settings__actions">
          <Button disabled={busy} onClick={onClose}>
            {messages.settings.cancel}
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={busy || !draft.name.trim()}
            onClick={() => void save()}
          >
            {messages.settings.save}
          </Button>
        </div>
      </form>
    </Drawer>
  );
}

function ImageSetting({
  accept,
  description,
  disabled,
  error,
  id,
  label,
  loading,
  onClear,
  onFiles,
  url,
  wide = false,
}: {
  accept: string;
  description: string;
  disabled: boolean;
  error?: string;
  id: string;
  label: string;
  loading: boolean;
  onClear: () => void;
  onFiles: (files: File[]) => void;
  url: string;
  wide?: boolean;
}) {
  return (
    <div class="site-settings__image">
      <span class="site-settings__label">{label}</span>
      <small>{description}</small>
      {url ? (
        <div class={`site-settings__preview${wide ? ' is-wide' : ''}`}>
          <img src={url} alt="" />
          <Button
            variant="ghost"
            size="small"
            disabled={disabled}
            onClick={onClear}
          >
            {messages.settings.clearImage}
          </Button>
        </div>
      ) : null}
      <FileInput
        id={id}
        accept={accept}
        label={
          url ? messages.settings.replaceImage : messages.settings.uploadImage
        }
        loading={loading}
        disabled={disabled}
        error={error}
        onFilesChange={onFiles}
      />
    </div>
  );
}
