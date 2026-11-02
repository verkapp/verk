import { ChangeDetectionStrategy, Component, computed, inject, input, output } from '@angular/core';

import { createdAgo, dueMeta, elapsedMs, formatDuration, resumeDay, subtaskProgress } from '../dates';
import { Task } from '../models';
import { Store } from '../store';
import { IconComponent } from '../ui/icon.component';

@Component({
  selector: 'app-task-card',
  standalone: true,
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './task-card.component.html',
  styleUrl: './task-card.component.css',
})
export class TaskCardComponent {
  readonly store = inject(Store);
  readonly task = input.required<Task>();
  readonly reorderable = input(false);
  readonly dragging = input(false);
  readonly dropPosition = input<'before' | 'after' | null>(null);
  readonly opened = output<void>();
  readonly toggled = output<void>();
  readonly started = output<void>();
  readonly deferred = output<void>();

  private suppressClick = false;

  readonly due = computed(() => dueMeta(this.task()));
  readonly resumeLabel = computed(() => resumeDay(this.store.now()).label);
  readonly progress = computed(() => subtaskProgress(this.task()));
  readonly doneSubtasks = computed(() => this.task().subtasks.filter((subtask) => subtask.done).length);
  readonly age = computed(() => createdAgo(this.task().createdAt));

  readonly tracked = computed(() => {
    if (!this.store.settings().timeTracking) {
      return null;
    }
    const task = this.task();
    const ms = elapsedMs(task, this.store.now());
    return ms > 0 || task.startedAt !== null ? formatDuration(ms) : null;
  });

  onToggle(event: Event): void {
    event.stopPropagation();
    this.toggled.emit();
  }

  onClick(): void {
    if (this.suppressClick) {
      return;
    }
    this.opened.emit();
  }

  onDragStart(): void {
    this.suppressClick = true;
  }

  onDragEnd(): void {
    // The click that follows a drag must not open the drawer.
    setTimeout(() => {
      this.suppressClick = false;
    });
  }

  onStart(event: Event): void {
    event.stopPropagation();
    this.started.emit();
  }

  onDefer(event: Event): void {
    event.stopPropagation();
    this.deferred.emit();
  }

  onKeydown(event: Event): void {
    if (event.target !== event.currentTarget) {
      return;
    }
    event.preventDefault();
    this.opened.emit();
  }
}
