import { render } from 'preact';

import { App } from './app';
import { initializeTheme, isThemeMode } from './theme/theme';
import './styles.css';

const previewTheme = import.meta.env.DEV
  ? new URLSearchParams(window.location.search).get('theme')
  : null;
initializeTheme(isThemeMode(previewTheme) ? previewTheme : null);

const root = document.getElementById('app');

if (!root) {
  throw new Error('Missing application root');
}

render(<App />, root);
