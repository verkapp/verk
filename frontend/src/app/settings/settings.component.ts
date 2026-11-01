import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { formatCreatedAt } from '../dates';
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
  readonly syncRemote = signal('');
  readonly syncBusy = signal(false);

  ngOnInit(): void {
    this.pathDraft.set(this.store.settings().claudePath);
    void this.store.refreshClaude();
    void this.loadSync();
  }

  private async loadSync(): Promise<void> {
    const status = await this.store.refreshSync();
    this.syncRemote.set(status?.remote ?? '');
  }

  setAccent(accent: AccentName): void {
    void this.store.saveSettings({ accent });
  }

  toggleTimeTracking(): void {
    void this.store.saveSettings({ timeTracking: !this.store.settings().timeTracking });
  }

  toggleMomentum(): void {
    void this.store.saveSettings({ showMomentum: !this.store.settings().showMomentum });
  }

  toggleCoachEnabled(): void {
    void this.store.saveSettings({ coachEnabled: !this.store.settings().coachEnabled });
  }

  toggleCoach(): void {
    if (!this.store.settings().coachEnabled) {
      return;
    }
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

  onSyncRemoteInput(value: string): void {
    this.syncRemote.set(value);
  }

  async saveSyncRemote(): Promise<void> {
    if (this.syncBusy()) {
      return;
    }
    this.syncBusy.set(true);
    const remote = this.syncRemote().trim();
    const status = await this.store.configureSync(remote, remote !== '' && (this.store.sync()?.enabled ?? true));
    this.syncRemote.set(status?.remote ?? remote);
    this.syncBusy.set(false);
  }

  async toggleSync(): Promise<void> {
    const status = this.store.sync();
    if (!status?.configured) {
      this.store.notify('Ajoute d’abord l’adresse d’un dépôt git.', 'error');
      return;
    }
    if (this.syncBusy()) {
      return;
    }
    this.syncBusy.set(true);
    await this.store.configureSync(status.remote, !status.enabled);
    this.syncBusy.set(false);
  }

  async syncNow(): Promise<void> {
    if (this.syncBusy() || this.store.sync()?.syncing) {
      return;
    }
    this.syncBusy.set(true);
    await this.store.syncNow();
    this.syncBusy.set(false);
  }

  openSyncFolder(): void {
    void this.store.openSyncFolder();
  }

  syncHeadline(): string {
    const status = this.store.sync();
    if (!status) {
      return 'Vérification de la synchronisation…';
    }
    if (status.lastError) {
      return 'La synchronisation a échoué';
    }
    if (status.syncing) {
      return 'Synchronisation en cours…';
    }
    if (status.lastSyncAt > 0) {
      return 'Dernière synchronisation';
    }
    return 'Prêt à synchroniser';
  }

  syncDetail(): string {
    const status = this.store.sync();
    if (!status) {
      return '';
    }
    if (status.lastError) {
      return status.lastError;
    }
    if (status.syncing) {
      return 'Envoi et récupération des changements.';
    }
    if (!status.gitAvailable) {
      return 'git est introuvable sur cette machine.';
    }
    if (status.lastSyncAt > 0) {
      return `${formatCreatedAt(status.lastSyncAt)} · branche ${status.branch}`;
    }
    return `Branche ${status.branch}`;
  }
}
