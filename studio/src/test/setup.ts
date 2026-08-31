import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import { resetApiMock } from "./api-mock";

const storageValues = new Map<string, string>();
const testStorage: Storage = {
  get length() { return storageValues.size; },
  clear: () => storageValues.clear(),
  getItem: (key) => storageValues.get(key) ?? null,
  key: (index) => [...storageValues.keys()][index] ?? null,
  removeItem: (key) => { storageValues.delete(key); },
  setItem: (key, value) => { storageValues.set(key, String(value)); },
};
Object.defineProperty(globalThis, "localStorage", { configurable: true, value: testStorage });

class TestEventSource {
  static readonly CLOSED = 2;
  readonly readyState = 1;
  onopen: ((event: Event) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  close() {}
  addEventListener() {}
}

if (!(HTMLDialogElement.prototype as HTMLDialogElement & { showModal?: () => void }).showModal) {
  HTMLDialogElement.prototype.showModal = function showModal() {
    this.open = true;
    this.querySelector<HTMLElement>("button, [href], input, select, textarea, [tabindex]:not([tabindex='-1'])")?.focus();
  };
}

if (!(HTMLDialogElement.prototype as HTMLDialogElement & { close?: () => void }).close) {
  HTMLDialogElement.prototype.close = function close() {
    this.open = false;
    this.dispatchEvent(new Event("close"));
  };
}

beforeEach(() => {
  testStorage.clear();
  vi.stubGlobal("EventSource", TestEventSource);
  resetApiMock();
  vi.spyOn(console, "error").mockImplementation((...values: unknown[]) => {
    throw new Error(`Unexpected console.error: ${values.map(String).join(" ")}`);
  });
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});