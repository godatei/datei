// Node 26 ships experimental Web Storage, defining global `localStorage` and
// `sessionStorage` accessors that return `undefined` (and warn) unless
// `--localstorage-file` is set. Those accessors shadow jsdom's storage during
// environment setup, leaving the globals unusable. Install a simple in-memory
// Storage implementation so tests relying on web storage work.
class MemoryStorage implements Storage {
  private store = new Map<string, string>();

  get length(): number {
    return this.store.size;
  }

  clear(): void {
    this.store.clear();
  }

  getItem(key: string): string | null {
    return this.store.get(key) ?? null;
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null;
  }

  removeItem(key: string): void {
    this.store.delete(key);
  }

  setItem(key: string, value: string): void {
    this.store.set(key, String(value));
  }
}

Object.defineProperty(globalThis, 'localStorage', {
  value: new MemoryStorage(),
  configurable: true,
});
Object.defineProperty(globalThis, 'sessionStorage', {
  value: new MemoryStorage(),
  configurable: true,
});
