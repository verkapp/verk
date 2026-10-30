import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';

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

  go(view: ViewName): void {
    this.navigate.emit(view);
  }
}
