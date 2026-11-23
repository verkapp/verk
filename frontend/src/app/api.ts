import { Call, Events } from '@wailsio/runtime';

import { mockCall } from './mock';
import {
  AppInfo,
  AttachmentInput,
  ClaudeStatus,
  NotificationStatus,
  Profile,
  Settings,
  SyncStatus,
  Task,
  TaskInput,
  UpdateInfo,
} from './models';

/** Fully qualified Go type that the Wails v3 service binds (package.Type). */
const SERVICE = 'todo/src.App';

/**
 * True inside the desktop app. The plain `ng serve` preview has no native
 * host, so calls fall back to the in-memory mock.
 */
function inDesktopHost(): boolean {
  const w = window as unknown as {
    chrome?: { webview?: { postMessage?: unknown } };
    webkit?: { messageHandlers?: { external?: { postMessage?: unknown } } };
  };
  return Boolean(w.chrome?.webview?.postMessage || w.webkit?.messageHandlers?.external?.postMessage);
}

/**
 * True in the macOS desktop app, where the title bar is merged with the page.
 * Checked synchronously from the WebView user agent: the runtime's
 * System.IsMac() is only filled in after startup, too late for the layout.
 */
export function isMacDesktop(): boolean {
  return inDesktopHost() && /Macintosh/.test(navigator.userAgent);
}

async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  if (!inDesktopHost()) {
    return mockCall<T>(method, args);
  }
  return (await Call.ByName(`${SERVICE}.${method}`, ...args)) as T;
}

export const api = {
  listTasks: () => call<Task[]>('ListTasks'),
  createTask: (input: TaskInput) => call<Task>('CreateTask', input),
  updateTask: (id: string, input: TaskInput) => call<Task>('UpdateTask', id, input),
  setTaskDone: (id: string, done: boolean) => call<Task>('SetTaskDone', id, done),
  setTaskWontDo: (id: string) => call<Task>('SetTaskWontDo', id),
  setTaskStarted: (id: string, started: boolean) => call<Task>('SetTaskStarted', id, started),
  deleteTask: (id: string) => call<void>('DeleteTask', id),
  reorderTasks: (ids: string[]) => call<void>('ReorderTasks', ids),

  addSubtask: (taskId: string, title: string) => call<Task>('AddSubtask', taskId, title),
  addSubtasks: (taskId: string, titles: string[]) => call<Task>('AddSubtasks', taskId, titles),
  updateSubtask: (id: string, title: string) => call<Task>('UpdateSubtask', id, title),
  setSubtaskDone: (id: string, done: boolean) => call<Task>('SetSubtaskDone', id, done),
  deleteSubtask: (id: string) => call<Task>('DeleteSubtask', id),
  reorderSubtasks: (taskId: string, ids: string[]) => call<Task>('ReorderSubtasks', taskId, ids),

  listProfiles: () => call<Profile[]>('ListProfiles'),
  createProfile: (name: string) => call<Profile>('CreateProfile', name),
  updateProfile: (id: string, name: string, color: string) => call<Profile>('UpdateProfile', id, name, color),
  deleteProfile: (id: string) => call<void>('DeleteProfile', id),

  addAttachment: (taskId: string) => call<Task>('AddAttachment', taskId),
  addAttachments: (taskId: string, files: AttachmentInput[]) =>
    call<Task>('AddAttachments', taskId, files),
  addAttachmentPaths: (taskId: string, paths: string[]) =>
    call<Task>('AddAttachmentPaths', taskId, paths),
  deleteAttachment: (id: string) => call<Task>('DeleteAttachment', id),
  openAttachment: (id: string) => call<void>('OpenAttachment', id),

  getSettings: () => call<Settings>('GetSettings'),
  saveSettings: (settings: Settings) => call<void>('SaveSettings', settings),

  claudeStatus: () => call<ClaudeStatus>('ClaudeStatus'),
  breakDownTask: (taskId: string) => call<string[]>('BreakDownTask', taskId),
  coachNudge: () => call<string>('CoachNudge'),

  notificationStatus: () => call<NotificationStatus>('NotificationStatus'),
  requestNotifications: () => call<NotificationStatus>('RequestNotifications'),
  testNotification: () => call<void>('TestNotification'),

  openDataFolder: () => call<void>('OpenDataFolder'),
  appInfo: () => call<AppInfo>('AppInfo'),

  updateStatus: () => call<UpdateInfo>('UpdateStatus'),
  checkForUpdates: () => call<UpdateInfo>('CheckForUpdates'),
  downloadUpdate: () => call<UpdateInfo>('DownloadUpdate'),
  installUpdate: () => call<void>('InstallUpdate'),
  openUpdatePage: () => call<void>('OpenUpdatePage'),

  syncStatus: () => call<SyncStatus>('SyncStatus'),
  setSyncConfig: (remote: string, enabled: boolean) => call<void>('SetSyncConfig', remote, enabled),
  syncNow: () => call<SyncStatus>('SyncNow'),
  openSyncFolder: () => call<void>('OpenSyncFolder'),
};

/** Subscribes to a Wails event. Returns a no-op in the browser preview. */
export function onRuntimeEvent<T>(event: string, callback: (payload: T) => void): () => void {
  if (!inDesktopHost()) {
    return () => undefined;
  }
  return Events.On(event, (ev) => callback(ev.data as T));
}

export function errorMessage(error: unknown): string {
  if (typeof error === 'string') {
    return error;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'Une erreur inattendue est survenue.';
}
