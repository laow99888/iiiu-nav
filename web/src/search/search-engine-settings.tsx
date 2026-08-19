import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { Button, Drawer } from '../ui/primitives';
import { SortableList } from '../ui/sortable';
import { saveSearchEngines } from '../navigation/navigation-api';
import type { SearchEngine } from './search-engines';

type SearchEngineSettingsProps = {
  engines: readonly SearchEngine[];
  onClose: () => void;
  onSaved: () => void;
  open: boolean;
};

export function SearchEngineSettings({
  engines,
  onClose,
  onSaved,
  open,
}: SearchEngineSettingsProps) {
  const [draft, setDraft] = useState<SearchEngine[]>([...engines]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (open) {
      setDraft([...engines]);
      setError(false);
    }
  }, [engines, open]);

  const hasEnabledEngine = draft.some((engine) => engine.enabled);
  const save = async () => {
    if (!hasEnabledEngine) return;
    setSaving(true);
    setError(false);
    try {
      await saveSearchEngines(
        draft.map(({ id, enabled }) => ({ id, enabled })),
      );
      onSaved();
      onClose();
    } catch {
      setError(true);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Drawer
      open={open}
      title={messages.search.settingsTitle}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose}>{messages.search.cancel}</Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={!hasEnabledEngine}
            onClick={() => void save()}
          >
            {messages.search.save}
          </Button>
        </>
      }
    >
      <div class="search-engine-settings">
        <p>{messages.search.settingsDescription}</p>
        <SortableList
          items={draft}
          disabled={saving}
          getId={(engine) => engine.id}
          getLabel={(engine) => engine.name}
          onReorder={setDraft}
          renderItem={(engine) => (
            <label class="search-engine-settings__row">
              <input
                type="checkbox"
                checked={engine.enabled}
                disabled={saving}
                onChange={(event) =>
                  setDraft((current) =>
                    current.map((item) =>
                      item.id === engine.id
                        ? { ...item, enabled: event.currentTarget.checked }
                        : item,
                    ),
                  )
                }
              />
              <span class="search-engine-settings__mark" aria-hidden="true">
                {engine.mark}
              </span>
              <span>{engine.name}</span>
            </label>
          )}
        />
        {!hasEnabledEngine ? (
          <p class="ui-field__error" role="alert">
            {messages.search.oneRequired}
          </p>
        ) : null}
        {error ? (
          <p class="ui-field__error" role="alert">
            {messages.search.saveFailed}
          </p>
        ) : null}
      </div>
    </Drawer>
  );
}
