import { ChangeDetectionStrategy, Component, inject, input, output, signal } from '@angular/core';

import { Store } from '../store';
import { ViewName } from '../models';
import { IconComponent } from '../ui/icon.component';

@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './sidebar.component.html',
  styleUrl: './sidebar.component.css',
})
export class SidebarComponent {
  readonly store = inject(Store);
  readonly view = input.required<ViewName>();
  readonly collapsed = input(false);
  readonly navigate = output<ViewName>();
  readonly toggleCollapsed = output<void>();

  readonly creatingProfile = signal(false);
  readonly profileDraft = signal('');

  go(view: ViewName): void {
    this.navigate.emit(view);
  }

  selectProfile(id: string): void {
    this.store.selectProfile(id);
    this.navigate.emit('today');
  }

  startCreateProfile(): void {
    this.profileDraft.set('');
    this.creatingProfile.set(true);
    setTimeout(() => {
      const input = document.querySelector<HTMLInputElement>('.profile-form input');
      input?.focus();
    });
  }

  cancelCreateProfile(): void {
    this.creatingProfile.set(false);
    this.profileDraft.set('');
  }

  async createProfile(event: Event): Promise<void> {
    event.preventDefault();
    const name = this.profileDraft().trim();
    if (!name) {
      return;
    }
    const profile = await this.store.createProfile(name);
    if (profile) {
      this.cancelCreateProfile();
      this.navigate.emit('today');
    }
  }
}
