import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/preact';
import { afterEach, vi } from 'vitest';

afterEach(() => {
  cleanup();
  document.documentElement.removeAttribute('data-theme');
});

vi.mock('@floating-ui/dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@floating-ui/dom')>();
  return {
    ...actual,
    autoUpdate: (
      _reference: Element,
      _floating: HTMLElement,
      update: () => void,
    ) => {
      update();
      return () => undefined;
    },
    computePosition: () => Promise.resolve({ x: 0, y: 0 }),
  };
});

vi.mock('@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter', () => ({
  draggable: vi.fn(() => () => undefined),
  dropTargetForElements: vi.fn(() => () => undefined),
  monitorForElements: vi.fn(() => () => undefined),
}));
