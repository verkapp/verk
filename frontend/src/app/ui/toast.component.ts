import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';

import { Store } from '../store';
import { IconComponent } from '../ui/icon.component';

@Component({
  selector: 'app-toast',
  standalone: true,
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '[class.raised]': 'raised()',
  },
  template: `
    @if (store.toast(); as toast) {
      <div class="toast" [class.error]="toast.tone === 'error'" [class.accent]="toast.tone === 'accent'" role="status">
        <span class="toast-icon">
          @if (toast.tone === 'error') {
            <app-icon name="alert" [size]="15" />
          } @else {
            <app-icon name="check" [size]="14" [strokeWidth]="2.5" />
          }
        </span>
        <span class="toast-text">{{ toast.text }}</span>
        <button class="toast-close" type="button" (click)="store.dismissToast()" aria-label="Fermer">
          <app-icon name="close" [size]="13" />
        </button>
      </div>
    }
  `,
  styles: [
    `
      :host {
        position: fixed;
        left: 50%;
        bottom: 26px;
        transform: translateX(-50%);
        z-index: 120;
        pointer-events: none;
        transition: bottom 0.2s ease;
      }

      :host(.raised) {
        bottom: 88px;
      }

      .toast {
        display: flex;
        align-items: center;
        gap: 9px;
        max-width: min(520px, calc(100vw - 48px));
        padding: 9px 10px 9px 9px;
        border-radius: 12px;
        background: var(--surface);
        border: 1px solid var(--line-strong);
        box-shadow: 0 8px 20px -14px rgba(35, 30, 25, 0.22);
        animation: toast-in 0.25s cubic-bezier(0.2, 0.8, 0.2, 1) both;
        pointer-events: auto;
      }

      .toast-icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: none;
        width: 24px;
        height: 24px;
        border-radius: 7px;
        background: var(--success-soft);
        color: var(--success);
      }

      .toast.accent .toast-icon {
        background: var(--accent-soft);
        color: var(--accent-strong);
      }

      .toast.error .toast-icon {
        background: var(--danger-soft);
        color: var(--danger);
      }

      .toast-text {
        flex: 1;
        font-size: 12.4px;
        font-weight: 500;
        line-height: 1.4;
        color: var(--text-soft);
      }

      .toast-close {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: none;
        width: 22px;
        height: 22px;
        border-radius: 6px;
        color: var(--muted-2);
      }

      .toast-close:hover {
        background: var(--surface-muted);
        color: var(--text);
      }

      @keyframes toast-in {
        from {
          opacity: 0;
          transform: translateY(8px) scale(0.98);
        }
        to {
          opacity: 1;
          transform: translateY(0) scale(1);
        }
      }
    `,
  ],
})
export class ToastComponent {
  readonly store = inject(Store);
  /** Lifts the toast above the drawer footer while it is open. */
  readonly raised = input(false);
}
