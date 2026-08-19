import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { FileDown } from '../ui/icons/interface-icons';
import { Button, Drawer } from '../ui/primitives';
import {
  downloadBookmarkExport,
  type ExportFormat,
  type ExportScope,
} from './bookmark-export-api';

type Props = {
  onClose: () => void;
  open: boolean;
};

export function BookmarkExportDrawer({ onClose, open }: Props) {
  const [scope, setScope] = useState<ExportScope>('all');
  const [downloading, setDownloading] = useState<ExportFormat | null>(null);
  const [error, setError] = useState(false);
  const download = async (format: ExportFormat) => {
    if (downloading) return;
    setDownloading(format);
    setError(false);
    try {
      await downloadBookmarkExport(format, scope);
    } catch {
      setError(true);
    } finally {
      setDownloading(null);
    }
  };
  return (
    <Drawer
      open={open}
      title={messages.exports.title}
      onClose={downloading ? () => undefined : onClose}
    >
      <div class="bookmark-export">
        <p>{messages.exports.description}</p>
        <label class="bookmark-import__field">
          <span>{messages.exports.scope}</span>
          <select
            value={scope}
            disabled={downloading !== null}
            onChange={(event) =>
              setScope(event.currentTarget.value as ExportScope)
            }
          >
            <option value="all">{messages.exports.all}</option>
            <option value="public">{messages.exports.public}</option>
            <option value="private">{messages.exports.private}</option>
          </select>
        </label>
        <ExportChoice
          title={messages.exports.html}
          description={messages.exports.htmlDescription}
          loading={downloading === 'html'}
          disabled={downloading !== null}
          onDownload={() => void download('html')}
        />
        <ExportChoice
          title={messages.exports.json}
          description={messages.exports.jsonDescription}
          loading={downloading === 'json'}
          disabled={downloading !== null}
          onDownload={() => void download('json')}
        />
        {error ? (
          <p class="bookmark-import__error" role="alert">
            {messages.exports.failed}
          </p>
        ) : null}
        <div class="bookmark-import__actions">
          <Button disabled={downloading !== null} onClick={onClose}>
            {messages.exports.close}
          </Button>
        </div>
      </div>
    </Drawer>
  );
}

function ExportChoice({
  description,
  disabled,
  loading,
  onDownload,
  title,
}: {
  description: string;
  disabled: boolean;
  loading: boolean;
  onDownload: () => void;
  title: string;
}) {
  return (
    <section class="bookmark-export__choice">
      <div>
        <strong>{title}</strong>
        <small>{description}</small>
      </div>
      <Button
        icon={FileDown}
        loading={loading}
        disabled={disabled}
        onClick={onDownload}
      >
        {messages.exports.download}
      </Button>
    </section>
  );
}
