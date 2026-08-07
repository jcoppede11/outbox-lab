import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { DatePipe } from '@angular/common';

import { OutboxRow, OutboxStatus } from './models';

/** Panel de la tabla outbox, con el estado de cada evento. */
@Component({
  selector: 'app-outbox-panel',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DatePipe],
  template: `
    <section class="flex h-full flex-col rounded-xl border border-slate-200 bg-white shadow-sm">
      <header class="flex items-center justify-between border-b border-slate-100 px-4 py-3">
        <h2 class="font-semibold text-slate-800">📤 Outbox</h2>
        <span class="rounded-full bg-slate-100 px-2 py-0.5 text-sm text-slate-600">
          {{ rows().length }}
        </span>
      </header>
      <div class="flex-1 overflow-auto">
        @if (rows().length === 0) {
          <p class="p-4 text-sm text-slate-400">Sin eventos.</p>
        } @else {
          <ul class="divide-y divide-slate-100">
            @for (row of rows(); track row.id) {
              <li class="flex items-center justify-between px-4 py-2 text-sm">
                <div class="min-w-0">
                  <p class="font-medium text-slate-800">#{{ row.id }} · {{ row.payload.type }}</p>
                  <p class="truncate font-mono text-xs text-slate-400">
                    {{ row.payload.payment_id.slice(0, 8) }}
                    @if (row.published_at) {
                      · {{ row.published_at | date: 'HH:mm:ss' }}
                    }
                  </p>
                </div>
                <span class="rounded-full border px-2 py-0.5 text-xs font-medium" [class]="badge(row.status)">
                  {{ label(row.status) }}
                </span>
              </li>
            }
          </ul>
        }
      </div>
    </section>
  `,
})
export class OutboxPanel {
  readonly rows = input.required<OutboxRow[]>();

  protected badge(status: OutboxStatus): string {
    switch (status) {
      case 'sent':
        return 'border-emerald-300 bg-emerald-50 text-emerald-700';
      case 'failed':
        return 'border-rose-300 bg-rose-50 text-rose-700';
      default:
        return 'border-amber-300 bg-amber-50 text-amber-700';
    }
  }

  protected label(status: OutboxStatus): string {
    switch (status) {
      case 'sent':
        return 'enviado';
      case 'failed':
        return 'fallido';
      default:
        return 'pendiente';
    }
  }
}
