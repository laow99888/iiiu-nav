import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationCategory } from '../navigation/types';
import type { CategoryIconName } from '../ui/icons/category-icon-registry';
import { Button, TextField } from '../ui/primitives';
import { createCategory, updateCategory } from './category-api';
import { CategoryIconPicker } from './category-icon-picker';

type Props = {
  category: NavigationCategory | null;
  onCancel: () => void;
  onSaved: () => void;
};

export function CategoryForm({ category, onCancel, onSaved }: Props) {
  const [name, setName] = useState(category?.name ?? '');
  const [iconName, setIconName] = useState<CategoryIconName | ''>(
    category?.icon ?? '',
  );
  const [privateCategory, setPrivateCategory] = useState(
    category?.visibility === 'private',
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(false);
  const valid = name.trim().length > 0 && Array.from(name.trim()).length <= 80;
  const save = async () => {
    if (!valid || saving) return;
    setSaving(true);
    setError(false);
    const input = {
      name: name.trim(),
      iconName,
      visibility: privateCategory ? ('private' as const) : ('public' as const),
    };
    try {
      if (category) await updateCategory(category.id, input);
      else await createCategory(input);
      onSaved();
    } catch {
      setError(true);
    } finally {
      setSaving(false);
    }
  };
  return (
    <form
      class="category-form"
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
    >
      <TextField
        id="category-name"
        autoFocus
        required
        maxLength={80}
        label={messages.categories.name}
        value={name}
        disabled={saving}
        error={error ? messages.categories.saveFailed : undefined}
        onInput={(event) => setName(event.currentTarget.value)}
      />
      <label class="category-form__visibility">
        <input
          type="checkbox"
          checked={privateCategory}
          disabled={saving}
          onChange={(event) => setPrivateCategory(event.currentTarget.checked)}
        />
        <span>
          <strong>{messages.categories.private}</strong>
          <small>{messages.categories.privateDescription}</small>
        </span>
      </label>
      <CategoryIconPicker
        value={iconName}
        disabled={saving}
        onChange={setIconName}
      />
      <div class="category-form__actions">
        <Button disabled={saving} onClick={onCancel}>
          {messages.categories.cancel}
        </Button>
        <Button
          variant="primary"
          loading={saving}
          disabled={!valid}
          onClick={() => void save()}
        >
          {messages.categories.save}
        </Button>
      </div>
    </form>
  );
}
