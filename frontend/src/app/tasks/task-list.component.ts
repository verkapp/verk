import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';

import { CollapsedSections } from '../collapse';
import { formatLongDate, isDueToday, isOverdue, sortOpen } from '../dates';
import { Task, TaskFilter } from '../models';
import { Store } from '../store';
import { IconComponent } from '../ui/icon.component';
import { TaskCardComponent } from './task-card.component';

interface TaskGroup {
  key: string;
  label: string;
  tasks: Task[];
}

@Component({
  selector: 'app-task-list',
  standalone: true,
  imports: [IconComponent, TaskCardComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './task-list.component.html',
  styleUrl: './task-list.component.css',
})
export class TaskListComponent {
  readonly store = inject(Store);
  readonly sections = new CollapsedSections('todo.sections.tasks');
  readonly mode = input<'today' | 'all'>('today');
  readonly openTask = output<string>();
  readonly newTask = output<void>();
  readonly openCoach = output<void>();

  readonly quickTitle = signal('');
  readonly filter = signal<TaskFilter>('all');
  readonly bannerDismissed = signal(false);
  readonly draggedId = signal<string | null>(null);
  readonly dragList = signal<string>('');
  readonly dropTarget = signal<{ id: string; after: boolean } | null>(null);

  readonly filters: { id: TaskFilter; label: string }[] = [
    { id: 'all', label: 'Toutes' },
    { id: 'overdue', label: 'En retard' },
    { id: 'today', label: "Aujourd'hui" },
    { id: 'undated', label: 'Sans date' },
  ];

  readonly title = computed(() => (this.mode() === 'today' ? "Aujourd'hui" : 'Toutes les tâches'));
  readonly eyebrow = computed(() => (this.mode() === 'today' ? formatLongDate(Date.now()) : 'Ma liste'));
  readonly isTodayMode = computed(() => this.mode() === 'today');

  readonly counts = computed<Record<TaskFilter, number>>(() => {
    const open = this.store.openTasks();
    return {
      all: open.length,
      overdue: open.filter((task) => isOverdue(task)).length,
      today: open.filter((task) => isDueToday(task) && !isOverdue(task)).length,
      undated: open.filter((task) => task.dueAt === null).length,
    };
  });

  readonly todayGroups = computed<TaskGroup[]>(() => {
    const now = Date.now();
    const open = this.store.openTasks().filter((task) => task.startedAt === null);
    const overdue = open.filter((task) => isOverdue(task, now));
    const dueToday = open.filter((task) => !isOverdue(task, now) && isDueToday(task));
    const upcoming = open.filter(
      (task) => task.dueAt !== null && !isOverdue(task, now) && !isDueToday(task) && task.dueAt > now,
    );
    const undated = open.filter((task) => task.dueAt === null);

    return [
      { key: 'overdue', label: 'En retard', tasks: sortOpen(overdue) },
      { key: 'today', label: "Aujourd'hui", tasks: sortOpen(dueToday) },
      { key: 'upcoming', label: 'À venir', tasks: sortOpen(upcoming) },
      { key: 'undated', label: 'Sans date', tasks: sortOpen(undated) },
    ].filter((group) => group.tasks.length > 0);
  });

  readonly filteredTasks = computed(() => {
    const open = this.store.openTasks().filter((task) => task.startedAt === null);
    switch (this.filter()) {
      case 'overdue':
        return open.filter((task) => isOverdue(task));
      case 'today':
        return open.filter((task) => isDueToday(task) && !isOverdue(task));
      case 'undated':
        return open.filter((task) => task.dueAt === null);
      default:
        return open;
    }
  });

  readonly showBanner = computed(
    () =>
      this.mode() === 'today' &&
      this.store.coachActive() &&
      !this.bannerDismissed() &&
      this.store.attentionTasks().length > 0,
  );

  readonly bannerTitle = computed(() => {
    const overdue = this.store.stats().overdue;
    return overdue > 0 ? 'Un petit coup de pouce ?' : 'On garde le cap ?';
  });

  readonly bannerText = computed(() => {
    const overdue = this.store.stats().overdue;
    if (overdue > 0) {
      return `${overdue} tâche${overdue > 1 ? 's ont' : ' a'} dépassé l’échéance. Claude peut t’aider à t’y remettre.`;
    }
    return 'Quelques tâches attendent depuis un moment. Un mot du coach peut aider à choisir la prochaine.';
  });

  readonly subtitle = computed(() => {
    const open = this.store.stats().open;
    const done = this.store.doneTasks().length;
    if (this.mode() === 'today') {
      return open === 0 ? 'Rien d’ouvert, la journée est à toi.' : `${open} tâche${open > 1 ? 's' : ''} ouverte${open > 1 ? 's' : ''} dans ta liste.`;
    }
    return `${open} ouverte${open > 1 ? 's' : ''} · ${done} terminée${done > 1 ? 's' : ''}`;
  });

  readonly emptyTitle = computed(() => {
    if (this.mode() === 'today') {
      return this.store.openTasks().length === 0 ? 'Tout est calme' : 'Rien ici';
    }
    switch (this.filter()) {
      case 'overdue':
        return 'Rien en retard';
      case 'today':
        return 'Rien pour aujourd’hui';
      case 'undated':
        return 'Aucune tâche sans date';
      default:
        return 'Ta liste est vide';
    }
  });

  readonly emptyText = computed(() => {
    if (this.mode() === 'today') {
      return this.store.openTasks().length === 0
        ? 'Rien d’ouvert pour l’instant. Ajoute une tâche, ou savoure le silence.'
        : 'Cette vue est vide, mais ta liste contient encore des choses.';
    }
    switch (this.filter()) {
      case 'overdue':
        return 'Tu es pile à jour. Profite de cet élan.';
      case 'today':
        return 'Aucune échéance aujourd’hui. Tu peux respirer.';
      case 'undated':
        return 'Toutes tes tâches ont une date. C’est très cadré.';
      default:
        return 'Ajoute une première tâche, même toute petite.';
    }
  });

  add(event: Event): void {
    event.preventDefault();
    const title = this.quickTitle().trim();
    if (!title) {
      return;
    }
    this.quickTitle.set('');
    const promise =
      this.mode() === 'today'
        ? this.store.quickAdd(title)
        : this.store.createTask({
            title,
            description: '',
            dueAt: null,
            profileId: this.store.activeProfile()?.id ?? '',
          });
    void promise
      .then((task) => {
        if (task) {
          this.store.notify('Tâche ajoutée.', 'accent');
        }
      })
      .catch(() => undefined);
  }

  toggle(task: Task): void {
    void this.store.setTaskDone(task.id, !task.done).catch(() => undefined);
  }

  toggleStarted(task: Task): void {
    void this.store.setTaskStarted(task.id, task.startedAt === null).catch(() => undefined);
  }

  defer(task: Task): void {
    void this.store.deferTask(task).catch(() => undefined);
  }

  setFilter(filter: TaskFilter): void {
    this.filter.set(filter);
  }

  /* ----------------------------- réordonnancement ---------------------------- */

  private cardFrom(event: Event): HTMLElement | null {
    const target = event.target as HTMLElement | null;
    return (target?.closest?.('.task-card[data-task-id]') as HTMLElement | null) ?? null;
  }

  onDragStart(event: DragEvent, listKey: string): void {
    const id = this.cardFrom(event)?.dataset['taskId'];
    if (!id) {
      event.preventDefault();
      return;
    }
    this.draggedId.set(id);
    this.dragList.set(listKey);
    this.dropTarget.set(null);
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move';
      event.dataTransfer.setData('text/plain', id);
    }
  }

  onDragOver(event: DragEvent, listKey: string): void {
    if (!this.draggedId() || this.dragList() !== listKey) {
      return;
    }
    const card = this.cardFrom(event);
    if (!card) {
      return;
    }
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move';
    }
    const id = card.dataset['taskId'];
    if (!id || id === this.draggedId()) {
      this.dropTarget.set(null);
      return;
    }
    const rect = card.getBoundingClientRect();
    this.dropTarget.set({ id, after: event.clientY > rect.top + rect.height / 2 });
  }

  onDrop(event: DragEvent, listKey: string): void {
    if (!this.draggedId() || this.dragList() !== listKey) {
      return;
    }
    event.preventDefault();
    const dragged = this.draggedId();
    const target = this.dropTarget();
    this.resetDrag();
    if (!dragged || !target || dragged === target.id) {
      return;
    }
    const ids = this.store
      .openTasks()
      .map((task) => task.id)
      .filter((id) => id !== dragged);
    const targetIndex = ids.indexOf(target.id);
    if (targetIndex < 0) {
      return;
    }
    ids.splice(targetIndex + (target.after ? 1 : 0), 0, dragged);
    void this.store.reorderTasks(ids).catch(() => undefined);
  }

  onDragEnd(): void {
    this.resetDrag();
  }

  dropPositionFor(id: string): 'before' | 'after' | null {
    const target = this.dropTarget();
    if (!target || target.id !== id) {
      return null;
    }
    return target.after ? 'after' : 'before';
  }

  private resetDrag(): void {
    this.draggedId.set(null);
    this.dragList.set('');
    this.dropTarget.set(null);
  }
}
