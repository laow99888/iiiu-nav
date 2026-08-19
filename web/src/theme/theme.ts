export const themeModes = ['light', 'system', 'dark'] as const;
export type ThemeMode = (typeof themeModes)[number];

const storageKey = 'iiiu-nav.theme';

export function isThemeMode(value: unknown): value is ThemeMode {
  return themeModes.includes(value as ThemeMode);
}

export function readStoredTheme(
  storage: Pick<Storage, 'getItem'> = localStorage,
): ThemeMode {
  try {
    const stored = storage.getItem(storageKey);
    return isThemeMode(stored) ? stored : 'system';
  } catch {
    return 'system';
  }
}

export function applyTheme(
  mode: ThemeMode,
  root: Pick<HTMLElement, 'dataset' | 'style'> = document.documentElement,
) {
  if (mode === 'system') {
    delete root.dataset.theme;
    root.style.colorScheme = '';
    return;
  }

  root.dataset.theme = mode;
  root.style.colorScheme = mode;
}

export function storeTheme(
  mode: ThemeMode,
  storage: Pick<Storage, 'setItem'> = localStorage,
) {
  try {
    storage.setItem(storageKey, mode);
  } catch {
    // A blocked storage API must not prevent an in-memory theme change.
  }
}

export function initializeTheme(previewMode?: ThemeMode | null) {
  const mode = previewMode ?? readStoredTheme();
  applyTheme(mode);
  return mode;
}

export function readAppliedTheme(): ThemeMode {
  const applied = document.documentElement.dataset.theme;
  return isThemeMode(applied) ? applied : 'system';
}
