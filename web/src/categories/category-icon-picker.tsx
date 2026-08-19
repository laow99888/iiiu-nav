import { useMemo, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { CategoryIcon } from '../ui/icons/category-icons';
import {
  findCategoryIcons,
  type CategoryIconName,
} from '../ui/icons/category-icon-registry';
import { Search, X } from '../ui/icons/interface-icons';

type Props = {
  disabled?: boolean;
  onChange: (value: CategoryIconName | '') => void;
  value: CategoryIconName | '';
};

export function CategoryIconPicker({ disabled, onChange, value }: Props) {
  const [query, setQuery] = useState('');
  const icons = useMemo(() => findCategoryIcons(query), [query]);
  return (
    <fieldset class="category-icon-picker" disabled={disabled}>
      <legend>{messages.categories.icon}</legend>
      <label class="category-icon-picker__search">
        <Search aria-hidden="true" />
        <input
          type="search"
          value={query}
          aria-label={messages.categories.searchIcons}
          placeholder={messages.categories.searchIcons}
          onInput={(event) => setQuery(event.currentTarget.value)}
        />
      </label>
      <div class="category-icon-picker__grid">
        <button
          type="button"
          class="category-icon-picker__option"
          aria-pressed={value === ''}
          onClick={() => onChange('')}
        >
          <X aria-hidden="true" />
          <span>{messages.categories.noIcon}</span>
        </button>
        {icons.map(([name, item]) => (
          <button
            key={name}
            type="button"
            class="category-icon-picker__option"
            aria-pressed={value === name}
            onClick={() => onChange(name)}
          >
            <CategoryIcon name={name} />
            <span>{item.label}</span>
          </button>
        ))}
      </div>
      {icons.length === 0 ? (
        <p class="category-icon-picker__empty">
          {messages.categories.noIconResults}
        </p>
      ) : null}
    </fieldset>
  );
}
