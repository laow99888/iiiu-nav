import { useEffect, useId, useMemo, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { LinkLogo } from '../navigation/link-logo';
import type { NavigationCategory } from '../navigation/types';
import { Search, Settings } from '../ui/icons/interface-icons';
import { Button, Menu, Tooltip } from '../ui/primitives';
import { searchLocalLinks } from './local-search';
import {
  defaultSearchEngines,
  enabledSearchEngines,
  buildSearchURL,
  type SearchEngine,
} from './search-engines';

type CombinedSearchProps = {
  categories: readonly NavigationCategory[];
  engines?: readonly SearchEngine[];
  onConfigure?: () => void;
  onOpen?: (url: string) => void;
};

function openExternal(url: string) {
  const opened = window.open(url, '_blank', 'noopener,noreferrer');
  if (opened) {
    opened.opener = null;
  }
}

export function CombinedSearch({
  categories,
  engines = defaultSearchEngines,
  onConfigure,
  onOpen = openExternal,
}: CombinedSearchProps) {
  const availableEngines = useMemo(
    () => enabledSearchEngines(engines),
    [engines],
  );
  const [engineID, setEngineID] = useState(availableEngines[0]?.id ?? null);
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(-1);
  const [focused, setFocused] = useState(false);
  const listboxID = useId();
  const results = useMemo(
    () => searchLocalLinks(categories, query),
    [categories, query],
  );
  const selectedEngine =
    availableEngines.find((engine) => engine.id === engineID) ??
    availableEngines[0];
  const showResults = focused && query.trim().length > 0;

  useEffect(() => {
    if (!selectedEngine && availableEngines[0]) {
      setEngineID(availableEngines[0].id);
    }
  }, [availableEngines, selectedEngine]);

  const openLocalResult = (index: number) => {
    const result = results[index];
    if (!result) return;
    onOpen(result.link.url);
    setFocused(false);
  };

  const submitWebSearch = () => {
    if (!selectedEngine || !query.trim()) return;
    onOpen(buildSearchURL(selectedEngine, query));
    setFocused(false);
  };

  return (
    <div class={`combined-search${showResults ? ' is-open' : ''}`}>
      <form
        class="combined-search__bar"
        role="search"
        onSubmit={(event) => {
          event.preventDefault();
          if (activeIndex >= 0) {
            openLocalResult(activeIndex);
          } else {
            submitWebSearch();
          }
        }}
      >
        <Search class="combined-search__icon" aria-hidden="true" />
        <input
          type="search"
          value={query}
          role="combobox"
          aria-label={messages.search.label}
          aria-autocomplete="list"
          aria-expanded={showResults}
          aria-controls={showResults ? listboxID : undefined}
          aria-activedescendant={
            activeIndex >= 0 ? `${listboxID}-${activeIndex}` : undefined
          }
          placeholder={messages.search.placeholder}
          onFocus={() => setFocused(true)}
          onBlur={() => window.setTimeout(() => setFocused(false), 120)}
          onInput={(event) => {
            setQuery(event.currentTarget.value);
            setActiveIndex(-1);
          }}
          onKeyDown={(event) => {
            if (event.key === 'ArrowDown' && results.length > 0) {
              event.preventDefault();
              setActiveIndex((index) => (index + 1) % results.length);
            } else if (event.key === 'ArrowUp' && results.length > 0) {
              event.preventDefault();
              setActiveIndex((index) =>
                index <= 0 ? results.length - 1 : index - 1,
              );
            } else if (event.key === 'Escape') {
              event.preventDefault();
              setFocused(false);
              setActiveIndex(-1);
            }
          }}
        />
        <div class="combined-search__actions">
          {selectedEngine ? (
            <Menu
              label={selectedEngine.name}
              items={availableEngines.map((engine) => ({
                id: engine.id,
                label: engine.name,
                onSelect: () => setEngineID(engine.id),
              }))}
            />
          ) : null}
          {onConfigure ? (
            <Tooltip content={messages.search.configure}>
              <Button
                variant="ghost"
                icon={Settings}
                aria-label={messages.search.configure}
                onClick={onConfigure}
              />
            </Tooltip>
          ) : null}
        </div>
      </form>

      {showResults ? (
        <div class="combined-search__results" id={listboxID} role="listbox">
          {results.length > 0 ? (
            results.map((result, index) => (
              <button
                key={result.link.id}
                id={`${listboxID}-${index}`}
                type="button"
                class="combined-search__result"
                role="option"
                aria-selected={activeIndex === index}
                onMouseDown={(event) => event.preventDefault()}
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => openLocalResult(index)}
              >
                <LinkLogo
                  label={result.link.name}
                  text={result.link.logoText}
                  tone={result.link.logoTone}
                  url={result.link.logoURL}
                />
                <span>
                  <strong>{result.link.name}</strong>
                  <small>{result.categoryName}</small>
                </span>
              </button>
            ))
          ) : (
            <div class="combined-search__web-option">
              <Search aria-hidden="true" />
              <span>
                {selectedEngine
                  ? messages.search.webSearch(selectedEngine.name, query.trim())
                  : messages.search.noEngine}
              </span>
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}
