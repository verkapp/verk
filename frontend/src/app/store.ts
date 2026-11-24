import { computed, Injectable, signal } from '@angular/core';

import { api, errorMessage, onRuntimeEvent } from './api';
import {
  compareOpenTasks,
  endOfToday,
  groupHistory,
  isDueToday,
  isOverdue,
  isSameDay,
  resumeDay,
  resumeDueAt,
  startOfDay,
} from './dates';
import {
  AccentName,
  AppInfo,
  AttachmentInput,
  ClaudeStatus,
  HistoryGroup,
  NotificationStatus,
  Profile,
  Settings,
  Stats,
  SyncStatus,
  Task,
  TaskInput,
  ToastMessage,
  ToastTone,
  UpdateInfo,
} from './models';

const DAY = 86_400_000;

@Injectable({ providedIn: 'root' })
export class Store {
  readonly tasks = signal<Task[]>([]);
  readonly profiles = signal<Profile[]>([]);
  readonly activeProfileId = signal<string>('');
  readonly settings = signal<Settings>({ accent: 'lavande', coachEnabled: true, coachOnStart: false, timeTracking: true, showMomentum: true, notifications: true, autoUpdate: true, claudeEnabled: true, claudePath: '', closeToTray: false });
  readonly claude = signal<ClaudeStatus | null>(null);
  /** The coach needs both its own switch and the global Claude integration. */
  readonly coachActive = computed(() => this.settings().claudeEnabled && this.settings().coachEnabled);
  readonly notifications = signal<NotificationStatus | null>(null);
  readonly sync = signal<SyncStatus | null>(null);
  readonly update = signal<UpdateInfo | null>(null);
  readonly checkingUpdate = signal(false);
  readonly installingUpdate = signal(false);
  readonly info = signal<AppInfo | null>(null);
  readonly loading = signal(true);
  readonly fatalError = signal<string | null>(null);
  readonly toast = signal<ToastMessage | null>(null);
  readonly nudge = signal<{ text: string; at: number } | null>(null);
  readonly nudgeLoading = signal(false);
  readonly nudgeError = signal<string | null>(null);

  readonly activeProfile = computed(() =>
    this.profiles().find((profile) => profile.id === this.activeProfileId()) ?? this.profiles()[0] ?? null,
  );

  /** Tasks of the active profile. Profiles are containers: every view is scoped. */
  readonly scopedTasks = computed(() => {
    const profileId = this.activeProfile()?.id;
    return profileId ? this.tasks().filter((task) => task.profileId === profileId) : this.tasks();
  });

  readonly profileCounts = computed<Record<string, number>>(() => {
    const counts: Record<string, number> = {};
    for (const task of this.tasks()) {
      if (!task.done) {
        counts[task.profileId] = (counts[task.profileId] ?? 0) + 1;
      }
    }
    return counts;
  });

  readonly openTasks = computed(() => this.scopedTasks().filter((task) => !task.done).sort(compareOpenTasks));
  readonly startedTasks = computed(() =>
    this.openTasks()
      .filter((task) => task.startedAt !== null)
      .sort((a, b) => (a.startedAt ?? 0) - (b.startedAt ?? 0)),
  );
  /** Wall clock refreshed every 15 s so running timers update on screen. */
  readonly now = signal(Date.now());
  readonly doneTodayTasks = computed(() =>
    this.scopedTasks()
      .filter((task) => task.done && task.completedAt !== null && isSameDay(task.completedAt, this.now()))
      .sort((a, b) => (b.completedAt ?? 0) - (a.completedAt ?? 0)),
  );
  readonly doneTasks = computed(() =>
    this.scopedTasks()
      .filter((task) => task.done)
      .sort((a, b) => (b.completedAt ?? 0) - (a.completedAt ?? 0)),
  );
  /** Closed tasks that were actually done (not dropped as "won't do"). */
  readonly completedTasks = computed(() => this.doneTasks().filter((task) => !task.wontDo));
  readonly history = computed<HistoryGroup[]>(() => groupHistory(this.scopedTasks()));

