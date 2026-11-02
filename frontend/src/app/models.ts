export interface Subtask {
  id: string;
  title: string;
  done: boolean;
  position: number;
}

export interface Profile {
  id: string;
  name: string;
  color: string;
  position: number;
  createdAt: number;
}

export interface TaskImage {
  id: string;
  name: string;
  dataUrl: string;
  createdAt: number;
}

export interface Task {
  id: string;
  profileId: string;
  title: string;
  description: string;
  done: boolean;
  position: number;
  createdAt: number;
  dueAt: number | null;
  completedAt: number | null;
  startedAt: number | null;
  timeSpent: number;
  subtasks: Subtask[];
  images: TaskImage[];
}

export interface TaskInput {
  title: string;
  description: string;
  dueAt: number | null;
  profileId: string;
}

export interface Settings {
  accent: AccentName;
  coachEnabled: boolean;
  coachOnStart: boolean;
  timeTracking: boolean;
  showMomentum: boolean;
  notifications: boolean;
  autoUpdate: boolean;
  claudePath: string;
}

export interface NotificationStatus {
  available: boolean;
  authorized: boolean;
  error: string;
}

export interface ClaudeStatus {
  available: boolean;
  path: string;
  version: string;
  error: string;
}

export interface SyncStatus {
  configured: boolean;
  remote: string;
  branch: string;
  enabled: boolean;
  syncing: boolean;
  lastSyncAt: number;
  lastError: string;
  generation: number;
  gitAvailable: boolean;
}

export interface AppInfo {
  version: string;
  dataDir: string;
  database: string;
  databasePath: string;
}

export interface UpdateInfo {
  currentVersion: string;
  latestVersion: string;
  available: boolean;
  downloading: boolean;
  ready: boolean;
  notes: string;
  releaseUrl: string;
  publishedAt: number;
  error: string;
}

export type ViewName = 'today' | 'all' | 'coach' | 'history' | 'settings';
export type AccentName = 'lavande' | 'menthe' | 'ciel' | 'peche' | 'rose' | 'beurre';
export type TaskFilter = 'all' | 'overdue' | 'today' | 'undated';
export type ToastTone = 'default' | 'error' | 'accent';

export interface ToastMessage {
  id: number;
  text: string;
  tone: ToastTone;
}

export interface Stats {
  open: number;
  dueToday: number;
  overdue: number;
  doneToday: number;
  stale: number;
  completion: number;
}

export interface HistoryGroup {
  key: string;
  label: string;
  tasks: Task[];
}

export const ACCENTS: { id: AccentName; label: string; color: string; soft: string }[] = [
  { id: 'lavande', label: 'Lavande', color: '#7b6cf0', soft: '#ece8ff' },
  { id: 'menthe', label: 'Menthe', color: '#4aa985', soft: '#e3f4ec' },
  { id: 'ciel', label: 'Ciel', color: '#559fe0', soft: '#e6f1fb' },
  { id: 'peche', label: 'Pêche', color: '#ea8a62', soft: '#fceee7' },
  { id: 'rose', label: 'Rose', color: '#dd7ba2', soft: '#fbe9f0' },
  { id: 'beurre', label: 'Beurre', color: '#d3a640', soft: '#f9f1da' },
];
