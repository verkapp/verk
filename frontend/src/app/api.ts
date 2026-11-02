import { mockCall } from './mock';
import { AppInfo, ClaudeStatus, NotificationStatus, Profile, Settings, SyncStatus, Task, TaskInput, UpdateInfo } from './models';

type Bridge = Record<string, (...args: unknown[]) => Promise<unknown>>;

interface RuntimeBridge {
  EventsOn?: (event: string, callback: (...data: unknown[]) => void) => () => void;
}

function bridge(): Bridge | null {
  const wails = (window as unknown as { go?: { src?: { App?: Bridge }; main?: { App?: Bridge } } }).go;
  return wails?.src?.App ?? wails?.main?.App ?? null;
}

async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  const app = bridge();
  if (!app) {
    return mockCall<T>(method, args);
  }
  const fn = app[method];
  if (typeof fn !== 'function') {
    throw new Error(`La méthode « ${method} » n'existe pas dans le backend.`);
  }
  return (await fn(...args)) as T;
}

export const api = {
  listTasks: () => call<Task[]>('ListTasks'),
  createTask: (input: TaskInput) => call<Task>('CreateTask', input),
  updateTask: (id: string, input: TaskInput) => call<Task>('UpdateTask', id, input),
  setTaskDone: (id: string, done: boolean) => call<Task>('SetTaskDone', id, done),
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

  addImage: (taskId: string) => call<Task>('AddImage', taskId),
  deleteImage: (id: string) => call<Task>('DeleteImage', id),

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
  const runtime = (window as unknown as { runtime?: RuntimeBridge }).runtime;
  if (!runtime?.EventsOn) {
    return () => undefined;
  }
  return runtime.EventsOn(event, (...data) => callback(data[0] as T));
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
