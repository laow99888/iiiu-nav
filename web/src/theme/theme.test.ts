import { afterEach, describe, expect, it } from 'vitest';

import {
  applyTheme,
  initializeTheme,
  isThemeMode,
  readStoredTheme,
  storeTheme,
} from './theme';

describe('主题状态', () => {
  afterEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    document.documentElement.style.colorScheme = '';
  });

  it('仅接受支持的主题值', () => {
    expect(isThemeMode('light')).toBe(true);
    expect(isThemeMode('system')).toBe(true);
    expect(isThemeMode('dark')).toBe(true);
    expect(isThemeMode('sepia')).toBe(false);
  });

  it('无效或不可读取的存储值回退为系统主题', () => {
    localStorage.setItem('iiiu-nav.theme', 'unknown');
    expect(readStoredTheme()).toBe('system');
    expect(
      readStoredTheme({
        getItem: () => {
          throw new Error('blocked');
        },
      }),
    ).toBe('system');
  });

  it('初始化已保存主题并允许预览值覆盖', () => {
    localStorage.setItem('iiiu-nav.theme', 'dark');
    expect(initializeTheme()).toBe('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');

    expect(initializeTheme('light')).toBe('light');
    expect(document.documentElement.dataset.theme).toBe('light');
  });

  it('系统模式移除强制主题且存储失败不影响应用', () => {
    applyTheme('dark');
    applyTheme('system');
    expect(document.documentElement).not.toHaveAttribute('data-theme');
    expect(document.documentElement.style.colorScheme).toBe('');

    expect(() =>
      storeTheme('light', {
        setItem: () => {
          throw new Error('blocked');
        },
      }),
    ).not.toThrow();
  });
});