  readonly stats = computed<Stats>(() => {
    const now = Date.now();
    const open = this.scopedTasks().filter((task) => !task.done);
    const doneToday = this.scopedTasks().filter(
      (task) => task.done && !task.wontDo && task.completedAt !== null && isSameDay(task.completedAt, now),
    ).length;
    const dueToday = open.filter((task) => isDueToday(task) && !isOverdue(task, now)).length;
    const overdue = open.filter((task) => isOverdue(task, now)).length;
    const stale = open.filter(
      (task) =>
        now - task.createdAt > 3 * DAY &&
        !isOverdue(task, now) &&
        (task.dueAt === null || startOfDay(task.dueAt).getTime() >= startOfDay(now).getTime()),
    ).length;
    const denominator = doneToday + dueToday + overdue;
    return {
      open: open.length,
      dueToday,
      overdue,
      doneToday,
      stale,
      completion: denominator === 0 ? 0 : Math.round((doneToday / denominator) * 100),
    };
  });

  readonly attentionTasks = computed(() => {
    const now = Date.now();
    return this.openTasks().filter(
      (task) => isOverdue(task, now) || now - task.createdAt > 3 * DAY,
    );
  });

  readonly coachNudge = computed(() => this.nudge());

  private toastTimer: ReturnType<typeof setTimeout> | null = null;
  /** True while a manual refresh (reload + git sync) is running. */
  readonly refreshing = signal(false);
  private syncGeneration = -1;
  private syncPolling = false;
  private focusRefreshing = false;
  private lastFocusRefresh = 0;
  private lastFocusSync = 0;

  async init(): Promise<void> {
    this.listenForUpdates();
    this.listenForFocus();
    setInterval(() => {
      this.now.set(Date.now());
      void this.pollSync();
    }, 15_000);
    try {
      const [tasks, settings, info, profiles] = await Promise.all([
        api.listTasks(),
        api.getSettings(),
        api.appInfo(),
        api.listProfiles(),
      ]);
      this.tasks.set(tasks);
      this.profiles.set(profiles);
      this.restoreProfile(profiles);
      this.info.set(info);
      this.applySettings(settings);
      void this.refreshClaude();
      void this.refreshNotifications();
      void this.pollSync();
      void this.refreshUpdateStatus();
      if (settings.claudeEnabled && settings.coachEnabled && settings.coachOnStart) {
        setTimeout(() => void this.requestNudge(true), 1200);
      }
    } catch (error) {
      this.fatalError.set(errorMessage(error));
    } finally {
      this.loading.set(false);
    }
  }

  private applySettings(settings: Settings): void {
    this.settings.set(settings);
    this.applyAccent(settings.accent);
    try {
      localStorage.setItem('verk.settings', JSON.stringify(settings));
    } catch {
      // localStorage can be unavailable in some webview contexts; harmless.
    }
  }

  private applyAccent(accent: AccentName): void {
    document.documentElement.dataset['accent'] = accent;
  }

  private restoreProfile(profiles: Profile[]): void {
    let stored = '';
    try {
      stored = localStorage.getItem('verk.profile') ?? '';
    } catch {
      // storage can be unavailable in some webview contexts
    }
    const id = profiles.some((profile) => profile.id === stored) ? stored : (profiles[0]?.id ?? '');
    this.activeProfileId.set(id);
  }

  private ensureActiveProfile(profiles: Profile[]): void {
    if (profiles.some((profile) => profile.id === this.activeProfileId())) {
      return;
    }
    this.activeProfileId.set(profiles[0]?.id ?? '');
  }

  selectProfile(id: string): void {
    if (!this.profiles().some((profile) => profile.id === id)) {
      return;
    }
    this.activeProfileId.set(id);
    try {
      localStorage.setItem('verk.profile', id);
    } catch {
      // ignore storage restrictions
    }
  }

