import { ChangeDetectionStrategy, Component, computed, inject, OnInit, output, signal } from '@angular/core';

import { errorMessage } from '../api';
import { dueMeta } from '../dates';
import { Task } from '../models';
import { Store } from '../store';
import { IconComponent } from '../ui/icon.component';

@Component({
  selector: 'app-coach',
  standalone: true,
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './coach.component.html',
  styleUrl: './coach.component.css',
})
export class CoachComponent implements OnInit {
  readonly store = inject(Store);
  readonly openTask = output<string>();
  readonly openSettings = output<void>();

  readonly selectedId = signal('');
  readonly suggestions = signal<string[]>([]);
  readonly breakdownLoading = signal(false);
  readonly breakdownError = signal<string | null>(null);

  readonly selectedTask = computed(() => this.store.openTasks().find((task) => task.id === this.selectedId()) ?? null);

  ngOnInit(): void {
    if (!this.store.claude()) {
      void this.store.refreshClaude();
    }
  }

  dueLabel(task: Task): string | null {
    return dueMeta(task)?.label ?? null;
  }

  dueTone(task: Task): string {
    return dueMeta(task)?.tone ?? 'later';
  }

  nudgeAge(): string {
    const nudge = this.store.nudge();
    if (!nudge) {
      return '';
    }
    const minutes = Math.floor((Date.now() - nudge.at) / 60_000);
    if (minutes < 1) {
      return 'à l’instant';
    }
    if (minutes < 60) {
      return `il y a ${minutes} min`;
    }
    return `il y a ${Math.floor(minutes / 60)} h`;
  }

  requestNudge(): void {
    void this.store.requestNudge();
  }

  selectTask(value: string): void {
    this.selectedId.set(value);
    this.suggestions.set([]);
    this.breakdownError.set(null);
  }

  async breakdown(): Promise<void> {
    const task = this.selectedTask();
    if (!task || this.breakdownLoading()) {
      return;
    }
    this.breakdownLoading.set(true);
    this.breakdownError.set(null);
    this.suggestions.set([]);
    try {
      const all = await this.store.breakDown(task.id);
      const existing = new Set(task.subtasks.map((subtask) => subtask.title.toLowerCase()));
      const fresh = all.filter((suggestion) => !existing.has(suggestion.toLowerCase()));
      if (fresh.length === 0) {
        this.breakdownError.set('Rien de nouveau à ajouter. Essaie avec une description plus précise.');
      }
      this.suggestions.set(fresh);
    } catch (error) {
      this.breakdownError.set(errorMessage(error));
    } finally {
      this.breakdownLoading.set(false);
    }
  }

  async addSuggestion(suggestion: string): Promise<void> {
    const task = this.selectedTask();
    if (!task) {
      return;
    }
    try {
      await this.store.addSubtask(task.id, suggestion);
      this.suggestions.update((items) => items.filter((item) => item !== suggestion));
    } catch {
      // handled by the store
    }
  }

  async addAll(): Promise<void> {
    const task = this.selectedTask();
    if (!task || this.suggestions().length === 0) {
      return;
    }
    const titles = this.suggestions();
    try {
      await this.store.addSubtasks(task.id, titles);
      this.suggestions.set([]);
      this.store.notify(
        `${titles.length} sous-tâche${titles.length > 1 ? 's ajoutées' : ' ajoutée'}.`,
        'accent',
      );
    } catch {
      // handled by the store
    }
  }
}
