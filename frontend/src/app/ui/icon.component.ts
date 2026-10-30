import { ChangeDetectionStrategy, Component, input } from '@angular/core';

export type IconName =
  | 'today'
  | 'tasks'
  | 'sparkles'
  | 'history'
  | 'settings'
  | 'plus'
  | 'filter'
  | 'calendar'
  | 'clock'
  | 'subtasks'
  | 'image'
  | 'trash'
  | 'close'
  | 'check'
  | 'chevron-left'
  | 'chevron-right'
  | 'chevron-down'
  | 'arrow-right'
  | 'wand'
  | 'alert'
  | 'folder'
  | 'sidebar'
  | 'refresh'
  | 'check-circle'
  | 'leaf';

@Component({
  selector: 'app-icon',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <svg
      [attr.width]="size()"
      [attr.height]="size()"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      [attr.stroke-width]="strokeWidth()"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      @switch (name()) {
        @case ('today') {
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
        }
        @case ('tasks') {
          <rect x="8" y="2" width="8" height="4" rx="1" />
          <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
          <path d="M9 12h6M9 16h4" />
        }
        @case ('sparkles') {
          <path d="M12 3l1.7 4.6L18 9.3l-4.3 1.7L12 15.6l-1.7-4.6L6 9.3l4.3-1.7L12 3z" />
          <path d="M19 15l.8 2.2L22 18l-2.2.8L19 21l-.8-2.2L16 18l2.2-.8L19 15z" />
          <path d="M5 14l.6 1.6L7 16l-1.4.6L5 18l-.6-1.4L3 16l1.4-.4L5 14z" />
        }
        @case ('history') {
          <path d="M3 12a9 9 0 1 0 3-6.7L3 8" />
          <path d="M3 3v5h5" />
          <path d="M12 7v5l3 2" />
        }
        @case ('settings') {
          <path d="M4 6h9M17 6h3" />
          <circle cx="15" cy="6" r="2" />
          <path d="M4 12h3M11 12h9" />
          <circle cx="9" cy="12" r="2" />
          <path d="M4 18h9M17 18h3" />
          <circle cx="15" cy="18" r="2" />
        }
        @case ('plus') {
          <path d="M12 5v14M5 12h14" />
        }
        @case ('filter') {
          <path d="M4 5h16l-6.4 7.3V19l-3.2 1.5v-8.2L4 5z" />
        }
        @case ('calendar') {
          <rect x="3" y="5" width="18" height="16" rx="3" />
          <path d="M8 3v4M16 3v4M3 10h18" />
        }
        @case ('clock') {
          <circle cx="12" cy="12" r="9" />
          <path d="M12 7v5l3 2" />
        }
        @case ('subtasks') {
          <path d="M3 6l2 2 3.5-4M3 13l2 2 3.5-4M3 20l2 2 3.5-4" />
          <path d="M12 7h9M12 14h9M12 21h9" />
        }
        @case ('image') {
          <rect x="3" y="4" width="18" height="16" rx="3" />
          <circle cx="9" cy="10" r="1.6" />
          <path d="M4 18l5-4 4 3 2.5-2L20 19" />
        }
        @case ('trash') {
          <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2M6 7l1 13a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1l1-13" />
          <path d="M10 11v6M14 11v6" />
        }
        @case ('close') {
          <path d="M6 6l12 12M18 6L6 18" />
        }
        @case ('check') {
          <path d="M5 12.5l4.5 4.5L19 7.5" />
        }
        @case ('check-circle') {
          <circle cx="12" cy="12" r="9" />
          <path d="M8.5 12.5l2.5 2.5 4.5-5" />
        }
        @case ('chevron-left') {
          <path d="M15 5l-7 7 7 7" />
        }
        @case ('chevron-right') {
          <path d="M9 5l7 7-7 7" />
        }
        @case ('chevron-down') {
          <path d="M5 9l7 7 7-7" />
        }
        @case ('arrow-right') {
          <path d="M4 12h16M14 6l6 6-6 6" />
        }
        @case ('wand') {
          <path d="M15 4V2M15 16v-2M8 9h2M20 9h2M17.8 11.8l1.4 1.4M17.8 6.2l1.4-1.4M12.5 6.2l-1.4-1.4M3 21l9-9" />
        }
        @case ('alert') {
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8v5M12 16.5v.01" />
        }
        @case ('folder') {
          <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" />
        }
        @case ('sidebar') {
          <rect x="3" y="4" width="18" height="16" rx="3" />
          <path d="M9.5 4v16" />
          <path d="M15.5 10l-2 2 2 2" />
        }
        @case ('refresh') {
          <path d="M20 11a8 8 0 1 0-2 5.3" />
          <path d="M20 4v7h-7" />
        }
        @case ('leaf') {
          <path d="M20 4c0 9-5.5 14-12 14a5 5 0 0 1-4-8c3-5 9-6 16-6z" />
          <path d="M4 21c2-6 6-9 12-11" />
        }
      }
    </svg>
  `,
  styles: [
    `
      :host {
        display: inline-flex;
        flex: none;
      }
      svg {
        display: block;
      }
    `,
  ],
})
export class IconComponent {
  readonly name = input.required<IconName>();
  readonly size = input(18);
  readonly strokeWidth = input(1.75);
}