  async createProfile(name: string): Promise<Profile | null> {
    try {
      const profile = await api.createProfile(name);
      const profiles = await api.listProfiles();
      this.profiles.set(profiles);
      this.selectProfile(profile.id);
      this.notify(`Profil « ${profile.name} » créé.`, 'accent');
      return profile;
    } catch (error) {
      this.fail(error);
      return null;
    }
  }

  async updateProfile(id: string, name: string, color: string): Promise<Profile | null> {
    try {
      const profile = await api.updateProfile(id, name, color);
      this.profiles.update((profiles) => profiles.map((item) => (item.id === id ? profile : item)));
      return profile;
    } catch (error) {
      this.fail(error);
      return null;
    }
  }

  /** Sets (or clears with "") the time after which a profile's unfinished tasks move on. */
  async setProfileAutoDefer(id: string, at: string): Promise<Profile | null> {
    try {
      const profile = await api.setProfileAutoDefer(id, at);
      this.profiles.update((profiles) => profiles.map((item) => (item.id === id ? profile : item)));
      return profile;
    } catch (error) {
      this.fail(error);
      return null;
    }
  }

  async deleteProfile(id: string): Promise<void> {
    try {
      await api.deleteProfile(id);
      await this.reload();
      this.notify('Profil supprimé. Ses tâches ont été déplacées.', 'default');
    } catch (error) {
      this.fail(error);
    }
  }

  async refreshNotifications(): Promise<NotificationStatus | null> {
    try {
      const status = await api.notificationStatus();
      this.notifications.set(status);
      return status;
    } catch {
      this.notifications.set(null);
      return null;
    }
  }

  async requestNotifications(): Promise<NotificationStatus | null> {
    try {
      const status = await api.requestNotifications();
      this.notifications.set(status);
      return status;
    } catch (error) {
      this.fail(error);
      return null;
    }
  }

  async testNotification(): Promise<void> {
    try {
      await api.testNotification();
      this.notify('Notification de test envoyée.', 'accent');
    } catch (error) {
      this.fail(error);
    }
  }

  private listenForUpdates(): void {
    onRuntimeEvent<UpdateInfo>('update:status', (info) => this.applyUpdate(info));
    // Sent after every git sync (including the one at startup), so pulled
    // changes show up without waiting for the 15 s poll.
    onRuntimeEvent<SyncStatus>('sync:status', () => void this.refreshSync());
    // The backend moved unfinished tasks past a profile's auto-defer time.
    onRuntimeEvent<{ count: number; label: string }>('tasks:deferred', (event) => {
      void this.reload().catch(() => undefined);
      const plural = event.count > 1;
      this.notify(
        `${event.count} tâche${plural ? 's non terminées reportées' : ' non terminée reportée'} à ${event.label}.`,
        'default',
      );
    });
  }

  /** Refreshes from the local database when the window regains focus. */
  private listenForFocus(): void {
    window.addEventListener('focus', this.onWindowFocus);
    document.addEventListener('visibilitychange', this.onVisibilityChange);
  }

  private readonly onWindowFocus = (): void => {
    if (document.visibilityState === 'hidden') {
      return;
    }
    void this.refreshOnFocus();
  };

  private readonly onVisibilityChange = (): void => {
    if (document.visibilityState === 'visible') {
      void this.refreshOnFocus();
    }
  };

