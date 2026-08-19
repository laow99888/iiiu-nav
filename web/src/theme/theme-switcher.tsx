import { messages } from '../i18n/messages';
import { Monitor, Moon, Sun } from '../ui/icons/interface-icons';
import { Tooltip } from '../ui/primitives';
import { themeModes, type ThemeMode } from './theme';

const icons = {
  light: Sun,
  system: Monitor,
  dark: Moon,
} as const;

type ThemeSwitcherProps = {
  mode: ThemeMode;
  onChange: (mode: ThemeMode) => void;
};

export function ThemeSwitcher({ mode, onChange }: ThemeSwitcherProps) {
  return (
    <div
      class="theme-switcher"
      role="group"
      aria-label={messages.theme.groupLabel}
    >
      {themeModes.map((themeMode) => {
        const Icon = icons[themeMode];
        const label = messages.theme.modes[themeMode];
        return (
          <Tooltip key={themeMode} content={label}>
            <button
              type="button"
              class="theme-switcher__button"
              aria-label={label}
              aria-pressed={mode === themeMode}
              onClick={() => onChange(themeMode)}
            >
              <Icon aria-hidden="true" />
            </button>
          </Tooltip>
        );
      })}
    </div>
  );
}
