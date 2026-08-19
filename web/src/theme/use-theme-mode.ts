import { useState } from 'preact/hooks';

import {
  applyTheme,
  readAppliedTheme,
  storeTheme,
  type ThemeMode,
} from './theme';

export function useThemeMode() {
  const [mode, setModeState] = useState<ThemeMode>(readAppliedTheme);

  const setMode = (nextMode: ThemeMode) => {
    applyTheme(nextMode);
    storeTheme(nextMode);
    setModeState(nextMode);
  };

  return { mode, setMode };
}