  private async refreshOnFocus(): Promise<void> {
    if (this.loading() || this.fatalError() || this.focusRefreshing) {
      return;
    }
    // Focus and visibilitychange often fire together; one refresh is enough.
    const now = Date.now();
    if (now - this.lastFocusRefresh < 1500) {
      return;
    }
    this.lastFocusRefresh = now;
    this.focusRefreshing = true;
    this.now.set(now);
    try {
      await this.reload();
      const status = await this.refreshSync();
      if (status?.configured && status.enabled && !status.syncing && now - this.lastFocusSync >= 5000) {
        this.lastFocusSync = now;
        try {
          await api.syncNow();
        } catch {
          // The error is kept in SyncStatus rather than shown as a toast here.
        }
        await this.refreshSync();
        await this.reload();
      }
    } catch {
      // A background refresh must never disrupt the current view.
    } finally {
      this.focusRefreshing = false;
    }
  }

  /** Manual refresh: reloads tasks and runs a git sync when it is configured. */
  async refreshNow(): Promise<void> {
    if (this.refreshing()) {
      return;
    }
    this.refreshing.set(true);
    this.now.set(Date.now());
    try {
      await this.reload();
      const status = await this.refreshSync();
      if (status?.configured && status.enabled && !status.syncing) {
        try {
          await api.syncNow();
        } catch (error) {
          this.notify(errorMessage(error), 'error');
        }
        const after = await this.refreshSync();
        if (after?.lastError) {
          this.notify(after.lastError, 'error');
        }
        await this.reload();
      }
    } catch (error) {
      this.fail(error);
    } finally {
      this.refreshing.set(false);
    }
  }

  private applyUpdate(info: UpdateInfo): void {
    const previous = this.update();
    this.update.set(info);
    if (info.ready && !previous?.ready) {
      this.notify(`Verk ${info.latestVersion} est prêt. Il s’installera à la fermeture.`, 'accent');
    }
  }

  async refreshUpdateStatus(): Promise<UpdateInfo | null> {
    try {
      const info = await api.updateStatus();
      this.applyUpdate(info);
      return info;
    } catch {
      return null;
    }
  }

  async checkForUpdates(): Promise<UpdateInfo | null> {
    if (this.checkingUpdate()) {
      return this.update();
    }
    this.checkingUpdate.set(true);
    try {
      const info = await api.checkForUpdates();
      this.applyUpdate(info);
      if (info.error) {
        this.notify(info.error, 'error');
      } else if (info.available) {
        this.notify(`Version ${info.latestVersion} disponible.`, 'accent');
        if (this.settings().autoUpdate && !info.ready && !info.downloading) {
          void this.downloadUpdate();
        }
      } else {
        this.notify('Verk est à jour.', 'accent');
      }
      return info;
    } catch (error) {
      this.fail(error);
      return null;
    } finally {
      this.checkingUpdate.set(false);
    }
  }

  async downloadUpdate(): Promise<UpdateInfo | null> {
    try {
      const info = await api.downloadUpdate();
      this.applyUpdate(info);
      if (info.error) {
        this.notify(info.error, 'error');
      }
      return info;
    } catch (error) {
      this.fail(error);
      return null;
    }
  }

  async installUpdate(): Promise<void> {
    if (this.installingUpdate()) {
      return;
    }
    this.installingUpdate.set(true);
    try {
      await api.installUpdate();
    } catch (error) {
      this.installingUpdate.set(false);
      this.fail(error);
    }
  }

  openUpdatePage(): void {
    void api.openUpdatePage();
  }


  async saveSettings(patch: Partial<Settings>): Promise<void> {
    const next: Settings = { ...this.settings(), ...patch };
    if (patch.coachEnabled === false || patch.claudeEnabled === false) {
      this.nudge.set(null);
      this.nudgeError.set(null);
    }
    this.applySettings(next);
    try {
      await api.saveSettings(next);
      if (patch.claudeEnabled !== undefined || patch.claudePath !== undefined) {
        void this.refreshClaude();
      }
    } catch (error) {
      this.notify(errorMessage(error), 'error');
    }
  }

