import { ChangeDetectionStrategy, Component, computed, inject, output } from '@angular/core';

import { formatTime } from '../dates';
import { Task } from '../models';
import { Store } from '../store';
import { IconComponent } from '../ui/icon.component';

@Component({
  selector: 'app-history',
  standalone: true,
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './history.component.html',
  styleUrl: './history.component.css',
})
export class HistoryComponent {
  readonly store = inject(Store);
  readonly openTask = output<string>();

  readonly groups = this.store.history;
  readonly total = computed(() => this.store.doneTasks().length);
  readonly thisWeek = computed(() => {
    const weekAgo = Date.now() - 7 * 86_400_000;
    return this.store.doneTasks().filter((task) => (task.completedAt ?? 0) >= weekAgo).length;
  });

  formatTime = formatTime;

  toggle(event: Event, task: Task): void {
    event.stopPropagation();
    void this.store.setTaskDone(task.id, false).catch(() => undefined);
  }

  onKeydown(event: Event, task: Task): void {
    if (event.target !== event.currentTarget) {
      return;
    }
    event.preventDefault();
    this.openTask.emit(task.id);
  }
}
