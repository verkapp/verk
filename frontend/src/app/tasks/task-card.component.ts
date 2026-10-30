import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';

import { createdAgo, dueMeta, subtaskProgress } from '../dates';
import { Task } from '../models';
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
  readonly task = input.required<Task>();
  readonly opened = output<void>();
  readonly toggled = output<void>();

  readonly due = computed(() => dueMeta(this.task()));
  readonly progress = computed(() => subtaskProgress(this.task()));
  readonly doneSubtasks = computed(() => this.task().subtasks.filter((subtask) => subtask.done).length);
  readonly age = computed(() => createdAgo(this.task().createdAt));

  onToggle(event: Event): void {
    event.stopPropagation();
    this.toggled.emit();
  }

  onKeydown(event: Event): void {
    if (event.target !== event.currentTarget) {
      return;
    }
    event.preventDefault();
    this.opened.emit();
  }
}