  /** Reads the git sync status and refreshes the task list after a pull. */
  async refreshSync(): Promise<SyncStatus | null> {
    try {
      const status = await api.syncStatus();
      const previousGeneration = this.syncGeneration;
      this.sync.set(status);
      this.syncGeneration = status.generation;
      if (status.generation !== previousGeneration) {
        await this.reload();
      }
      return status;
    } catch {
      this.sync.set(null);
      return null;
    }
  }

  private async pollSync(): Promise<void> {
    if (this.syncPolling) {
      return;
    }
    this.syncPolling = true;
    try {
      await this.refreshSync();
    } finally {
      this.syncPolling = false;
    }
  }

  async configureSync(remote: string, enabled: boolean): Promise<SyncStatus | null> {
    try {
      await api.setSyncConfig(remote, enabled);
      const status = await this.refreshSync();
      this.notify(remote ? 'Synchronisation git configurée.' : 'Synchronisation désactivée.', remote ? 'accent' : 'default');
      return status;
    } catch (error) {
      this.notify(errorMessage(error), 'error');
      await this.refreshSync();
      return null;
    }
  }

  async syncNow(): Promise<boolean> {
    try {
      await api.syncNow();
      const status = await this.refreshSync();
      if (status?.lastError) {
        this.notify(status.lastError, 'error');
        return false;
      }
      this.notify('Synchronisation terminée.', 'accent');
      return true;
    } catch (error) {
      this.notify(errorMessage(error), 'error');
      await this.refreshSync();
      return false;
    }
  }

  async refreshClaude(): Promise<ClaudeStatus | null> {
    if (!this.settings().claudeEnabled) {
      this.claude.set(null);
      return null;
    }
    try {
      const status = await api.claudeStatus();
      this.claude.set(status);
      return status;
    } catch (error) {
      this.claude.set({ available: false, path: '', version: '', error: errorMessage(error) });
      return null;
    }
  }

  private async reload(): Promise<void> {
    const [tasks, profiles] = await Promise.all([api.listTasks(), api.listProfiles()]);
    this.tasks.set(tasks);
    this.profiles.set(profiles);
    this.ensureActiveProfile(profiles);
  }

  private fail(error: unknown): void {
    this.notify(errorMessage(error), 'error');
  }

  private async mutate<T>(work: Promise<T>, refresh = true): Promise<T> {
    const result = await work;
    if (refresh) {
      await this.reload();
    }
    return result;
  }

