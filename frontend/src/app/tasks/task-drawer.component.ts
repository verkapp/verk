import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  HostListener,
  computed,
  inject,
  input,
  OnInit,
  output,
  signal,
  viewChild,
} from '@angular/core';

import { errorMessage } from '../api';
import { formatCreatedAt, fromDueInputs, resumeDay, toDueInputs } from '../dates';
import { Subtask, Task, TaskImage } from '../models';
import { Store } from '../store';
import { IconComponent } from '../ui/icon.component';

@Component({
  selector: 'app-task-drawer',
  standalone: true,
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './task-drawer.component.html',
  styleUrl: './task-drawer.component.css',
})
export class TaskDrawerComponent implements OnInit, AfterViewInit {
  readonly store = inject(Store);

  readonly task = input<Task | null>(null);
  readonly presetDue = input<number | null>(null);
  readonly closed = output<void>();

  readonly title = signal('');
  readonly description = signal('');
  readonly dueDate = signal('');
  readonly dueTime = signal('');
  readonly profileId = signal('');
  readonly taskId = signal<string | null>(null);
  readonly createdAt = signal<number | null>(null);
  readonly isDone = signal(false);
  readonly subtasks = signal<Subtask[]>([]);
  readonly images = signal<TaskImage[]>([]);

  readonly saving = signal(false);
  readonly savedAt = signal<number | null>(null);
  readonly dirty = signal(false);
  readonly confirmDelete = signal(false);
  readonly lightbox = signal<string | null>(null);

  readonly suggestions = signal<string[]>([]);
  readonly breakingDown = signal(false);
  readonly aiError = signal<string | null>(null);
  readonly addSubtaskTitle = signal('');
  readonly draggedSubtaskId = signal<string | null>(null);
  readonly subtaskDrop = signal<{ id: string; after: boolean } | null>(null);

  readonly titleInput = viewChild<ElementRef<HTMLTextAreaElement>>('titleInput');

  readonly doneSubtasks = computed(() => this.subtasks().filter((subtask) => subtask.done).length);
  readonly progress = computed(() =>
    this.subtasks().length === 0
      ? 0
      : Math.round((this.doneSubtasks() / this.subtasks().length) * 100),
  );
  readonly createdLabel = computed(() => {
    const created = this.createdAt();
    return created ? `Créée le ${formatCreatedAt(created)}` : 'À la création';
  });
  readonly dueAt = computed(() => fromDueInputs(this.dueDate(), this.dueTime()));
  readonly resumeLabel = computed(() => resumeDay().label);

  private saveTimer: ReturnType<typeof setTimeout> | null = null;
  private saveQueue: Promise<Task | null> = Promise.resolve(null);
  private confirmTimer: ReturnType<typeof setTimeout> | null = null;

  ngOnInit(): void {
    const task = this.task();
    if (task) {
      this.hydrate(task);
    } else {
      const due = toDueInputs(this.presetDue());
      this.dueDate.set(due.date);
      this.dueTime.set(due.time);
      this.profileId.set(this.store.activeProfile()?.id ?? '');
    }
  }

