import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { DecimalPipe } from '@angular/common';

import { Payment } from './models';

/** Panel de pagos registrados (el estado de negocio). */
@Component({
  selector: 'app-payments-panel',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DecimalPipe],
  template: `
    <section class="flex h-full flex-col rounded-xl border border-slate-200 bg-white shadow-sm">
      <header class="flex items-center justify-between border-b border-slate-100 px-4 py-3">
        <h2 class="font-semibold text-slate-800">💳 Pagos</h2>
        <span class="rounded-full bg-slate-100 px-2 py-0.5 text-sm text-slate-600">
          {{ payments().length }}
        </span>
      </header>
      <div class="flex-1 overflow-auto">
        @if (payments().length === 0) {
          <p class="p-4 text-sm text-slate-400">Aún no hay pagos.</p>
        } @else {
          <ul class="divide-y divide-slate-100">
            @for (p of payments(); track p.id) {
              <li class="flex items-center justify-between px-4 py-2 text-sm">
                <div class="min-w-0">
                  <p class="truncate font-medium text-slate-800">{{ p.customer }}</p>
                  <p class="truncate font-mono text-xs text-slate-400">{{ p.id.slice(0, 8) }}</p>
                </div>
                <div class="text-right">
                  <p class="font-semibold text-slate-900">
                    {{ p.amount | number: '1.2-2' }} {{ p.currency }}
                  </p>
                  <p class="text-xs text-emerald-600">{{ p.status }}</p>
                </div>
              </li>
            }
          </ul>
        }
      </div>
    </section>
  `,
})
export class PaymentsPanel {
  readonly payments = input.required<Payment[]>();
}