  async createTask(input: TaskInput): Promise<Task> {
    try {
      const profileId = input.profileId || this.activeProfile()?.id || '';
      return await this.mutate(api.createTask({ ...input, profileId }));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async updateTask(id: string, input: TaskInput): Promise<Task> {
    try {
      return await this.mutate(api.updateTask(id, input));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async setTaskDone(id: string, done: boolean): Promise<Task> {
    try {
      const task = await this.mutate(api.setTaskDone(id, done));
      this.notify(done ? 'Tâche terminée. Bien joué.' : 'Tâche rouverte.', done ? 'accent' : 'default');
      return task;
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  /** Closes a task without doing it. Reopening goes through setTaskDone(id, false). */
  async setTaskWontDo(id: string): Promise<Task> {
    try {
      const task = await this.mutate(api.setTaskWontDo(id));
      this.notify('Tâche classée comme pas à faire.', 'default');
      return task;
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async setTaskStarted(id: string, started: boolean): Promise<Task> {
    try {
      const task = await this.mutate(api.setTaskStarted(id, started));
      this.notify(started ? 'Tâche en cours.' : 'Tâche remise en attente.', 'default');
      return task;
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  /** Pushes a task back to tomorrow (or Monday after the weekend) and pauses it. */
  async deferTask(task: Task): Promise<Task> {
    const { label } = resumeDay();
    try {
      if (task.startedAt !== null) {
        await api.setTaskStarted(task.id, false);
      }
      const updated = await this.mutate(
        api.updateTask(task.id, {
          title: task.title,
          description: task.description,
          dueAt: resumeDueAt(task),
          profileId: task.profileId,
        }),
      );
      this.notify(`Tâche reportée à ${label}.`, 'default');
      return updated;
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async deleteTask(id: string): Promise<void> {
    try {
      await this.mutate(api.deleteTask(id), true);
      this.notify('Tâche supprimée.', 'default');
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  /** Persists the manual order of the open tasks of the active profile. */
  async reorderTasks(ids: string[]): Promise<void> {
    try {
      await this.mutate(api.reorderTasks(ids));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async addSubtask(taskId: string, title: string): Promise<Task> {
    try {
      return await this.mutate(api.addSubtask(taskId, title));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async addSubtasks(taskId: string, titles: string[]): Promise<Task> {
    try {
      return await this.mutate(api.addSubtasks(taskId, titles));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async updateSubtask(id: string, title: string): Promise<Task> {
    try {
      return await this.mutate(api.updateSubtask(id, title));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async setSubtaskDone(id: string, done: boolean): Promise<Task> {
    try {
      return await this.mutate(api.setSubtaskDone(id, done));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async deleteSubtask(id: string): Promise<Task> {
    try {
      return await this.mutate(api.deleteSubtask(id));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async reorderSubtasks(taskId: string, ids: string[]): Promise<Task> {
    try {
      return await this.mutate(api.reorderSubtasks(taskId, ids));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async addAttachment(taskId: string): Promise<Task> {
    try {
      return await this.mutate(api.addAttachment(taskId));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async addAttachments(taskId: string, files: AttachmentInput[]): Promise<Task> {
    try {
      return await this.mutate(api.addAttachments(taskId, files));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async addAttachmentPaths(taskId: string, paths: string[]): Promise<Task> {
    try {
      return await this.mutate(api.addAttachmentPaths(taskId, paths));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async deleteAttachment(id: string): Promise<Task> {
    try {
      return await this.mutate(api.deleteAttachment(id));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async openAttachment(id: string): Promise<void> {
    try {
      await api.openAttachment(id);
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  /** Creates a task due today, used by the quick-add field on the Today view. */
  async quickAdd(title: string): Promise<Task | null> {
    const clean = title.trim();
    if (!clean) {
      return null;
    }
    return this.createTask({
      title: clean,
      description: '',
      dueAt: endOfToday(),
      profileId: this.activeProfile()?.id ?? '',
    });
  }

  async breakDown(taskId: string): Promise<string[]> {
    return api.breakDownTask(taskId);
  }

  async requestNudge(silent = false): Promise<void> {
    if (!this.coachActive() || this.nudgeLoading()) {
      return;
    }
    this.nudgeLoading.set(true);
    this.nudgeError.set(null);
    try {
      const text = await api.coachNudge();
      if (!this.coachActive()) {
        return;
      }
      this.nudge.set({ text, at: Date.now() });
      if (silent) {
        this.notify('Le coach a un mot pour toi.', 'accent');
      }
    } catch (error) {
      const message = errorMessage(error);
      this.nudgeError.set(message);
      if (!silent) {
        this.notify(message, 'error');
      }
    } finally {
      this.nudgeLoading.set(false);
    }
  }

  async openDataFolder(): Promise<void> {
    try {
      await api.openDataFolder();
    } catch (error) {
      this.fail(error);
    }
  }

  async openSyncFolder(): Promise<void> {
    try {
      await api.openSyncFolder();
    } catch (error) {
      this.fail(error);
    }
  }

  notify(text: string, tone: ToastTone = 'default'): void {
    if (this.toastTimer) {
      clearTimeout(this.toastTimer);
    }
    const message: ToastMessage = { id: Date.now(), text, tone };
    this.toast.set(message);
    this.toastTimer = setTimeout(() => {
      if (this.toast()?.id === message.id) {
        this.toast.set(null);
      }
    }, 4200);
  }

  dismissToast(): void {
    this.toast.set(null);
  }
}
