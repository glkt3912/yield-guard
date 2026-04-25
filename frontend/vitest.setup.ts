/// <reference types="vitest/globals" />
import "@testing-library/jest-dom";

// Recharts uses ResizeObserver which is not available in jsdom
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.ResizeObserver = ResizeObserverMock;

// Restore any vi.stubGlobal overrides between tests to prevent cross-test pollution
afterEach(() => {
  vi.unstubAllGlobals();
});
