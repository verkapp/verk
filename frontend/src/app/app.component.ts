import { ChangeDetectionStrategy, Component, ElementRef, HostListener, computed, inject, OnInit, signal, viewChild } from '@angular/core';

import { endOfToday } from './dates';
import { CoachComponent } from './coach/coach.component';
import { HistoryComponent } from './history/history.component';
import { SettingsComponent } from './settings/settings.component';
import { SidebarComponent } from './sidebar/sidebar.component';
import { TaskDrawerComponent } from './tasks/task-drawer.component';
import { TaskListComponent } from './tasks/task-list.component';
import { Task, ViewName } from './models';
import { Store } from './store';
import { isMacDesktop, onRuntimeEvent } from './api';
import { IconComponent } from './ui/icon.component';
import { ToastComponent } from './ui/toast.component';

interface DrawerState {
  task: Task | null;
  presetDue: number | null;
  key: number;
}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    SidebarComponent,
    TaskListComponent,
    HistoryComponent,
    CoachComponent,
    SettingsComponent,
    TaskDrawerComponent,
    ToastComponent,
    IconComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './app.component.html',
  styleUrl: './app.component.css',
})
export class AppComponent implements OnInit {
  readonly store = inject(Store);
  readonly view = signal<ViewName>('today');
  readonly collapsed = signal(false);
  readonly drawer = signal<DrawerState | null>(null);
  readonly main = viewChild<ElementRef<HTMLElement>>('main');
  readonly drawerOpen = computed(() => this.drawer() !== null);

  ngOnInit(): void {
    try {
      this.collapsed.set(localStorage.getItem('todo.sidebar') === 'collapsed');
    } catch {
      // ignore storage restrictions
    }
    if (isMacDesktop()) {
      document.documentElement.classList.add('mac-titlebar');
    }
    void this.store.init();
    onRuntimeEvent<string>('menu:action', (action) => this.onMenuAction(action));
  }

  /** Actions sent by the native menu bar and the tray icon. */
  private onMenuAction(action: string): void {
    if (action === 'new-task') {
      if (!this.drawer()) {
        this.newTask();
      }
      return;
    }
    if (action === 'refresh') {
      void this.store.refreshNow();
      return;
    }
    const views: Record<string, ViewName> = {
      'view:today': 'today',
      'view:all': 'all',
      'view:coach': 'coach',
      'view:history': 'history',
      'view:settings': 'settings',
    };
    const view = views[action];
    if (view) {
      this.drawer.set(null);
      this.navigate(view);
    }
  }

  @HostListener('document:keydown', ['$event'])
  onShortcut(event: KeyboardEvent): void {
    const mod = event.metaKey || event.ctrlKey;
    if (mod && event.key.toLowerCase() === 'n' && !this.drawer()) {
      event.preventDefault();
      this.newTask();
    }
  }

  navigate(view: ViewName): void {
    if (view === 'coach' && !this.store.coachActive()) {
      return;
    }
    this.view.set(view);
    const element = this.main()?.nativeElement;
    if (element) {
      element.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  toggleCollapsed(): void {
    const next = !this.collapsed();
    this.collapsed.set(next);
    try {
      localStorage.setItem('todo.sidebar', next ? 'collapsed' : 'open');
    } catch {
      // ignore storage restrictions
    }
  }

  openTask(id: string): void {
    const task = this.store.tasks().find((candidate) => candidate.id === id);
    if (!task) {
      return;
    }
    this.drawer.set({ task, presetDue: null, key: Date.now() });
  }

  newTask(): void {
    const presetDue = this.view() === 'today' ? endOfToday() : null;
    this.drawer.set({ task: null, presetDue, key: Date.now() });
  }

  closeDrawer(): void {
    this.drawer.set(null);
  }

  openCoach(): void {
    this.navigate('coach');
  }

  openSettings(): void {
    this.navigate('settings');
  }

  retry(): void {
    window.location.reload();
  }
}
