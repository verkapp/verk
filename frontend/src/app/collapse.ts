import { signal } from '@angular/core';

/** Remembers which list sections are folded, per view. */
export class CollapsedSections {
  private readonly state = signal<Set<string>>(new Set<string>());

  constructor(private readonly storageKey: string) {
    this.state.set(this.restore());
  }

  isCollapsed(key: string): boolean {
    return this.state().has(key);
  }

  toggle(key: string): void {
    this.state.update((current) => {
      const next = new Set(current);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
    this.persist();
  }

  private restore(): Set<string> {
    try {
      const raw = localStorage.getItem(this.storageKey);
      if (!raw) {
        return new Set<string>();
      }
      const parsed: unknown = JSON.parse(raw);
      if (!Array.isArray(parsed)) {
        return new Set<string>();
      }
      return new Set(parsed.filter((value): value is string => typeof value === 'string'));
    } catch {
      return new Set<string>();
    }
  }

  private persist(): void {
    try {
      localStorage.setItem(this.storageKey, JSON.stringify([...this.state()]));
    } catch {
      // storage can be unavailable in some webview contexts; harmless.
    }
  }
}
