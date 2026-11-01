import { computed, Injectable, signal } from '@angular/core';

import { api, errorMessage } from './api';
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
  ClaudeStatus,
  HistoryGroup,
  Settings,
  Stats,
  Task,
  TaskInput,
  ToastMessage,
  ToastTone,
} from './models';

const DAY = 86_400_000;

@Injectable({ providedIn: 'root' })
export class Store {
  readonly tasks = signal<Task[]>([]);
  readonly settings = signal<Settings>({ accent: 'lavande', coachEnabled: true, coachOnStart: false, timeTracking: true, showMomentum: true, claudePath: '' });
  readonly claude = signal<ClaudeStatus | null>(null);
  readonly info = signal<AppInfo | null>(null);
  readonly loading = signal(true);
  readonly fatalError = signal<string | null>(null);
  readonly toast = signal<ToastMessage | null>(null);
  readonly nudge = signal<{ text: string; at: number } | null>(null);
  readonly nudgeLoading = signal(false);
  readonly nudgeError = signal<string | null>(null);

  readonly openTasks = computed(() => this.tasks().filter((task) => !task.done).sort(compareOpenTasks));
  readonly startedTasks = computed(() =>
    this.openTasks()
      .filter((task) => task.startedAt !== null)
      .sort((a, b) => (a.startedAt ?? 0) - (b.startedAt ?? 0)),
  );
  /** Wall clock refreshed every 15 s so running timers update on screen. */
  readonly now = signal(Date.now());
  readonly doneTodayTasks = computed(() =>
    this.tasks()
      .filter((task) => task.done && task.completedAt !== null && isSameDay(task.completedAt, this.now()))
      .sort((a, b) => (b.completedAt ?? 0) - (a.completedAt ?? 0)),
  );
  readonly doneTasks = computed(() =>
    this.tasks()
      .filter((task) => task.done)
      .sort((a, b) => (b.completedAt ?? 0) - (a.completedAt ?? 0)),
  );
  readonly history = computed<HistoryGroup[]>(() => groupHistory(this.tasks()));

  readonly stats = computed<Stats>(() => {
    const now = Date.now();
    const open = this.tasks().filter((task) => !task.done);
    const doneToday = this.tasks().filter(
      (task) => task.done && task.completedAt !== null && isSameDay(task.completedAt, now),
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

  async init(): Promise<void> {
    setInterval(() => this.now.set(Date.now()), 15_000);
    try {
      const [tasks, settings, info] = await Promise.all([api.listTasks(), api.getSettings(), api.appInfo()]);
      this.tasks.set(tasks);
      this.info.set(info);
      this.applySettings(settings);
      void this.refreshClaude();
      if (settings.coachEnabled && settings.coachOnStart) {
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
      localStorage.setItem('todo.settings', JSON.stringify(settings));
    } catch {
      // localStorage can be unavailable in some webview contexts; harmless.
    }
  }

  applyAccent(accent: AccentName): void {
    document.documentElement.dataset['accent'] = accent;
  }

  async saveSettings(patch: Partial<Settings>): Promise<void> {
    const next: Settings = { ...this.settings(), ...patch };
    if (patch.coachEnabled === false) {
      this.nudge.set(null);
      this.nudgeError.set(null);
    }
    this.applySettings(next);
    try {
      await api.saveSettings(next);
    } catch (error) {
      this.notify(errorMessage(error), 'error');
    }
  }

  async refreshClaude(): Promise<ClaudeStatus | null> {
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
    this.tasks.set(await api.listTasks());
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
      return await this.mutate(api.createTask(input));
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

  async addImage(taskId: string): Promise<Task> {
    try {
      return await this.mutate(api.addImage(taskId));
    } catch (error) {
      this.fail(error);
      throw error;
    }
  }

  async deleteImage(id: string): Promise<Task> {
    try {
      return await this.mutate(api.deleteImage(id));
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
    return this.createTask({ title: clean, description: '', dueAt: endOfToday() });
  }

  async breakDown(taskId: string): Promise<string[]> {
    return api.breakDownTask(taskId);
  }

  async requestNudge(silent = false): Promise<void> {
    if (!this.settings().coachEnabled || this.nudgeLoading()) {
      return;
    }
    this.nudgeLoading.set(true);
    this.nudgeError.set(null);
    try {
      const text = await api.coachNudge();
      if (!this.settings().coachEnabled) {
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