  ngAfterViewInit(): void {
    const element = this.titleInput()?.nativeElement;
    if (element) {
      setTimeout(() => {
        this.resizeTitle();
        if (!this.taskId()) {
          element.focus();
        }
      });
    }
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.lightbox()) {
      this.lightbox.set(null);
      return;
    }
    void this.close();
  }

  private hydrate(task: Task): void {
    this.taskId.set(task.id);
    this.createdAt.set(task.createdAt);
    this.isDone.set(task.done);
    this.title.set(task.title);
    this.description.set(task.description);
    this.profileId.set(task.profileId);
    const due = toDueInputs(task.dueAt);
    this.dueDate.set(due.date);
    this.dueTime.set(due.time);
    this.subtasks.set(task.subtasks);
    this.images.set(task.images);
  }

  private applyTask(task: Task): void {
    this.subtasks.set(task.subtasks);
    this.images.set(task.images);
    if (!this.taskId()) {
      this.taskId.set(task.id);
      this.createdAt.set(task.createdAt);
    }
  }

  private markDirty(): void {
    this.dirty.set(true);
    this.savedAt.set(null);
    if (this.saveTimer) {
      clearTimeout(this.saveTimer);
    }
    this.saveTimer = setTimeout(() => void this.persist(), 650);
  }

  private clearSaveTimer(): void {
    if (this.saveTimer) {
      clearTimeout(this.saveTimer);
      this.saveTimer = null;
    }
  }

  /** Persists title/description/due date. Creates the task on first save. */
  persist(): Promise<Task | null> {
    this.clearSaveTimer();
    const title = this.title().trim();
    if (!title) {
      return Promise.resolve(null);
    }
    const input = { title, description: this.description(), dueAt: this.dueAt(), profileId: this.profileId() };
    this.dirty.set(false);
    this.saving.set(true);

    const run = async (): Promise<Task | null> => {
      try {
        let updated: Task;
        const currentId = this.taskId();
        if (currentId) {
          updated = await this.store.updateTask(currentId, input);
        } else {
          updated = await this.store.createTask(input);
          this.taskId.set(updated.id);
          this.createdAt.set(updated.createdAt);
        }
        this.savedAt.set(Date.now());
        return updated;
      } catch {
        this.dirty.set(true);
        return null;
      } finally {
        this.saving.set(false);
      }
    };

    this.saveQueue = this.saveQueue.then(run, run);
    return this.saveQueue;
  }

  private async ensureSaved(): Promise<string | null> {
    if (!this.title().trim()) {
      this.store.notify('Donne d’abord un titre à la tâche.', 'error');
      return null;
    }
    if (this.taskId() && !this.dirty()) {
      return this.taskId();
    }
    const task = await this.persist();
    return task?.id ?? this.taskId();
  }

  onTitle(value: string): void {
    this.title.set(value);
    this.markDirty();
    this.resizeTitle();
  }

  /** Keeps the one-line-looking title usable on WebViews without field-sizing. */
  private resizeTitle(): void {
    const element = this.titleInput()?.nativeElement;
    if (!element) {
      return;
    }
    element.style.height = 'auto';
    element.style.height = `${element.scrollHeight}px`;
  }

  onTitleKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      void this.persist();
    }
  }

  onDescription(value: string): void {
    this.description.set(value);
    this.markDirty();
  }

  onDate(value: string): void {
    this.dueDate.set(value);
    if (!value) {
      this.dueTime.set('');
    }
    this.markDirty();
  }

  onTime(value: string): void {
    this.dueTime.set(value);
    this.markDirty();
  }

  onProfileChange(value: string): void {
    this.profileId.set(value);
    this.markDirty();
  }

  clearDue(): void {
    this.dueDate.set('');
    this.dueTime.set('');
    this.markDirty();
  }

  async close(): Promise<void> {
    this.clearSaveTimer();
    if (this.dirty() && this.title().trim()) {
      await this.persist();
    }
    this.closed.emit();
  }

  async toggleDone(): Promise<void> {
    const id = await this.ensureSaved();
    if (!id) {
      return;
    }
    try {
      await this.store.setTaskDone(id, !this.isDone());
      this.closed.emit();
    } catch {
      // The store already surfaced the error.
    }
  }

  async defer(): Promise<void> {
    const id = await this.ensureSaved();
    if (!id) {
      return;
    }
    const task = this.store.tasks().find((candidate) => candidate.id === id);
    if (!task) {
      return;
    }
    try {
      await this.store.deferTask(task);
      this.closed.emit();
    } catch {
      // The store already surfaced the error.
    }
  }

  async deleteTask(): Promise<void> {
    const id = this.taskId();
    if (!id) {
      this.closed.emit();
      return;
    }
    try {
      await this.store.deleteTask(id);
      this.closed.emit();
    } catch {
      this.confirmDelete.set(false);
    }
  }

  requestDelete(): void {
    this.confirmDelete.set(true);
    if (this.confirmTimer) {
      clearTimeout(this.confirmTimer);
    }
    this.confirmTimer = setTimeout(() => this.confirmDelete.set(false), 5000);
  }

  /* ------------------------------ subtasks ----------------------------- */

  async addSubtask(event: Event): Promise<void> {
    event.preventDefault();
    const title = this.addSubtaskTitle().trim();
    if (!title) {
      return;
    }
    const id = await this.ensureSaved();
    if (!id) {
      return;
    }
    try {
      const task = await this.store.addSubtask(id, title);
      this.applyTask(task);
      this.addSubtaskTitle.set('');
    } catch {
      // handled by the store
    }
  }

  async toggleSubtask(subtask: Subtask): Promise<void> {
    try {
      const task = await this.store.setSubtaskDone(subtask.id, !subtask.done);
      this.applyTask(task);
    } catch {
      // handled by the store
    }
  }

  async renameSubtask(subtask: Subtask, event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const value = input.value.trim();
    if (!value || value === subtask.title) {
      input.value = subtask.title;
      return;
    }
    try {
      const task = await this.store.updateSubtask(subtask.id, value);
      this.applyTask(task);
    } catch {
      input.value = subtask.title;
    }
  }

  async deleteSubtask(subtask: Subtask): Promise<void> {
    try {
      const task = await this.store.deleteSubtask(subtask.id);
      this.applyTask(task);
    } catch {
      // handled by the store
    }
  }

  /* ----------------------- réordonnancement des sous-tâches ------------------- */

  onSubtaskDragStart(event: DragEvent, subtask: Subtask): void {
    this.draggedSubtaskId.set(subtask.id);
    this.subtaskDrop.set(null);
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move';
      event.dataTransfer.setData('text/plain', subtask.id);
      const row = (event.target as HTMLElement | null)?.closest('.sub-row');
      if (row) {
        event.dataTransfer.setDragImage(row, 14, 18);
      }
    }
  }

  onSubtaskDragOver(event: DragEvent, subtask: Subtask): void {
    const dragged = this.draggedSubtaskId();
    if (!dragged) {
      return;
    }
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move';
    }
    if (dragged === subtask.id) {
      this.subtaskDrop.set(null);
      return;
    }
    const row = (event.target as HTMLElement | null)?.closest('.sub-row');
    if (!row) {
      return;
    }
    const rect = row.getBoundingClientRect();
    this.subtaskDrop.set({ id: subtask.id, after: event.clientY > rect.top + rect.height / 2 });
  }

  onSubtaskDrop(event: DragEvent, subtask: Subtask): void {
    const dragged = this.draggedSubtaskId();
    const target = this.subtaskDrop();
    this.resetSubtaskDrag();
    if (!dragged || dragged === subtask.id) {
      return;
    }
    event.preventDefault();
    const targetId = target?.id ?? subtask.id;
    const after = target?.after ?? false;
    const ids = this.subtasks()
      .map((item) => item.id)
      .filter((id) => id !== dragged);
    const targetIndex = ids.indexOf(targetId);
    if (targetIndex < 0) {
      return;
    }
    ids.splice(targetIndex + (after ? 1 : 0), 0, dragged);
    const ordered = ids
      .map((id) => this.subtasks().find((item) => item.id === id))
      .filter((item): item is Subtask => item !== undefined);
    this.subtasks.set(ordered);
    const taskId = this.taskId();
    if (!taskId) {
      return;
    }
    void this.store
      .reorderSubtasks(taskId, ids)
      .then((task) => this.applyTask(task))
      .catch(() => undefined);
  }

  onSubtaskDragEnd(): void {
    this.resetSubtaskDrag();
  }

  subtaskDropPosition(id: string): 'before' | 'after' | null {
    const target = this.subtaskDrop();
    if (!target || target.id !== id) {
      return null;
    }
    return target.after ? 'after' : 'before';
  }

  private resetSubtaskDrag(): void {
    this.draggedSubtaskId.set(null);
    this.subtaskDrop.set(null);
  }

  /* -------------------------------- images ----------------------------- */

  async addImage(): Promise<void> {
    const id = await this.ensureSaved();
    if (!id) {
      return;
    }
    try {
      const task = await this.store.addImage(id);
      this.applyTask(task);
    } catch {
      // handled by the store
    }
  }

  async deleteImage(image: TaskImage, event: Event): Promise<void> {
    event.stopPropagation();
    try {
      const task = await this.store.deleteImage(image.id);
      this.applyTask(task);
      if (this.lightbox()) {
        this.lightbox.set(null);
      }
    } catch {
      // handled by the store
    }
  }

  /* --------------------------------- Claude ----------------------------- */

  async breakdown(): Promise<void> {
    this.aiError.set(null);
    if (!this.title().trim()) {
      this.store.notify('Donne d’abord un titre à la tâche.', 'error');
      return;
    }
    const id = await this.ensureSaved();
    if (!id) {
      return;
    }
    this.breakingDown.set(true);
    try {
      const suggestions = await this.store.breakDown(id);
      const existing = new Set(this.subtasks().map((subtask) => subtask.title.toLowerCase()));
      const fresh = suggestions.filter((suggestion) => !existing.has(suggestion.toLowerCase()));
      if (fresh.length === 0) {
        this.aiError.set('Claude n’a rien proposé de nouveau. Essaie avec une description plus détaillée.');
      }
      this.suggestions.set(fresh);
    } catch (error) {
      this.aiError.set(errorMessage(error));
    } finally {
      this.breakingDown.set(false);
    }
  }

  dismissSuggestions(): void {
    this.suggestions.set([]);
  }

  editSuggestion(index: number, value: string): void {
    this.suggestions.update((items) => items.map((item, i) => (i === index ? value : item)));
  }

  async addSuggestion(index: number): Promise<void> {
    const title = (this.suggestions()[index] ?? '').trim();
    if (!title) {
      this.suggestions.update((items) => items.filter((_, i) => i !== index));
      return;
    }
    const id = await this.ensureSaved();
    if (!id) {
      return;
    }
    try {
      const task = await this.store.addSubtask(id, title);
      this.applyTask(task);
      this.suggestions.update((items) => items.filter((_, i) => i !== index));
    } catch {
      // handled by the store
    }
  }

  async addAllSuggestions(): Promise<void> {
    const id = await this.ensureSaved();
    if (!id || this.suggestions().some((item) => item.trim()) === false) {
      return;
    }
    const titles = this.suggestions()
      .map((item) => item.trim())
      .filter(Boolean);
    try {
      const task = await this.store.addSubtasks(id, titles);
      this.applyTask(task);
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
