import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { ACCENTS, AccentName } from '../models';
import { Store } from '../store';
import { IconComponent } from '../ui/icon.component';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './settings.component.html',
  styleUrl: './settings.component.css',
})
export class SettingsComponent implements OnInit {
  readonly store = inject(Store);
  readonly accents = ACCENTS;
  readonly pathDraft = signal('');
  readonly checking = signal(false);

  ngOnInit(): void {
    this.pathDraft.set(this.store.settings().claudePath);
    void this.store.refreshClaude();
  }

  setAccent(accent: AccentName): void {
    void this.store.saveSettings({ accent });
  }

  toggleCoach(): void {
    void this.store.saveSettings({ coachOnStart: !this.store.settings().coachOnStart });
  }

  onPathInput(value: string): void {
    this.pathDraft.set(value);
  }

  async savePath(): Promise<void> {
    await this.store.saveSettings({ claudePath: this.pathDraft().trim() });
    await this.verify();
  }

  async verify(): Promise<void> {
    if (this.checking()) {
      return;
    }
    this.checking.set(true);
    const status = await this.store.refreshClaude();
    this.checking.set(false);
    if (status?.available) {
      this.store.notify('Claude Code est bien connecté.', 'accent');
    } else {
      this.store.notify(status?.error || 'Claude Code est introuvable.', 'error');
    }
  }

  openFolder(): void {
    void this.store.openDataFolder();
  }
}
