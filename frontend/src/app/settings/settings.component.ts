import { ChangeDetectionStrategy, Component, computed, inject, OnInit, signal } from '@angular/core';
import { marked } from 'marked';

import { formatCreatedAt } from '../dates';
import { ACCENTS, AccentName, Profile } from '../models';
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
  readonly newProfileName = signal('');
  readonly profileColors = ['#7b6cf0', '#4aa985', '#559fe0', '#ea8a62', '#dd7ba2', '#d3a640'];

  readonly releaseNotes = computed(() =>
    marked.parse(this.store.update()?.notes?.trim() ?? '', {
      async: false,
      gfm: true,
      breaks: true,
    }),
  );

  ngOnInit(): void {
    this.pathDraft.set(this.store.settings().claudePath);
    void this.store.refreshClaude();
    void this.store.refreshNotifications();
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

  async toggleNotifications(): Promise<void> {
    const enabled = !this.store.settings().notifications;
    await this.store.saveSettings({ notifications: enabled });
    if (enabled) {
      const status = await this.store.requestNotifications();
      if (status && !status.authorized) {
        this.store.notify('Les notifications sont refusées par le système.', 'error');
      }
    }
  }

  async testNotification(): Promise<void> {
    await this.store.testNotification();
  }

  async createProfile(event: Event): Promise<void> {
    event.preventDefault();
    const name = this.newProfileName().trim();
    if (!name) {
      return;
    }
    const profile = await this.store.createProfile(name);
    if (profile) {
      this.newProfileName.set('');
    }
  }

  async renameProfile(profile: Profile, event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const name = input.value.trim();
    if (!name || name === profile.name) {
      input.value = profile.name;
      return;
    }
    const updated = await this.store.updateProfile(profile.id, name, profile.color);
    if (!updated) {
      input.value = profile.name;
    }
  }

  renameOnEnter(event: Event): void {
    (event.target as HTMLInputElement).blur();
  }

  /** Default time proposed when auto-defer is switched on. */
  readonly defaultAutoDefer = '18:00';

  toggleAutoDefer(profile: Profile): void {
    void this.store.setProfileAutoDefer(profile.id, profile.autoDeferAt ? '' : this.defaultAutoDefer);
  }

  setAutoDeferTime(profile: Profile, event: Event): void {
    const input = event.target as HTMLInputElement;
    const value = input.value;
    if (!value) {
      // Clearing the field would disable the feature by accident: keep the time.
      input.value = profile.autoDeferAt;
      return;
    }
    if (value !== profile.autoDeferAt) {
      void this.store.setProfileAutoDefer(profile.id, value);
    }
  }

  setProfileColor(profile: Profile, color: string): void {
    if (profile.color === color) {
      return;
    }
    void this.store.updateProfile(profile.id, profile.name, color);
  }

  deleteProfile(profile: Profile): void {
    if (this.store.profiles().length <= 1) {
      this.store.notify('Il faut conserver au moins un profil.', 'error');
      return;
    }
    void this.store.deleteProfile(profile.id);
  }

  toggleCloseToTray(): void {
    void this.store.saveSettings({ closeToTray: !this.store.settings().closeToTray });
  }

  toggleClaude(): void {
    void this.store.saveSettings({ claudeEnabled: !this.store.settings().claudeEnabled });
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

  toggleAutoUpdate(): void {
    const enabled = !this.store.settings().autoUpdate;
    void this.store.saveSettings({ autoUpdate: enabled });
    if (enabled) {
      void this.store.checkForUpdates();
    }
  }

  checkUpdates(): void {
    void this.store.checkForUpdates();
  }

  downloadUpdate(): void {
    void this.store.downloadUpdate();
  }

  installUpdate(): void {
    void this.store.installUpdate();
  }

  openUpdatePage(): void {
    this.store.openUpdatePage();
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
