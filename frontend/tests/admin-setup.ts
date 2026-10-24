import { afterEach, vi } from 'vitest';
import { cleanup } from '@testing-library/react';

vi.stubGlobal(
  'ResizeObserver',
  class {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
);
window.matchMedia = vi.fn().mockImplementation((query) => ({
  matches: false,
  media: query,
  onchange: null,
  addListener: vi.fn(),
  removeListener: vi.fn(),
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  dispatchEvent: vi.fn(),
}));
Element.prototype.scrollIntoView = vi.fn();
Element.prototype.getAnimations = vi.fn(() => []);
afterEach(() => {
  cleanup();
  localStorage.clear();
});
