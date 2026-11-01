import { HistoryGroup, Task } from './models';

const DAY = 86_400_000;
const longDate = new Intl.DateTimeFormat('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' });
const longDateWithYear = new Intl.DateTimeFormat('fr-FR', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
});
const shortDate = new Intl.DateTimeFormat('fr-FR', { day: 'numeric', month: 'short' });
const shortDateWithYear = new Intl.DateTimeFormat('fr-FR', { day: 'numeric', month: 'short', year: 'numeric' });
const weekdayShort = new Intl.DateTimeFormat('fr-FR', { weekday: 'short' });
const timeFmt = new Intl.DateTimeFormat('fr-FR', { hour: '2-digit', minute: '2-digit' });

export function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export function startOfDay(value: number | Date): Date {
  const date = new Date(value);
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

export function endOfDay(value: number | Date): Date {
  const date = startOfDay(value);
  date.setHours(23, 59, 0, 0);
  return date;
}

export function endOfToday(): number {
  return endOfDay(new Date()).getTime();
}

export function isSameDay(a: number | Date, b: number | Date): boolean {
  const da = new Date(a);
  const db = new Date(b);
  return (
    da.getFullYear() === db.getFullYear() && da.getMonth() === db.getMonth() && da.getDate() === db.getDate()
  );
}

export function isToday(value: number): boolean {
  return isSameDay(value, new Date());
}

export function isOverdue(task: Task, now = Date.now()): boolean {
  return !task.done && task.dueAt !== null && task.dueAt < now;
}

export function isDueToday(task: Task): boolean {
  return !task.done && task.dueAt !== null && isToday(task.dueAt);
}

export function hasSpecificTime(value: number): boolean {
  const date = new Date(value);
  return !(date.getHours() === 23 && date.getMinutes() === 59);
}

export function formatTime(value: number): string {
  return timeFmt.format(new Date(value));
}

export function formatLongDate(value: number | Date): string {
  const date = new Date(value);
  const sameYear = date.getFullYear() === new Date().getFullYear();
  return capitalize((sameYear ? longDate : longDateWithYear).format(date));
}

export function formatShortDate(value: number): string {
  const date = new Date(value);
  return (date.getFullYear() === new Date().getFullYear() ? shortDate : shortDateWithYear).format(date);
}

export function formatCreatedAt(value: number): string {
  return `${formatShortDate(value)} à ${formatTime(value)}`;
}

export function dayKey(value: number): string {
  const date = new Date(value);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

export function dayLabel(value: number): string {
  const date = new Date(value);
  const today = startOfDay(new Date()).getTime();
  const day = startOfDay(date).getTime();
  if (day === today) {
    return "Aujourd'hui";
  }
  if (day === today - DAY) {
    return 'Hier';
  }
  const sameYear = date.getFullYear() === new Date().getFullYear();
  return capitalize((sameYear ? longDate : longDateWithYear).format(date));
}

export interface DueMeta {
  label: string;
  tone: 'danger' | 'today' | 'soon' | 'later';
  full: string;
}

export function dueMeta(task: Task, now = Date.now()): DueMeta | null {
  if (task.dueAt === null) {
    return null;
  }
  const due = task.dueAt;
  const time = hasSpecificTime(due) ? ` ${formatTime(due)}` : '';
  const full = `Échéance : ${formatShortDate(due)}${hasSpecificTime(due) ? ` à ${formatTime(due)}` : ''}`;

  if (due < now && !task.done) {
    return { label: 'En retard', tone: 'danger', full };
  }

  const today = startOfDay(now).getTime();
  const dueDay = startOfDay(due).getTime();
  if (dueDay === today) {
    return { label: `Aujourd'hui${time}`, tone: 'today', full };
  }
  if (dueDay === today + DAY) {
    return { label: `Demain${time}`, tone: 'soon', full };
  }
  if (dueDay > today && dueDay <= today + 6 * DAY) {
    return { label: `${capitalize(weekdayShort.format(new Date(due)))} ${formatShortDate(due)}${time}`, tone: 'soon', full };
  }
  return { label: `${formatShortDate(due)}${time}`, tone: 'later', full };
}

/** Day a task is picked back up: Monday when the weekend is ahead, tomorrow otherwise. */
export function resumeDay(now = Date.now()): { date: Date; label: 'demain' | 'lundi' } {
  const today = new Date(now);
  const weekday = today.getDay();
  const daysAhead = weekday === 5 ? 3 : weekday === 6 ? 2 : 1;
  const date = new Date(today.getFullYear(), today.getMonth(), today.getDate() + daysAhead);
  return { date, label: daysAhead === 1 ? 'demain' : 'lundi' };
}

/** Due timestamp for "reprendre": keeps a specific time, otherwise end of day. */
export function resumeDueAt(task: Task, now = Date.now()): number {
  const { date } = resumeDay(now);
  if (task.dueAt !== null && hasSpecificTime(task.dueAt)) {
    const time = new Date(task.dueAt);
    date.setHours(time.getHours(), time.getMinutes(), 0, 0);
  } else {
    date.setHours(23, 59, 0, 0);
  }
  return date.getTime();
}

export function createdAgo(value: number, now = Date.now()): string {
  if (isToday(value)) {
    return "aujourd'hui";
  }
  const days = Math.floor((startOfDay(now).getTime() - startOfDay(value).getTime()) / DAY);
  if (days === 1) {
    return 'hier';
  }
  if (days < 7) {
    return `il y a ${days} j`;
  }
  if (days < 30) {
    return `il y a ${Math.floor(days / 7)} sem.`;
  }
  return `le ${formatShortDate(value)}`;
}

export function compareOpenTasks(a: Task, b: Task): number {
  if (a.done !== b.done) {
    return a.done ? 1 : -1;
  }
  if (a.done) {
    return (b.completedAt ?? 0) - (a.completedAt ?? 0);
  }
  if ((a.dueAt === null) !== (b.dueAt === null)) {
    return a.dueAt === null ? 1 : -1;
  }
  if (a.dueAt !== null && b.dueAt !== null && a.dueAt !== b.dueAt) {
    return a.dueAt - b.dueAt;
  }
  return b.createdAt - a.createdAt;
}

export function sortOpen(tasks: Task[]): Task[] {
  return [...tasks].sort(compareOpenTasks);
}

export function groupHistory(tasks: Task[]): HistoryGroup[] {
  const groups = new Map<string, Task[]>();
  const sorted = [...tasks]
    .filter((task) => task.done && task.completedAt !== null)
    .sort((a, b) => (b.completedAt ?? 0) - (a.completedAt ?? 0));

  for (const task of sorted) {
    const key = dayKey(task.completedAt as number);
    const bucket = groups.get(key);
    if (bucket) {
      bucket.push(task);
    } else {
      groups.set(key, [task]);
    }
  }

  return [...groups.entries()].map(([key, groupTasks]) => ({
    key,
    label: dayLabel(groupTasks[0].completedAt as number),
    tasks: groupTasks,
  }));
}

/** Splits a timestamp into the values expected by native date/time inputs. */
export function toDueInputs(value: number | null): { date: string; time: string } {
  if (value === null) {
    return { date: '', time: '' };
  }
  const date = new Date(value);
  const pad = (n: number) => String(n).padStart(2, '0');
  return {
    date: `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`,
    time: hasSpecificTime(value) ? `${pad(date.getHours())}:${pad(date.getMinutes())}` : '',
  };
}

/** Empty time means "end of day", keeping the date-only case clean. */
export function fromDueInputs(date: string, time: string): number | null {
  if (!date) {
    return null;
  }
  const [year, month, day] = date.split('-').map(Number);
  if (time) {
    const [hours, minutes] = time.split(':').map(Number);
    return new Date(year, month - 1, day, hours, minutes, 0, 0).getTime();
  }
  return new Date(year, month - 1, day, 23, 59, 0, 0).getTime();
}

export function subtaskProgress(task: Task): number {
  if (task.subtasks.length === 0) {
    return 0;
  }
  return Math.round((task.subtasks.filter((subtask) => subtask.done).length / task.subtasks.length) * 100);
}

/** Total tracked time in ms, including the currently running session. */
export function elapsedMs(task: Task, now = Date.now()): number {
  return task.timeSpent + (task.startedAt !== null ? Math.max(0, now - task.startedAt) : 0);
}

export function formatDuration(ms: number): string {
  const minutes = Math.floor(ms / 60_000);
  if (minutes < 1) {
    return '< 1 min';
  }
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return rest === 0 ? `${hours} h` : `${hours} h ${String(rest).padStart(2, '0')}`;
}
